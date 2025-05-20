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

package mongocluster

import (
	"context"
	"fmt"
	"net/http"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/crossplane/crossplane-runtime/pkg/statemetrics"
	ionoscloud "github.com/ionos-cloud/sdk-go-dbaas-mongo"
	"github.com/pkg/errors"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/dbaas/mongo/v1alpha1"
	apisv1alpha1 "github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/dbaas/mongo"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"
)

const errNotCluster = "managed resource is not a Cluster custom resource"

// Setup adds a controller that reconciles Cluster managed resources.
func Setup(mgr ctrl.Manager, opts *utils.ConfigurationOptions) error {
	name := managed.ControllerName(v1alpha1.MongoClusterGroupKind)
	logger := opts.CtrlOpts.Logger
	if opts.CtrlOpts.MetricOptions != nil && opts.CtrlOpts.MetricOptions.MRStateMetrics != nil {
		stateMetricsRecorder := statemetrics.NewMRStateRecorder(
			mgr.GetClient(), opts.CtrlOpts.Logger, opts.CtrlOpts.MetricOptions.MRStateMetrics, &v1alpha1.MongoClusterList{}, opts.CtrlOpts.MetricOptions.PollStateMetricInterval,
		)
		if err := mgr.Add(stateMetricsRecorder); err != nil {
			return errors.Wrap(err, "cannot register MR state metrics recorder for kind"+name)
		}
	}

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: opts.GetMaxConcurrentReconcileRate(v1alpha1.MongoClusterKind),
			RateLimiter:             ratelimiter.NewController(),
			RecoverPanic:            ptr.To(true),
		}).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.MongoCluster{}).
		Complete(managed.NewReconciler(mgr,
			resource.ManagedKind(v1alpha1.MongoClusterGroupVersionKind),
			managed.WithExternalConnecter(&connectorCluster{
				kube:                 mgr.GetClient(),
				usage:                resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
				log:                  logger,
				isUniqueNamesEnabled: opts.GetIsUniqueNamesEnabled()}),
			managed.WithReferenceResolver(managed.NewAPISimpleReferenceResolver(mgr.GetClient())),
			managed.WithInitializers(),
			managed.WithPollInterval(opts.GetPollInterval()),
			managed.WithTimeout(opts.GetTimeout()),
			managed.WithCreationGracePeriod(opts.GetCreationGracePeriod()),
			managed.WithLogger(logger.WithValues("controller", name)),
			managed.WithMetricRecorder(opts.CtrlOpts.MetricOptions.MRMetrics),
			managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name)))))
}

// A connectorCluster is expected to produce an ExternalClient when its Connect method
// is called.
type connectorCluster struct {
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
func (c *connectorCluster) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	_, ok := mg.(*v1alpha1.MongoCluster)
	if !ok {
		return nil, errors.New(errNotCluster)
	}
	svc, err := clients.ConnectForCRD(ctx, mg, c.kube, c.usage)
	return &externalCluster{
		service:              &mongo.ClusterAPIClient{IonosServices: svc},
		log:                  c.log,
		isUniqueNamesEnabled: c.isUniqueNamesEnabled,
		client:               c.kube}, err

}

// An ExternalClient observes, then either creates, updates, or deletes an
// externalCluster resource to ensure it reflects the managed resource's desired state.
type externalCluster struct {
	// A 'client' used to connect to the externalCluster resource API. In practice this
	// would be something like an IONOS Cloud SDK client.
	service              mongo.ClusterClient
	client               client.Client
	log                  logging.Logger
	isUniqueNamesEnabled bool
}

func (c *externalCluster) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) { // nolint:gocyclo
	cr, ok := mg.(*v1alpha1.MongoCluster)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotCluster)
	}

	// External Name of the CR is the DBaaS Mongo Cluster ID
	if meta.GetExternalName(cr) == "" {
		return managed.ExternalObservation{}, nil
	}
	observed, resp, err := c.service.GetCluster(ctx, meta.GetExternalName(cr))
	if err != nil {
		if resp.HttpNotFound() {
			return managed.ExternalObservation{}, nil
		}
		return managed.ExternalObservation{}, fmt.Errorf("failed to get mongo cluster by id. err: %w", err)
	}

	lateInitialized := mongo.LateInitializer(&cr.Spec.ForProvider, &observed)

	cr.Status.AtProvider.ClusterID = meta.GetExternalName(cr)
	if observed.Metadata != nil && observed.Metadata.State != nil {
		cr.Status.AtProvider.State = string(*observed.GetMetadata().GetState())
	}
	c.log.Debug(fmt.Sprintf("Observing state: %v", cr.Status.AtProvider.State))
	clients.UpdateCondition(cr, cr.Status.AtProvider.State)

	return managed.ExternalObservation{
		ResourceExists:          true,
		ResourceUpToDate:        mongo.IsClusterUpToDate(cr, observed),
		ConnectionDetails:       managed.ConnectionDetails{},
		ResourceLateInitialized: lateInitialized,
	}, nil
}

