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

package postgrescluster

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/google/go-cmp/cmp"

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

	ionoscloud "github.com/ionos-cloud/sdk-go-dbaas-postgres"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/dbaas/postgres/v1alpha1"
	apisv1alpha1 "github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/dbaas/postgrescluster"
)

const errNotCluster = "managed resource is not a Cluster custom resource"

// Setup adds a controller that reconciles Cluster managed resources.
func Setup(mgr ctrl.Manager, l logging.Logger, rl workqueue.RateLimiter, poll time.Duration, creationGracePeriod time.Duration) error {
	name := managed.ControllerName(v1alpha1.PostgresClusterGroupKind)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(controller.Options{
			RateLimiter: ratelimiter.NewDefaultManagedRateLimiter(rl),
		}).
		For(&v1alpha1.PostgresCluster{}).
		Complete(managed.NewReconciler(mgr,
			resource.ManagedKind(v1alpha1.PostgresClusterGroupVersionKind),
			managed.WithExternalConnecter(&connectorCluster{
				kube:  mgr.GetClient(),
				usage: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
				log:   l}),
			managed.WithReferenceResolver(managed.NewAPISimpleReferenceResolver(mgr.GetClient())),
			managed.WithInitializers(),
			managed.WithPollInterval(poll),
			managed.WithCreationGracePeriod(creationGracePeriod),
			managed.WithLogger(l.WithValues("controller", name)),
			managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name)))))
}

// A connectorCluster is expected to produce an ExternalClient when its Connect method
// is called.
type connectorCluster struct {
	kube  client.Client
	usage resource.Tracker
	log   logging.Logger
}

// Connect typically produces an ExternalClient by:
// 1. Tracking that the managed resource is using a ProviderConfig.
// 2. Getting the managed resource's ProviderConfig.
// 3. Getting the credentials specified by the ProviderConfig.
// 4. Using the credentials to form a client.
func (c *connectorCluster) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	_, ok := mg.(*v1alpha1.PostgresCluster)
	if !ok {
		return nil, errors.New(errNotCluster)
	}
	svc, err := clients.ConnectForCRD(ctx, mg, c.kube, c.usage)
	return &externalCluster{
		service: &postgrescluster.ClusterAPIClient{IonosServices: svc},
		log:     c.log}, err
}

// An ExternalClient observes, then either creates, updates, or deletes an
// externalCluster resource to ensure it reflects the managed resource's desired state.
type externalCluster struct {
	// A 'client' used to connect to the externalCluster resource API. In practice this
	// would be something like an IONOS Cloud SDK client.
	service postgrescluster.ClusterClient
	log     logging.Logger
}

func (c *externalCluster) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) { // nolint:gocyclo
	cr, ok := mg.(*v1alpha1.PostgresCluster)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotCluster)
	}

	// External Name of the CR is the DBaaS Postgres Cluster ID
	if meta.GetExternalName(cr) == "" {
		return managed.ExternalObservation{}, nil
	}
	observed, resp, err := c.service.GetCluster(ctx, meta.GetExternalName(cr))
	if err != nil {
		if resp != nil && resp.Response != nil && resp.StatusCode == http.StatusNotFound {
			return managed.ExternalObservation{}, nil
		}
		return managed.ExternalObservation{}, fmt.Errorf("failed to get postgres cluster by id. err: %w", err)
	}

	current := cr.Spec.ForProvider.DeepCopy()
	postgrescluster.LateInitializer(&cr.Spec.ForProvider, &observed)

	cr.Status.AtProvider.ClusterID = meta.GetExternalName(cr)
	cr.Status.AtProvider.State = string(clients.GetDBaaSResourceState(&observed))
	c.log.Debug(fmt.Sprintf("Observing state: %v", cr.Status.AtProvider.State))
	clients.UpdateCondition(cr, cr.Status.AtProvider.State)

	return managed.ExternalObservation{
		ResourceExists:          true,
		ResourceUpToDate:        postgrescluster.IsClusterUpToDate(cr, observed),
		ConnectionDetails:       managed.ConnectionDetails{},
		ResourceLateInitialized: !cmp.Equal(current, &cr.Spec.ForProvider),
	}, nil
}

func (c *externalCluster) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.PostgresCluster)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotCluster)
	}

	cr.SetConditions(xpv1.Creating())
	if cr.Status.AtProvider.State == string(ionoscloud.BUSY) {
		return managed.ExternalCreation{}, nil
	}
	instanceInput, err := postgrescluster.GenerateCreateClusterInput(cr)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	instance, apiResponse, err := c.service.CreateCluster(ctx, *instanceInput)
	creation := managed.ExternalCreation{ConnectionDetails: managed.ConnectionDetails{}}
	if err != nil {
		retErr := fmt.Errorf("failed to create postgres cluster: %w", err)
		if apiResponse != nil && apiResponse.Response != nil {
			retErr = fmt.Errorf("%w API Response Status: %v", retErr, apiResponse.Status)
		}
		return creation, retErr
	}

	// Set External Name
	cr.Status.AtProvider.ClusterID = *instance.Id
	meta.SetExternalName(cr, *instance.Id)
	return creation, nil
}

func (c *externalCluster) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.PostgresCluster)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotCluster)
	}
	if cr.Status.AtProvider.State == string(ionoscloud.BUSY) {
		return managed.ExternalUpdate{}, nil
	}

	clusterID := cr.Status.AtProvider.ClusterID
	instanceInput, err := postgrescluster.GenerateUpdateClusterInput(cr)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	_, apiResponse, err := c.service.UpdateCluster(ctx, clusterID, *instanceInput)
	if err != nil {
		retErr := fmt.Errorf("failed to update postgres cluster: %w", err)
		if apiResponse != nil && apiResponse.Response != nil {
			retErr = fmt.Errorf("%w API Response Status: %v", retErr, apiResponse.Status)
		}
		return managed.ExternalUpdate{}, retErr
	}
	return managed.ExternalUpdate{}, nil
}

func (c *externalCluster) Delete(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*v1alpha1.PostgresCluster)
	if !ok {
		return errors.New(errNotCluster)
	}

	cr.SetConditions(xpv1.Deleting())
	if cr.Status.AtProvider.State == string(ionoscloud.DESTROYING) {
		return nil
	}

	apiResponse, err := c.service.DeleteCluster(ctx, cr.Status.AtProvider.ClusterID)
	if err != nil {
		if apiResponse != nil && apiResponse.Response != nil && apiResponse.StatusCode == http.StatusNotFound {
			return nil
		}
		return fmt.Errorf("failed to delete postgres cluster. error: %w", err)
	}
	return nil
}
