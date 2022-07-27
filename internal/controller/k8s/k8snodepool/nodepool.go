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

package k8snodepool

import (
	"context"
	"fmt"

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

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/k8s/v1alpha1"
	apisv1alpha1 "github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute/datacenter"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute/ipblock"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/k8s"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/k8s/k8scluster"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/k8s/k8snodepool"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"
)

const errNotK8sNodePool = "managed resource is not a K8s NodePool custom resource"

// Setup adds a controller that reconciles K8sNodePool managed resources.
func Setup(mgr ctrl.Manager, l logging.Logger, rl workqueue.RateLimiter, opts *utils.ConfigurationOptions) error {
	name := managed.ControllerName(v1alpha1.NodePoolGroupKind)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(controller.Options{
			RateLimiter: ratelimiter.NewDefaultManagedRateLimiter(rl),
		}).
		For(&v1alpha1.NodePool{}).
		Complete(managed.NewReconciler(mgr,
			resource.ManagedKind(v1alpha1.NodePoolGroupVersionKind),
			managed.WithExternalConnecter(&connectorNodePool{
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

// A connectorK8sNodePool is expected to produce an ExternalClient when its Connect method
// is called.
type connectorNodePool struct {
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
func (c *connectorNodePool) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	_, ok := mg.(*v1alpha1.NodePool)
	if !ok {
		return nil, errors.New(errNotK8sNodePool)
	}
	svc, err := clients.ConnectForCRD(ctx, mg, c.kube, c.usage)
	return &externalNodePool{
		service:              &k8snodepool.APIClient{IonosServices: svc},
		clusterService:       &k8scluster.APIClient{IonosServices: svc},
		datacenterService:    &datacenter.APIClient{IonosServices: svc},
		ipBlockService:       &ipblock.APIClient{IonosServices: svc},
		log:                  c.log,
		isUniqueNamesEnabled: c.isUniqueNamesEnabled}, err
}

// An ExternalClient observes, then either creates, updates, or deletes an
// externalNodePool resource to ensure it reflects the managed resource's desired state.
type externalNodePool struct {
	// A 'client' used to connect to the externalK8sNodePool resource API. In practice this
	// would be something like an IONOS Cloud SDK client.
	service              k8snodepool.Client
	clusterService       k8scluster.Client
	datacenterService    datacenter.Client
	ipBlockService       ipblock.Client
	log                  logging.Logger
	isUniqueNamesEnabled bool
}

func (c *externalNodePool) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.NodePool)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotK8sNodePool)
	}

	// External Name of the CR is the K8sNodePool ID
	if meta.GetExternalName(cr) == "" {
		return managed.ExternalObservation{}, nil
	}
	observed, apiResponse, err := c.service.GetK8sNodePool(ctx, cr.Spec.ForProvider.ClusterCfg.ClusterID, meta.GetExternalName(cr))
	if err != nil {
		retErr := fmt.Errorf("failed to get k8s nodepool by id. error: %w", err)
		return managed.ExternalObservation{}, compute.CheckAPIResponseInfo(apiResponse, retErr)
	}

	current := cr.Spec.ForProvider.DeepCopy()
	k8snodepool.LateInitializer(&cr.Spec.ForProvider, &observed)
	k8snodepool.LateStatusInitializer(&cr.Status.AtProvider, &observed)

	// Set Ready condition based on State
	cr.Status.AtProvider.NodePoolID = meta.GetExternalName(cr)
	cr.Status.AtProvider.State = clients.GetCoreResourceState(&observed)
	c.log.Debug(fmt.Sprintf("Observing state: %v", cr.Status.AtProvider.State))
	clients.UpdateCondition(cr, cr.Status.AtProvider.State)

	publicIps, err := c.getPublicIPsSet(ctx, cr)
	if err != nil {
		return managed.ExternalObservation{}, fmt.Errorf("failed to get public IPs: %w", err)
	}
	return managed.ExternalObservation{
		ResourceExists:          true,
		ResourceUpToDate:        k8snodepool.IsK8sNodePoolUpToDate(cr, observed, publicIps),
		ResourceLateInitialized: !cmp.Equal(current, &cr.Spec.ForProvider),
		ConnectionDetails:       managed.ConnectionDetails{},
	}, nil
}

func (c *externalNodePool) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) { //nolint:gocyclo
	cr, ok := mg.(*v1alpha1.NodePool)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotK8sNodePool)
	}
	if err := c.ensureClusterIsActive(ctx, cr.Spec.ForProvider.ClusterCfg.ClusterID); err != nil {
		return managed.ExternalCreation{}, fmt.Errorf("cluster must be active to create nodepool: %w", err)
	}
	cr.SetConditions(xpv1.Creating())
	if cr.Status.AtProvider.State == k8s.DEPLOYING {
		return managed.ExternalCreation{}, nil
	}

	if c.isUniqueNamesEnabled {
		// NodePools should have unique names per cluster.
		// Check if there are any existing node pools with the same name.
		// If there are multiple, an error will be returned.
		instance, err := c.service.CheckDuplicateK8sNodePool(ctx, cr.Spec.ForProvider.ClusterCfg.ClusterID,
			cr.Spec.ForProvider.Name, cr)
		if err != nil {
			return managed.ExternalCreation{}, err
		}
		nodePoolID, err := c.service.GetK8sNodePoolID(instance)
		if err != nil {
			return managed.ExternalCreation{}, err
		}
		if nodePoolID != "" {
			// "Import" existing nodePool.
			cr.Status.AtProvider.NodePoolID = nodePoolID
			meta.SetExternalName(cr, nodePoolID)
			return managed.ExternalCreation{}, nil
		}
	}

	// Note: If the CPU Family is not set by the user, the Crossplane Provider IONOS Cloud
	// will take the first CPU Family offered by the Datacenter CPU Architectures available
	if cr.Spec.ForProvider.CPUFamily == "" {
		cpuFamilies, err := c.datacenterService.GetCPUFamiliesForDatacenter(ctx, cr.Spec.ForProvider.DatacenterCfg.DatacenterID)
		if err != nil {
			return managed.ExternalCreation{}, fmt.Errorf("failed to get CPU Families AVAILABLE for datacenter. error: %w", err)
		}
		if len(cpuFamilies) > 0 {
			cr.Spec.ForProvider.CPUFamily = cpuFamilies[0]
		}
	}
	publicIPs, err := c.getPublicIPsSet(ctx, cr)
	if err != nil {
		return managed.ExternalCreation{}, fmt.Errorf("failed to get public IPs: %w", err)
	}
	instanceInput := k8snodepool.GenerateCreateK8sNodePoolInput(cr, publicIPs)
	newInstance, apiResponse, err := c.service.CreateK8sNodePool(ctx, cr.Spec.ForProvider.ClusterCfg.ClusterID, *instanceInput)
	creation := managed.ExternalCreation{ConnectionDetails: managed.ConnectionDetails{}}
	if err != nil {
		retErr := fmt.Errorf("failed to create k8s nodepool. error: %w", err)
		return creation, compute.AddAPIResponseInfo(apiResponse, retErr)
	}
	// Set External Name
	cr.Status.AtProvider.NodePoolID = *newInstance.Id
	meta.SetExternalName(cr, *newInstance.Id)
	return creation, nil
}

