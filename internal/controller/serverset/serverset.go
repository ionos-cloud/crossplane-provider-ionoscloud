/*
Copyright 2022 The Crossplane Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package serverset

import (
	"context"
	"fmt"

	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/pkg/errors"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	apisv1alpha1 "github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute/server"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"
)

const (
	errNotServerSet = "managed resource is not a ServerSet custom resource"
	errTrackPCUsage = "cannot track ProviderConfig usage"
)

// A NoOpService does nothing.
type NoOpService struct{}

// Setup adds a controller that reconciles ServerSet managed resources.
func Setup(mgr ctrl.Manager, l logging.Logger, rl workqueue.RateLimiter, opts *utils.ConfigurationOptions) error {
	name := managed.ControllerName(v1alpha1.ServerSetGroupKind)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(controller.Options{
			RateLimiter: ratelimiter.NewController(),
		}).
		For(&v1alpha1.ServerSet{}).
		Complete(managed.NewReconciler(mgr,
			resource.ManagedKind(v1alpha1.ServerSetGroupVersionKind),
			managed.WithExternalConnecter(&connectorServerSet{
				kube:                 mgr.GetClient(),
				usage:                resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
				log:                  l,
				isUniqueNamesEnabled: opts.GetIsUniqueNamesEnabled()}),
			managed.WithReferenceResolver(managed.NewAPISimpleReferenceResolver(mgr.GetClient())),
			managed.WithInitializers(),
			managed.WithPollInterval(opts.GetPollInterval()),
			managed.WithTimeout(opts.GetTimeout()),
			managed.WithCreationGracePeriod(opts.GetCreationGracePeriod()),
			managed.WithLogger(l.WithValues("controller", name)),
			managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name)))))
}

// A connectorServerSet is expected to produce an ExternalClient when its Connect method
// is called.
type connectorServerSet struct {
	kube                 client.Client
	usage                resource.Tracker
	log                  logging.Logger
	isUniqueNamesEnabled bool
}

// Connect typically produces an ExternalClient by:
// 1. Tracking that the managed resource is using a ProviderConfig.
// 2. Getting the managed resource's ProviderConfig.
// 3. Getting the credentials specified by the ProviderConfig.
// 4. Using the credentials to form a client.
func (c *connectorServerSet) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	_, ok := mg.(*v1alpha1.ServerSet)
	if !ok {
		return nil, errors.New(errNotServerSet)
	}

	if err := c.usage.Track(ctx, mg); err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	svc, err := clients.ConnectForCRD(ctx, mg, c.kube, c.usage)
	return &externalServerSet{
		service:              &server.APIClient{IonosServices: svc},
		log:                  c.log,
		isUniqueNamesEnabled: c.isUniqueNamesEnabled}, err
}

// An ExternalClient observes, then either creates, updates, or deletes an
// externalServerSet resource to ensure it reflects the managed resource's desired state.
type externalServerSet struct {
	// A 'client' used to connect to the externalServer resource API. In practice this
	// would be something like an IONOS Cloud SDK client.
	service              server.Client
	log                  logging.Logger
	isUniqueNamesEnabled bool
}

func (c *externalServerSet) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.ServerSet)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotServerSet)
	}

	// These fmt statements should be removed in the real implementation.
	fmt.Printf("Observing: %+v", cr)

	return managed.ExternalObservation{
		// Return false when the externalServerSet resource does not exist. This lets
		// the managed resource reconciler know that it needs to call Create to
		// (re)create the resource, or that it has successfully been deleted.
		ResourceExists: true,

		// Return false when the externalServerSet resource exists, but it not up to date
		// with the desired managed resource state. This lets the managed
		// resource reconciler know that it needs to call Update.
		ResourceUpToDate: true,

		// Return any details that may be required to connect to the externalServerSet
		// resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *externalServerSet) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.ServerSet)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotServerSet)
	}

	fmt.Printf("Creating: %+v", cr)

	return managed.ExternalCreation{
		// Optionally return any details that may be required to connect to the
		// externalServerSet resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *externalServerSet) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.ServerSet)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotServerSet)
	}

	fmt.Printf("Updating: %+v", cr)

	return managed.ExternalUpdate{
		// Optionally return any details that may be required to connect to the
		// externalServerSet resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *externalServerSet) Delete(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*v1alpha1.ServerSet)
	if !ok {
		return errors.New(errNotServerSet)
	}

	fmt.Printf("Deleting: %+v", cr)

	return nil
}
