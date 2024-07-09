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

package postgresdatabase

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
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

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/dbaas/postgres/v1alpha1"
	apisv1alpha1 "github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/dbaas/postgrescluster"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"
)

const errNotDatabase = "managed resource is not a Postgres database custom resource"

// Setup adds a controller that reconciles Database managed resources.
func Setup(mgr ctrl.Manager, opts *utils.ConfigurationOptions) error {
	name := managed.ControllerName(v1alpha1.PostgresDatabaseGroupKind)
	logger := opts.CtrlOpts.Logger

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(controller.Options{
			RateLimiter: ratelimiter.NewController(),
		}).
		For(&v1alpha1.PostgresDatabase{}).
		Complete(managed.NewReconciler(mgr,
			resource.ManagedKind(v1alpha1.PostgresDatabaseGroupVersionKind),
			managed.WithExternalConnecter(&connectorDatabase{
				kube:  mgr.GetClient(),
				usage: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
				log:   logger}),
			managed.WithReferenceResolver(managed.NewAPISimpleReferenceResolver(mgr.GetClient())),
			managed.WithInitializers(),
			managed.WithPollInterval(opts.GetPollInterval()),
			managed.WithTimeout(opts.GetTimeout()),
			managed.WithCreationGracePeriod(opts.GetCreationGracePeriod()),
			managed.WithLogger(logger.WithValues("controller", name)),
			managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name)))))
}

// A connectorDatabase is expected to produce an ExternalClient when its Connect method
// is called.
type connectorDatabase struct {
	kube  client.Client
	usage resource.Tracker
	log   logging.Logger
}

// Connect typically produces an ExternalClient by:
// 1. Tracking that the managed resource is using a ProviderConfig.
// 2. Getting the managed resource's ProviderConfig.
// 3. Getting the credentials specified by the ProviderConfig.
// 4. Using the credentials to form a client.
func (c *connectorDatabase) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	_, ok := mg.(*v1alpha1.PostgresDatabase)
	if !ok {
		return nil, errors.New(errNotDatabase)
	}
	svc, err := clients.ConnectForCRD(ctx, mg, c.kube, c.usage)
	return &externalDatabase{
		service: &postgrescluster.ClusterAPIClient{IonosServices: svc},
		log:     c.log,
		client:  c.kube}, err

}

// An ExternalClient observes, then either creates, updates, or deletes an
// externalDatabase resource to ensure it reflects the managed resource's desired state.
type externalDatabase struct {
	// A 'client' used to connect to the externalDatabase resource API. In practice this
	// would be something like an IONOS Cloud SDK client.
	service postgrescluster.ClusterClient
	client  client.Client
	log     logging.Logger
}

// Observe checks whether the specified externalDatabase resource exists
func (u *externalDatabase) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) { // nolint:gocyclo
	cr, ok := mg.(*v1alpha1.PostgresDatabase)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotDatabase)
	}

	// External Name of the CR is the DBaaS Postgres Database ID
	if meta.GetExternalName(cr) == "" {
		return managed.ExternalObservation{}, nil
	}
	_, resp, err := u.service.GetDatabase(ctx, cr.Spec.ForProvider.ClusterCfg.ClusterID, meta.GetExternalName(cr))
	if err != nil {
		if resp.HttpNotFound() {
			return managed.ExternalObservation{}, nil
		}
		return managed.ExternalObservation{}, fmt.Errorf("failed to get postgres database by name %s : %w", meta.GetExternalName(cr), err)
	}
	cr.Status.AtProvider.DatabaseID = meta.GetExternalName(cr)
	cr.SetConditions(xpv1.Available())
	return managed.ExternalObservation{
		ResourceExists:    true,
		ResourceUpToDate:  true,
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

// Create creates the database
func (u *externalDatabase) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) { // nolint: gocyclo
	cr, ok := mg.(*v1alpha1.PostgresDatabase)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotDatabase)
	}
	cr.SetConditions(xpv1.Creating())

	clusterID := cr.Spec.ForProvider.ClusterCfg.ClusterID
	newInstance, apiResponse, err := u.service.CreateDatabase(ctx, clusterID, *cr)
	creation := managed.ExternalCreation{ConnectionDetails: managed.ConnectionDetails{}}
	if err != nil {
		retErr := fmt.Errorf("failed to create postgres database: %w", err)
		if apiResponse != nil && apiResponse.Response != nil {
			retErr = fmt.Errorf("%w API Response Status: %v", retErr, apiResponse.Status)
		}
		return creation, retErr
	}

	meta.SetExternalName(cr, *newInstance.Properties.Name)
	return creation, nil
}

// Update updates the database
func (u *externalDatabase) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	_, ok := mg.(*v1alpha1.PostgresDatabase)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotDatabase)
	}

	return managed.ExternalUpdate{}, nil
}

// Delete deletes the database
func (u *externalDatabase) Delete(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*v1alpha1.PostgresDatabase)
	if !ok {
		return errors.New(errNotDatabase)
	}

	cr.SetConditions(xpv1.Deleting())
	apiResponse, err := u.service.DeleteDatabase(ctx, cr.Spec.ForProvider.ClusterCfg.ClusterID, meta.GetExternalName(cr))
	if err != nil {
		if apiResponse.HttpNotFound() {
			return nil
		}
		return fmt.Errorf("failed to delete postgres database. error: %w", err)
	}
	return nil
}
