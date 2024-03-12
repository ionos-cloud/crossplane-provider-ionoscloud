/*
Copyright 2020 The Crossplane Authors.

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

package flowlog

import (
	"context"

	"github.com/pkg/errors"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/nlb/v1alpha1"
	apisv1alpha1 "github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/nlb/flowlog"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"
)

const errNotFlowLog = "managed resource is not a FlowLog"

// Setup adds a controller that reconciles FlowLog managed resources.
func Setup(mgr ctrl.Manager, l logging.Logger, rl workqueue.RateLimiter, opts *utils.ConfigurationOptions) error {
	name := managed.ControllerName(v1alpha1.FlowLogGroupKind)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(controller.Options{
			RateLimiter: ratelimiter.NewController(),
		}).
		For(&v1alpha1.FlowLog{}).
		Complete(managed.NewReconciler(mgr,
			resource.ManagedKind(v1alpha1.FlowLogGroupVersionKind),
			managed.WithExternalConnecter(&connectorFlowLog{
				kube:                 mgr.GetClient(),
				usage:                resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
				log:                  l,
				isUniqueNamesEnabled: opts.GetIsUniqueNamesEnabled()}),
			managed.WithReferenceResolver(managed.NewAPISimpleReferenceResolver(mgr.GetClient())),
			managed.WithInitializers(),
			managed.WithCreationGracePeriod(opts.GetCreationGracePeriod()),
			managed.WithPollInterval(opts.GetPollInterval()),
			managed.WithTimeout(opts.GetTimeout()),
			managed.WithLogger(l.WithValues("controller", name)),
			managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name)))))
}

// A connectorFlowLog is expected to produce an ExternalClient when its Connect method
// is called.
type connectorFlowLog struct {
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
func (c *connectorFlowLog) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	_, ok := mg.(*v1alpha1.FlowLog)
	if !ok {
		return nil, errors.New(errNotFlowLog)
	}
	svc, err := clients.ConnectForCRD(ctx, mg, c.kube, c.usage)
	return &externalFlowLog{
		service:              &flowlog.APIClient{IonosServices: svc},
		log:                  c.log,
		isUniqueNamesEnabled: c.isUniqueNamesEnabled}, err
}

// An ExternalClient observes, then either creates, updates, or deletes an
// externalFlowLog resource to ensure it reflects the managed resource's desired state.
type externalFlowLog struct {
	// A 'client' used to connect to the externalFlowLog resource API. In practice this
	// would be something like an IONOS Cloud SDK client.
	service              flowlog.Client
	log                  logging.Logger
	isUniqueNamesEnabled bool
}

func (c *externalFlowLog) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) { // nolint:gocyclo
	cr, ok := mg.(*v1alpha1.FlowLog)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotFlowLog)
	}

	if meta.GetExternalName(cr) == "" {
		return managed.ExternalObservation{}, nil
	}

	return managed.ExternalObservation{}, nil
}

func (c *externalFlowLog) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) { // nolint: gocyclo
	cr, ok := mg.(*v1alpha1.FlowLog)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotFlowLog)
	}
	cr.SetConditions(xpv1.Creating())
	// Check external name in order to avoid duplicates,
	// since the creation requests take longer than other resources.
	if meta.GetExternalName(cr) != "" {
		return managed.ExternalCreation{}, nil
	}
	if cr.Status.AtProvider.State == compute.BUSY {
		return managed.ExternalCreation{}, nil
	}

	if c.isUniqueNamesEnabled {
	}
	return managed.ExternalCreation{}, nil
}

func (c *externalFlowLog) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.FlowLog)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotFlowLog)
	}
	if cr.Status.AtProvider.State == compute.BUSY {
		return managed.ExternalUpdate{}, nil
	}

	return managed.ExternalUpdate{}, nil
}

func (c *externalFlowLog) Delete(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*v1alpha1.FlowLog)
	if !ok {
		return errors.New(errNotFlowLog)
	}

	cr.SetConditions(xpv1.Deleting())
	if cr.Status.AtProvider.State == compute.DESTROYING || cr.Status.AtProvider.State == compute.BUSY {
		return nil
	}

	return nil
}
