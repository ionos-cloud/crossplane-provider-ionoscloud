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

package k8scluster

import (
	"context"
	"fmt"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
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

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/k8s/v1alpha1"
	apisv1alpha1 "github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/k8s"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/k8s/k8scluster"
)

const (
	errNotK8sCluster = "managed resource is not a K8s Cluster custom resource"
	errTrackPCUsage  = "cannot track ProviderConfig usage"
	errGetPC         = "cannot get ProviderConfig"
	errGetCreds      = "cannot get credentials"

	errNewClient = "cannot create new Service"
)

// Setup adds a controller that reconciles K8sCluster managed resources.
func Setup(mgr ctrl.Manager, l logging.Logger, rl workqueue.RateLimiter, poll time.Duration, creationGracePeriod time.Duration) error {
	name := managed.ControllerName(v1alpha1.ClusterGroupKind)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(controller.Options{
			RateLimiter: ratelimiter.NewDefaultManagedRateLimiter(rl),
		}).
		For(&v1alpha1.Cluster{}).
		Complete(managed.NewReconciler(mgr,
			resource.ManagedKind(v1alpha1.ClusterGroupVersionKind),
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

// A connectorK8sCluster is expected to produce an ExternalClient when its Connect method
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
	_, ok := mg.(*v1alpha1.Cluster)
	if !ok {
		return nil, errors.New(errNotK8sCluster)
	}

	if err := c.usage.Track(ctx, mg); err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	pc := &apisv1alpha1.ProviderConfig{}
	if err := c.kube.Get(ctx, types.NamespacedName{Name: mg.GetProviderConfigReference().Name}, pc); err != nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	cd := pc.Spec.Credentials
	data, err := resource.CommonCredentialExtractor(ctx, cd.Source, c.kube, cd.CommonCredentialSelectors)
	if err != nil {
		return nil, errors.Wrap(err, errGetCreds)
	}

	svc, err := clients.NewIonosClients(data)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}
	return &externalCluster{service: &k8scluster.APIClient{IonosServices: svc}, log: c.log}, nil
}

// An ExternalClient observes, then either creates, updates, or deletes an
// externalCluster resource to ensure it reflects the managed resource's desired state.
type externalCluster struct {
	// A 'client' used to connect to the externalK8sCluster resource API. In practice this
	// would be something like an IONOS Cloud SDK client.
	service k8scluster.Client
	log     logging.Logger
}

func (c *externalCluster) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	var kubeconfig string

	cr, ok := mg.(*v1alpha1.Cluster)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotK8sCluster)
	}

	// External Name of the CR is the K8sCluster ID
	if meta.GetExternalName(cr) == "" {
		return managed.ExternalObservation{}, nil
	}
	observed, apiResponse, err := c.service.GetK8sCluster(ctx, meta.GetExternalName(cr))
	if err != nil {
		retErr := fmt.Errorf("failed to get k8s cluster by id. error: %w", err)
		return managed.ExternalObservation{}, compute.CheckAPIResponseInfo(apiResponse, retErr)
	}

	current := cr.Spec.ForProvider.DeepCopy()
	k8scluster.LateInitializer(&cr.Spec.ForProvider, &observed)
	k8scluster.LateStatusInitializer(&cr.Status.AtProvider, &observed)

	// Set Ready condition based on State
	cr.Status.AtProvider.ClusterID = meta.GetExternalName(cr)
	cr.Status.AtProvider.State = *observed.Metadata.State
	c.log.Debug(fmt.Sprintf("Observing state: %v", cr.Status.AtProvider.State))
	switch cr.Status.AtProvider.State {
	case k8s.AVAILABLE, k8s.ACTIVE:
		cr.SetConditions(xpv1.Available())
	case k8s.DESTROYING, k8s.TERMINATED:
		cr.SetConditions(xpv1.Deleting())
	case k8s.BUSY, k8s.DEPLOYING, k8s.UPDATING:
		cr.SetConditions(xpv1.Creating())
	default:
		cr.SetConditions(xpv1.Unavailable())
	}

	if kubeconfig, _, err = c.service.GetKubeConfig(ctx, cr.Status.AtProvider.ClusterID); err != nil {
		c.log.Info(fmt.Sprintf("failed to get connection details. error: %v", err))
	}

	return managed.ExternalObservation{
		ResourceExists:          true,
		ResourceUpToDate:        k8scluster.IsK8sClusterUpToDate(cr, observed),
		ResourceLateInitialized: !cmp.Equal(current, &cr.Spec.ForProvider),
		ConnectionDetails: managed.ConnectionDetails{
			"kubeconfig": []byte(kubeconfig),
		},
	}, nil
}

