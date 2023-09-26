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
	"encoding/json"
	"fmt"

	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	clientApi "k8s.io/client-go/tools/clientcmd/api"
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
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"
)

const (
	errNotK8sCluster = "managed resource is not a K8s Cluster custom resource"
)

// Setup adds a controller that reconciles K8sCluster managed resources.
func Setup(mgr ctrl.Manager, l logging.Logger, rl workqueue.RateLimiter, opts *utils.ConfigurationOptions) error {
	name := managed.ControllerName(v1alpha1.ClusterGroupKind)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(controller.Options{
			RateLimiter: ratelimiter.NewController(),
		}).
		For(&v1alpha1.Cluster{}).
		Complete(managed.NewReconciler(mgr,
			resource.ManagedKind(v1alpha1.ClusterGroupVersionKind),
			managed.WithExternalConnecter(&connectorCluster{
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

// A connectorK8sCluster is expected to produce an ExternalClient when its Connect method
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
	_, ok := mg.(*v1alpha1.Cluster)
	if !ok {
		return nil, errors.New(errNotK8sCluster)
	}
	svc, err := clients.ConnectForCRD(ctx, mg, c.kube, c.usage)
	return &externalCluster{
		service:              &k8scluster.APIClient{IonosServices: svc},
		log:                  c.log,
		isUniqueNamesEnabled: c.isUniqueNamesEnabled}, err
}

// An ExternalClient observes, then either creates, updates, or deletes an
// externalCluster resource to ensure it reflects the managed resource's desired state.
type externalCluster struct {
	// A 'client' used to connect to the externalK8sCluster resource API. In practice this
	// would be something like an IONOS Cloud SDK client.
	service              k8scluster.Client
	log                  logging.Logger
	isUniqueNamesEnabled bool
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
	cr.Status.AtProvider.State = clients.GetCoreResourceState(&observed)
	clients.UpdateCondition(cr, cr.Status.AtProvider.State)

	if kubeconfig, _, err = c.service.GetKubeConfig(ctx, cr.Status.AtProvider.ClusterID); err != nil {
		c.log.Info(fmt.Sprintf("failed to get connection details. error: %v", err))
	}

	var clientkubeconfig clientApi.Config
	if err := json.Unmarshal([]byte(kubeconfig), &clientkubeconfig); err != nil {
		c.log.Info(fmt.Sprintf("failed to unmasrshal connection details. error: %v", err))
	}

	return managed.ExternalObservation{
		ResourceExists:          true,
		ResourceUpToDate:        k8scluster.IsK8sClusterUpToDate(cr, observed),
		ResourceLateInitialized: !cmp.Equal(current, &cr.Spec.ForProvider),
		ConnectionDetails: managed.ConnectionDetails{
			"kubeconfig": []byte(kubeconfig),
			"server":     []byte(clientkubeconfig.Clusters[mg.GetName()].Server),
			"name":       []byte(mg.GetName()),
			"token":      []byte(clientkubeconfig.AuthInfos["cluster-admin"].Token),
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

	if c.isUniqueNamesEnabled {
		// Clusters should have unique names per account.
		// Check if there are any existing clusters with the same name.
		// If there are multiple, an error will be returned.
		instance, err := c.service.CheckDuplicateK8sCluster(ctx, cr.Spec.ForProvider.Name)
		if err != nil {
			return managed.ExternalCreation{}, err
		}
		clusterID, err := c.service.GetK8sClusterID(instance)
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

	instanceInput := k8scluster.GenerateCreateK8sClusterInput(cr)
	newInstance, apiResponse, err := c.service.CreateK8sCluster(ctx, *instanceInput)
	creation := managed.ExternalCreation{ConnectionDetails: managed.ConnectionDetails{}}
	if err != nil {
		retErr := fmt.Errorf("failed to create k8s cluster. error: %w", err)
		return creation, compute.AddAPIResponseInfo(apiResponse, retErr)
	}
	// Set External Name
	cr.Status.AtProvider.ClusterID = *newInstance.Id
	meta.SetExternalName(cr, *newInstance.Id)
	return creation, nil
}

func (c *externalCluster) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.Cluster)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotK8sCluster)
	}
	if cr.Status.AtProvider.State == k8s.UPDATING {
		return managed.ExternalUpdate{}, nil
	}
	if cr.Status.AtProvider.State != k8s.ACTIVE {
		return managed.ExternalUpdate{}, fmt.Errorf("resource needs to be in ACTIVE state to update it, current state: %v", cr.Status.AtProvider.State)
	}

	instanceInput := k8scluster.GenerateUpdateK8sClusterInput(cr)
	if _, _, err := c.service.UpdateK8sCluster(ctx, cr.Status.AtProvider.ClusterID, *instanceInput); err != nil {
		return managed.ExternalUpdate{}, fmt.Errorf("failed to update k8s cluster. error: %w", err)
	}
	cr.Status.AtProvider.State = k8s.UPDATING
	return managed.ExternalUpdate{}, nil
}

func (c *externalCluster) Delete(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*v1alpha1.Cluster)
	if !ok {
		return errors.New(errNotK8sCluster)
	}

	if meta.GetExternalName(cr) == "" {
		return nil
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
	switch cr.Status.AtProvider.State {
	case k8s.DESTROYING:
		return nil
	case k8s.TERMINATED:
		return nil
	case k8s.ACTIVE:
		apiResponse, err := c.service.DeleteK8sCluster(ctx, cr.Status.AtProvider.ClusterID)
		if err != nil {
			retErr := fmt.Errorf("failed to delete k8s cluster. error: %w", err)
			return compute.CheckAPIResponseInfo(apiResponse, retErr)
		}
	default:
		return fmt.Errorf("resource needs to be in ACTIVE state to delete it, current state: %v", cr.Status.AtProvider.State)
	}
	return nil
}
