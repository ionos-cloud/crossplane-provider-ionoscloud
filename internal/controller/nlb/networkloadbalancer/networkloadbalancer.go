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

package networkloadbalancer

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/rung/go-safecast"
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
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute/ipblock"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/nlb/networkloadbalancer"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"
)

const errNotNetworkLoadBalancer = "managed resource is not a NetworkLoadBalancer"

// Setup adds a controller that reconciles NetworkLoadBalancer managed resources.
func Setup(mgr ctrl.Manager, l logging.Logger, rl workqueue.RateLimiter, opts *utils.ConfigurationOptions) error {
	name := managed.ControllerName(v1alpha1.NetworkLoadBalancerGroupKind)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(controller.Options{
			RateLimiter: ratelimiter.NewController(),
		}).
		For(&v1alpha1.NetworkLoadBalancer{}).
		Complete(managed.NewReconciler(mgr,
			resource.ManagedKind(v1alpha1.NetworkLoadBalancerGroupVersionKind),
			managed.WithExternalConnecter(&connectorNetworkLoadBalancer{
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

// A connectorNetworkLoadBalancer is expected to produce an ExternalClient when its Connect method
// is called.
type connectorNetworkLoadBalancer struct {
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
func (c *connectorNetworkLoadBalancer) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	_, ok := mg.(*v1alpha1.NetworkLoadBalancer)
	if !ok {
		return nil, errors.New(errNotNetworkLoadBalancer)
	}
	svc, err := clients.ConnectForCRD(ctx, mg, c.kube, c.usage)
	return &externalNetworkLoadBalancer{
		service:              &networkloadbalancer.APIClient{IonosServices: svc},
		ipBlockService:       &ipblock.APIClient{IonosServices: svc},
		log:                  c.log,
		isUniqueNamesEnabled: c.isUniqueNamesEnabled}, err
}

// An ExternalClient observes, then either creates, updates, or deletes an
// externalNetworkLoadBalancer resource to ensure it reflects the managed resource's desired state.
type externalNetworkLoadBalancer struct {
	// A 'client' used to connect to the externalNetworkLoadBalancer resource API. In practice this
	// would be something like an IONOS Cloud SDK client.
	service              networkloadbalancer.Client
	ipBlockService       ipblock.Client
	log                  logging.Logger
	isUniqueNamesEnabled bool
}

func (c *externalNetworkLoadBalancer) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) { // nolint:gocyclo
	cr, ok := mg.(*v1alpha1.NetworkLoadBalancer)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotNetworkLoadBalancer)
	}

	if meta.GetExternalName(cr) == "" {
		return managed.ExternalObservation{}, nil
	}
	observed, apiResponse, err := c.service.GetNetworkLoadBalancerByID(ctx, cr.Spec.ForProvider.DatacenterCfg.DatacenterID, meta.GetExternalName(cr))
	if err != nil {
		err = fmt.Errorf("failed to get network load balancer by id: %w", err)
		return managed.ExternalObservation{}, compute.ErrorUnlessNotFound(apiResponse, err)
	}
	isLateInitialized := networkloadbalancer.LateInitializer(&cr.Spec.ForProvider, observed)
	networkloadbalancer.SetStatus(&cr.Status.AtProvider, observed)
	if err != nil {
		return managed.ExternalObservation{}, err
	}
	listenerLanID, targetLanID, err := getLanIDs(cr)
	if err != nil {
		return managed.ExternalObservation{}, err
	}
	publicIPs, err := c.getPublicIPs(ctx, cr)
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	clients.UpdateCondition(cr, cr.Status.AtProvider.State)
	return managed.ExternalObservation{
		ResourceExists:          true,
		ResourceUpToDate:        networkloadbalancer.IsUpToDate(cr, observed, listenerLanID, targetLanID, publicIPs),
		ConnectionDetails:       managed.ConnectionDetails{},
		ResourceLateInitialized: isLateInitialized,
	}, nil
}

func (c *externalNetworkLoadBalancer) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) { // nolint: gocyclo
	cr, ok := mg.(*v1alpha1.NetworkLoadBalancer)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotNetworkLoadBalancer)
	}
	cr.SetConditions(xpv1.Creating())
	if cr.Status.AtProvider.State == compute.BUSY {
		return managed.ExternalCreation{}, nil
	}

	datacenterID := cr.Spec.ForProvider.DatacenterCfg.DatacenterID
	if c.isUniqueNamesEnabled {
		// isUniqueNamesEnabled option enforces NetworkLoadBalancer names to be unique per Datacenter
		// Multiple Network Load Balancers with the same name will trigger an error
		// If only one instance is found, it will be "imported"
		nlbDuplicateID, err := c.service.CheckDuplicateNetworkLoadBalancer(ctx, datacenterID, cr.Spec.ForProvider.Name)
		if err != nil {
			return managed.ExternalCreation{}, err
		}
		if nlbDuplicateID != "" {
			cr.Status.AtProvider.NetworkLoadBalancerID = nlbDuplicateID
			meta.SetExternalName(cr, nlbDuplicateID)
			return managed.ExternalCreation{}, nil
		}
	}
	publicIPs, err := c.getPublicIPs(ctx, cr)
	if err != nil {
		return managed.ExternalCreation{}, err
	}
	listenerLanID, targetLanID, err := getLanIDs(cr)
	if err != nil {
		return managed.ExternalCreation{}, err
	}
	nlbInput := networkloadbalancer.GenerateCreateInput(cr, listenerLanID, targetLanID, publicIPs)
	newInstance, _, err := c.service.CreateNetworkLoadBalancer(ctx, datacenterID, nlbInput)
	if err != nil {
		return managed.ExternalCreation{}, err
	}
	cr.Status.AtProvider.NetworkLoadBalancerID = *newInstance.Id
	meta.SetExternalName(cr, *newInstance.Id)

	return managed.ExternalCreation{}, nil
}