func (c *externalNodePool) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.NodePool)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotK8sNodePool)
	}

	if err := c.ensureClusterIsActive(ctx, cr.Spec.ForProvider.ClusterCfg.ClusterID); err != nil {
		return managed.ExternalUpdate{}, fmt.Errorf("cluster must be active to update nodepool: %w", err)
	}

	if cr.Status.AtProvider.State == compute.UPDATING {
		return managed.ExternalUpdate{}, nil
	}

	publicIPs, err := c.getPublicIPsSet(ctx, cr)
	if err != nil {
		return managed.ExternalUpdate{}, fmt.Errorf("failed to get public IPs: %w", err)
	}
	instanceInput := k8snodepool.GenerateUpdateK8sNodePoolInput(cr, publicIPs)
	if _, _, err = c.service.UpdateK8sNodePool(ctx, cr.Spec.ForProvider.ClusterCfg.ClusterID, cr.Status.AtProvider.NodePoolID, *instanceInput); err != nil {
		return managed.ExternalUpdate{}, fmt.Errorf("failed to update k8s nodepool. error: %w", err)
	}
	cr.Status.AtProvider.State = compute.UPDATING
	return managed.ExternalUpdate{}, nil
}

func (c *externalNodePool) Delete(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*v1alpha1.NodePool)
	if !ok {
		return errors.New(errNotK8sNodePool)
	}

	if cr.Status.AtProvider.NodePoolID == "" {
		return nil
	}

	if cr.Status.AtProvider.State == k8s.DESTROYING || cr.Status.AtProvider.State == k8s.TERMINATED {
		cr.SetConditions(xpv1.Deleting())
		return nil
	}

	if cr.Status.AtProvider.State == k8s.DEPLOYING {
		return errors.New("can't delete nodepool in state DEPLOYING")
	}

	if err := c.ensureClusterIsActive(ctx, cr.Spec.ForProvider.ClusterCfg.ClusterID); err != nil {
		return fmt.Errorf("cluster must be active to delete nodepool: %w", err)
	}

	cr.SetConditions(xpv1.Deleting())

	apiResponse, err := c.service.DeleteK8sNodePool(ctx, cr.Spec.ForProvider.ClusterCfg.ClusterID, cr.Status.AtProvider.NodePoolID)
	if err != nil {
		retErr := fmt.Errorf("failed to delete k8s nodepool. error: %w", err)
		return compute.CheckAPIResponseInfo(apiResponse, retErr)
	}
	return nil
}