func (c *externalCluster) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.Cluster)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotK8sCluster)
	}

	cr.SetConditions(xpv1.Creating())
	if cr.Status.AtProvider.State == k8s.DEPLOYING {
		return managed.ExternalCreation{}, nil
	}
	instanceInput, err := k8scluster.GenerateCreateK8sClusterInput(cr)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	instance, apiResponse, err := c.service.CreateK8sCluster(ctx, *instanceInput)
	creation := managed.ExternalCreation{ConnectionDetails: managed.ConnectionDetails{}}
	if err != nil {
		retErr := fmt.Errorf("failed to create k8s cluster. error: %w", err)
		return creation, compute.AddAPIResponseInfo(apiResponse, retErr)
	}

	// Set External Name
	cr.Status.AtProvider.ClusterID = *instance.Id
	meta.SetExternalName(cr, *instance.Id)
	return creation, nil
}

func (c *externalCluster) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.Cluster)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotK8sCluster)
	}
	if cr.Status.AtProvider.State == compute.UPDATING {
		return managed.ExternalUpdate{}, nil
	}
	if cr.Status.AtProvider.State != compute.ACTIVE {
		return managed.ExternalUpdate{}, fmt.Errorf("resource needs to be in ACTIVE state to update it, current state: %v", cr.Status.AtProvider.State)
	}

	instanceInput, err := k8scluster.GenerateUpdateK8sClusterInput(cr)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}
	if _, _, err = c.service.UpdateK8sCluster(ctx, cr.Status.AtProvider.ClusterID, *instanceInput); err != nil {
		return managed.ExternalUpdate{}, fmt.Errorf("failed to update k8s cluster. error: %w", err)
	}
	cr.Status.AtProvider.State = compute.UPDATING
	return managed.ExternalUpdate{}, nil
}

func (c *externalCluster) Delete(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*v1alpha1.Cluster)
	if !ok {
		return errors.New(errNotK8sCluster)
	}

	// Note: If the K8s Cluster still has NodePools, the API Request will fail.
	hasNodePools, err := c.service.HasActiveK8sNodePools(ctx, cr.Status.AtProvider.ClusterID)
	if err != nil {
		return fmt.Errorf("failed to check if the Kubernetes Cluster has Active NodePools. error: %w", err)
	}
	if hasNodePools {
		return fmt.Errorf("kubernetes cluster cannot be deleted. NodePools still exist")
	}

	cr.SetConditions(xpv1.Deleting())
	if cr.Status.AtProvider.State == compute.DESTROYING || cr.Status.AtProvider.State == k8s.TERMINATED {
		return nil
	}
	if cr.Status.AtProvider.State != compute.ACTIVE {
		return fmt.Errorf("resource needs to be in ACTIVE state to delete it, current state: %v", cr.Status.AtProvider.State)
	}
	apiResponse, err := c.service.DeleteK8sCluster(ctx, cr.Status.AtProvider.ClusterID)
	if err != nil {
		retErr := fmt.Errorf("failed to delete k8s cluster. error: %w", err)
		return compute.AddAPIResponseInfo(apiResponse, retErr)
	}
	return nil
}