func (c *externalNetworkLoadBalancer) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.NetworkLoadBalancer)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotNetworkLoadBalancer)
	}
	if cr.Status.AtProvider.State == compute.BUSY || cr.Status.AtProvider.State == compute.UPDATING {
		return managed.ExternalUpdate{}, nil
	}

	datacenterID := cr.Spec.ForProvider.DatacenterCfg.DatacenterID
	nlbID := cr.Status.AtProvider.NetworkLoadBalancerID
	publicIPs, err := c.getPublicIPs(ctx, cr)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}
	listenerLanID, targetLanID, err := getLanIDs(cr)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}
	nlbInput := networkloadbalancer.GenerateUpdateInput(cr, listenerLanID, targetLanID, publicIPs)
	_, _, err = c.service.UpdateNetworkLoadBalancer(ctx, datacenterID, nlbID, nlbInput)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	return managed.ExternalUpdate{}, nil
}

func (c *externalNetworkLoadBalancer) Delete(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*v1alpha1.NetworkLoadBalancer)
	if !ok {
		return errors.New(errNotNetworkLoadBalancer)
	}

	cr.SetConditions(xpv1.Deleting())
	if cr.Status.AtProvider.State == compute.DESTROYING {
		return nil
	}
	datacenterID := cr.Spec.ForProvider.DatacenterCfg.DatacenterID
	nlbID := cr.Status.AtProvider.NetworkLoadBalancerID
	_, err := c.service.DeleteNetworkLoadBalancer(ctx, datacenterID, nlbID)
	if err != nil && errors.Is(err, networkloadbalancer.ErrNotFound) {
		return nil
	}
	return err
}

func (c *externalNetworkLoadBalancer) getPublicIPs(ctx context.Context, cr *v1alpha1.NetworkLoadBalancer) ([]string, error) {
	if len(cr.Spec.ForProvider.IpsCfg.IPs) == 0 && len(cr.Spec.ForProvider.IpsCfg.IPsBlocksCfg) == 0 {
		return nil, nil
	} else if len(cr.Spec.ForProvider.IpsCfg.IPs) > 0 {
		return cr.Spec.ForProvider.IpsCfg.IPs, nil
	}

	publicIPs := make([]string, 0, len(cr.Spec.ForProvider.IpsCfg.IPsBlocksCfg))
	for _, cfg := range cr.Spec.ForProvider.IpsCfg.IPsBlocksCfg {
		ips, err := c.ipBlockService.GetIPs(ctx, cfg.IPBlock.IPBlockID)
		if err != nil {
			return nil, err
		}
		// ips = index(cfg.Indexes, ips)
		publicIPs = append(publicIPs, ips...)
	}
	return publicIPs, nil
}

func getLanIDs(cr *v1alpha1.NetworkLoadBalancer) (int32, int32, error) {
	listenerLanID, err := safecast.Atoi32(cr.Spec.ForProvider.ListenerLanCfg.LanID)
	if err != nil {
		return 0, 0, fmt.Errorf("error converting listener Lan id: %w", err)
	}
	targetLanID, err := safecast.Atoi32(cr.Spec.ForProvider.TargetLanCfg.LanID)
	if err != nil {
		return 0, 0, fmt.Errorf("error converting target Lan id: %w", err)
	}
	return listenerLanID, targetLanID, nil
}

// func index(i string, s []string) []string {
//
// }