func (c *externalCluster) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) { // nolint: gocyclo
	cr, ok := mg.(*v1alpha1.MongoCluster)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotCluster)
	}
	cr.SetConditions(xpv1.Creating())
	if cr.Status.AtProvider.State == string(ionoscloud.STATE_BUSY) {
		return managed.ExternalCreation{}, nil
	}

	if c.isUniqueNamesEnabled {
		// Clusters should have unique names per account.
		// Check if there are any existing clusters with the same name.
		// If there are multiple, an error will be returned.
		instance, err := c.service.CheckDuplicateCluster(ctx, cr.Spec.ForProvider.DisplayName, cr)
		if err != nil {
			return managed.ExternalCreation{}, err
		}
		clusterID, err := c.service.GetClusterID(instance)
		if err != nil {
			return managed.ExternalCreation{}, err
		}
		if clusterID != "" {
			// "Import" existing cluster.
			cr.Status.AtProvider.ClusterID = clusterID
			meta.SetExternalName(cr, clusterID)
			return managed.ExternalCreation{}, nil
		}
	}
	instanceInput, err := mongo.GenerateCreateClusterInput(cr)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	newInstance, apiResponse, err := c.service.CreateCluster(ctx, *instanceInput)
	creation := managed.ExternalCreation{ConnectionDetails: managed.ConnectionDetails{}}
	if err != nil {
		retErr := fmt.Errorf("failed to create mongo cluster: %w", err)
		if apiResponse != nil && apiResponse.Response != nil {
			retErr = fmt.Errorf("%w API Response Status: %v", retErr, apiResponse.Status)
		}
		return creation, retErr
	}

	// Set External Name
	cr.Status.AtProvider.ClusterID = *newInstance.Id
	meta.SetExternalName(cr, *newInstance.Id)
	return creation, nil
}

func (c *externalCluster) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.MongoCluster)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotCluster)
	}
	if cr.Status.AtProvider.State == string(ionoscloud.STATE_BUSY) {
		return managed.ExternalUpdate{}, nil
	}

	clusterID := cr.Status.AtProvider.ClusterID
	instanceInput, err := mongo.GenerateUpdateClusterInput(cr)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	_, apiResponse, err := c.service.UpdateCluster(ctx, clusterID, *instanceInput)
	if err != nil {
		retErr := fmt.Errorf("failed to update mongo cluster: %w", err)
		if apiResponse != nil && apiResponse.Response != nil {
			retErr = fmt.Errorf("%w API Response Status: %v", retErr, apiResponse.Status)
		}
		return managed.ExternalUpdate{}, retErr
	}
	return managed.ExternalUpdate{}, nil
}

func (c *externalCluster) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.MongoCluster)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotCluster)
	}

	cr.SetConditions(xpv1.Deleting())
	if cr.Status.AtProvider.State == string(ionoscloud.STATE_DESTROYING) {
		return managed.ExternalDelete{}, nil
	}

	apiResponse, err := c.service.DeleteCluster(ctx, cr.Status.AtProvider.ClusterID)
	if err != nil {
		if apiResponse != nil && apiResponse.Response != nil && apiResponse.StatusCode == http.StatusNotFound {
			return managed.ExternalDelete{}, nil
		}
		return managed.ExternalDelete{}, fmt.Errorf("failed to delete mongo cluster. error: %w", err)
	}
	return managed.ExternalDelete{}, nil
}

// Disconnect does nothing because there are no resources to release. Needs to be implemented starting from crossplane-runtime v0.17
func (c *externalCluster) Disconnect(_ context.Context) error {
	return nil
}
