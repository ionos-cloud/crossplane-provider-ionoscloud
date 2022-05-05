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
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute/datacenter"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute/ipblock"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/k8s"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/k8s/k8scluster"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/k8s/k8snodepool"
)

const (
	errNotK8sNodePool = "managed resource is not a K8s NodePool custom resource"
	errTrackPCUsage   = "cannot track ProviderConfig usage"
	errGetPC          = "cannot get ProviderConfig"
	errGetCreds       = "cannot get credentials"

	errNewClient = "cannot create new Service"
)

// Setup adds a controller that reconciles K8sNodePool managed resources.
func Setup(mgr ctrl.Manager, l logging.Logger, rl workqueue.RateLimiter, poll time.Duration) error {
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
				kube:  mgr.GetClient(),
				usage: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
				log:   l}),
			managed.WithReferenceResolver(managed.NewAPISimpleReferenceResolver(mgr.GetClient())),
			managed.WithInitializers(),
			managed.WithPollInterval(poll),
			managed.WithLogger(l.WithValues("controller", name)),
			managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name)))))
}

// A connectorK8sNodePool is expected to produce an ExternalClient when its Connect method
// is called.
type connectorNodePool struct {
	kube  client.Client
	usage resource.Tracker
	log   logging.Logger
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
	return &externalNodePool{
		service:           &k8snodepool.APIClient{IonosServices: svc},
		clusterService:    &k8scluster.APIClient{IonosServices: svc},
		datacenterService: &datacenter.APIClient{IonosServices: svc},
		ipBlockService:    &ipblock.APIClient{IonosServices: svc},
		log:               c.log,
	}, nil
}

// An ExternalClient observes, then either creates, updates, or deletes an
// externalNodePool resource to ensure it reflects the managed resource's desired state.
type externalNodePool struct {
	// A 'client' used to connect to the externalK8sNodePool resource API. In practice this
	// would be something like an IONOS Cloud SDK client.
	service           k8snodepool.Client
	clusterService    k8scluster.Client
	datacenterService datacenter.Client
	ipBlockService    ipblock.Client
	log               logging.Logger
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

	publicIps, err := c.getPublicIPsSet(ctx, cr)
	if err != nil {
		return managed.ExternalObservation{}, fmt.Errorf("failed to get public IPs: %w", err)
	}
	gatewayIP, err := c.getGatewayIPSet(ctx, cr)
	if err != nil {
		return managed.ExternalObservation{}, fmt.Errorf("failed to get gateway IP: %w", err)
	}
	return managed.ExternalObservation{
		ResourceExists:          true,
		ResourceUpToDate:        k8snodepool.IsK8sNodePoolUpToDate(cr, observed, publicIps, gatewayIP),
		ResourceLateInitialized: !cmp.Equal(current, &cr.Spec.ForProvider),
		ConnectionDetails:       managed.ConnectionDetails{},
	}, nil
}

func (c *externalNodePool) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) { //nolint:gocyclo
	cr, ok := mg.(*v1alpha1.NodePool)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotK8sNodePool)
	}

	observedCluster, apiResponse, err := c.clusterService.GetK8sCluster(ctx, cr.Spec.ForProvider.ClusterCfg.ClusterID)
	if err != nil {
		retErr := fmt.Errorf("failed to get k8s cluster by id. error: %w", err)
		return managed.ExternalCreation{}, compute.CheckAPIResponseInfo(apiResponse, retErr)
	}
	if *observedCluster.Metadata.State != k8s.ACTIVE {
		return managed.ExternalCreation{}, fmt.Errorf("k8s cluster must be in ACTIVE state, current state: %v", *observedCluster.Metadata.State)
	}

	cr.SetConditions(xpv1.Creating())
	if cr.Status.AtProvider.State == k8s.DEPLOYING {
		return managed.ExternalCreation{}, nil
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
	gatewayIP, err := c.getGatewayIPSet(ctx, cr)
	if err != nil {
		return managed.ExternalCreation{}, fmt.Errorf("failed to get gateway IP: %w", err)
	}
	instanceInput, err := k8snodepool.GenerateCreateK8sNodePoolInput(cr, publicIPs, gatewayIP)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	instance, apiResponse, err := c.service.CreateK8sNodePool(ctx, cr.Spec.ForProvider.ClusterCfg.ClusterID, *instanceInput)
	creation := managed.ExternalCreation{ConnectionDetails: managed.ConnectionDetails{}}
	if err != nil {
		retErr := fmt.Errorf("failed to create k8s nodepool. error: %w", err)
		return creation, compute.AddAPIResponseInfo(apiResponse, retErr)
	}

	// Set External Name
	cr.Status.AtProvider.NodePoolID = *instance.Id
	meta.SetExternalName(cr, *instance.Id)
	creation.ExternalNameAssigned = true
	return creation, nil
}