// getPublicIPsSet will return Public IPs set by the user on ips or ipsConfig fields of the spec.
// If both fields are set, only the ips field will be considered by the Crossplane
// Provider IONOS Cloud.
func (c *externalNodePool) getPublicIPsSet(ctx context.Context, cr *v1alpha1.NodePool) ([]string, error) {
	if len(cr.Spec.ForProvider.PublicIPsCfg.IPs) == 0 && len(cr.Spec.ForProvider.PublicIPsCfg.IPBlockCfgs) == 0 {
		return nil, nil
	}
	if len(cr.Spec.ForProvider.PublicIPsCfg.IPs) > 0 {
		return cr.Spec.ForProvider.PublicIPsCfg.IPs, nil
	}
	ips := make([]string, 0)
	if len(cr.Spec.ForProvider.PublicIPsCfg.IPBlockCfgs) > 0 {
		for _, cfg := range cr.Spec.ForProvider.PublicIPsCfg.IPBlockCfgs {
			ipsCfg, err := c.ipBlockService.GetIPs(ctx, cfg.IPBlockID, cfg.Indexes...)
			if err != nil {
				return nil, err
			}
			ips = append(ips, ipsCfg...)
		}
	}
	return ips, nil
}

// ensureClusterIsActive returns an error if the cluster state is not found or the state of the cluster is not ACTIVE
func (c *externalNodePool) ensureClusterIsActive(ctx context.Context, clusterID string) error {
	observedCluster, _, err := c.clusterService.GetK8sCluster(ctx, clusterID)
	if err != nil {
		return fmt.Errorf("failed to get k8s cluster by id. error: %w", err)
	}
	if *observedCluster.Metadata.State != k8s.ACTIVE {
		return fmt.Errorf("k8s cluster must be in ACTIVE state, current state: %v", *observedCluster.Metadata.State)
	}
	return nil
}