func (c *externalNodePool) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.NodePool)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotK8sNodePool)
	}
	observedCluster, apiResponse, err := c.clusterService.GetK8sCluster(ctx, cr.Spec.ForProvider.ClusterCfg.ClusterID)
	if err != nil {
		retErr := fmt.Errorf("failed to get k8s cluster by id. error: %w", err)
		return managed.ExternalUpdate{}, compute.CheckAPIResponseInfo(apiResponse, retErr)
	}
	if *observedCluster.Metadata.State != k8s.ACTIVE {
		return managed.ExternalUpdate{}, fmt.Errorf("k8s cluster must be in ACTIVE state, current state: %v", *observedCluster.Metadata.State)
	}
	if cr.Status.AtProvider.State == compute.UPDATING {
		return managed.ExternalUpdate{}, nil
	}

	publicIPs, err := c.getPublicIPsSet(ctx, cr)
	if err != nil {
		return managed.ExternalUpdate{}, fmt.Errorf("failed to get public IPs: %w", err)
	}
	instanceInput, err := k8snodepool.GenerateUpdateK8sNodePoolInput(cr, publicIPs)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}
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

	observedCluster, apiResponse, err := c.clusterService.GetK8sCluster(ctx, cr.Spec.ForProvider.ClusterCfg.ClusterID)
	if err != nil {
		retErr := fmt.Errorf("failed to get k8s cluster by id. error: %w", err)
		return compute.CheckAPIResponseInfo(apiResponse, retErr)
	}
	if *observedCluster.Metadata.State != k8s.ACTIVE {
		return fmt.Errorf("k8s cluster must be in ACTIVE state, current state: %v", *observedCluster.Metadata.State)
	}
	cr.SetConditions(xpv1.Deleting())
	if cr.Status.AtProvider.State == compute.DESTROYING || cr.Status.AtProvider.State == k8s.TERMINATED {
		return nil
	}

	apiResponse, err = c.service.DeleteK8sNodePool(ctx, cr.Spec.ForProvider.ClusterCfg.ClusterID, cr.Status.AtProvider.NodePoolID)
	if err != nil {
		retErr := fmt.Errorf("failed to delete k8s nodepool. error: %w", err)
		return compute.AddAPIResponseInfo(apiResponse, retErr)
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

// getGatewayIPSet will return ip set by the user on ip or ipConfig fields of the spec.
// If both fields are set, only the ip field will be considered by the Crossplane
// Provider IONOS Cloud.
func (c *externalNodePool) getGatewayIPSet(ctx context.Context, cr *v1alpha1.NodePool) (string, error) {
	if cr.Spec.ForProvider.GatewayIPCfg.IP == "" && cr.Spec.ForProvider.GatewayIPCfg.IPBlockCfg.IPBlockID == "" {
		return "", nil
	}
	if cr.Spec.ForProvider.GatewayIPCfg.IP != "" {
		return cr.Spec.ForProvider.GatewayIPCfg.IP, nil
	}
	if cr.Spec.ForProvider.GatewayIPCfg.IPBlockCfg.IPBlockID != "" {
		ipsCfg, err := c.ipBlockService.GetIPs(ctx, cr.Spec.ForProvider.GatewayIPCfg.IPBlockCfg.IPBlockID, cr.Spec.ForProvider.GatewayIPCfg.IPBlockCfg.Index)
		if err != nil {
			return "", err
		}
		if len(ipsCfg) != 1 {
			return "", fmt.Errorf("error getting IP with index %v from IPBlock %v",
				cr.Spec.ForProvider.GatewayIPCfg.IPBlockCfg.Index, cr.Spec.ForProvider.GatewayIPCfg.IPBlockCfg.IPBlockID)
		}
		return ipsCfg[0], nil
	}
	return "", fmt.Errorf("error getting IP set")
}
