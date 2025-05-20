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

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/crossplane/crossplane-runtime/pkg/statemetrics"
	"github.com/pkg/errors"
	"github.com/rung/go-safecast"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

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
func Setup(mgr ctrl.Manager, opts *utils.ConfigurationOptions) error {
	name := managed.ControllerName(v1alpha1.NetworkLoadBalancerGroupKind)
	logger := opts.CtrlOpts.Logger
	if opts.CtrlOpts.MetricOptions != nil && opts.CtrlOpts.MetricOptions.MRStateMetrics != nil {
		stateMetricsRecorder := statemetrics.NewMRStateRecorder(
			mgr.GetClient(), opts.CtrlOpts.Logger, opts.CtrlOpts.MetricOptions.MRStateMetrics, &v1alpha1.NetworkLoadBalancerList{}, opts.CtrlOpts.MetricOptions.PollStateMetricInterval,
		)
		if err := mgr.Add(stateMetricsRecorder); err != nil {
			return errors.Wrap(err, "cannot register MR state metrics recorder for kind"+name)
		}
	}

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(opts.CtrlOpts.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.NetworkLoadBalancer{}).
		Complete(managed.NewReconciler(mgr,
			resource.ManagedKind(v1alpha1.NetworkLoadBalancerGroupVersionKind),
			managed.WithExternalConnecter(&connectorNetworkLoadBalancer{
				kube:                 mgr.GetClient(),
				usage:                resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
				log:                  logger,
				isUniqueNamesEnabled: opts.GetIsUniqueNamesEnabled()}),
			managed.WithReferenceResolver(managed.NewAPISimpleReferenceResolver(mgr.GetClient())),
			managed.WithInitializers(),
			managed.WithCreationGracePeriod(opts.GetCreationGracePeriod()),
			managed.WithPollInterval(opts.GetPollInterval()),
			managed.WithTimeout(opts.GetTimeout()),
			managed.WithLogger(logger.WithValues("controller", name)),
			managed.WithMetricRecorder(opts.CtrlOpts.MetricOptions.MRMetrics),
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
	networkLoadBalancerID := meta.GetExternalName(cr)
	if networkLoadBalancerID == "" {
		return managed.ExternalObservation{}, nil
	}
	observed, err := c.service.GetNetworkLoadBalancerByID(ctx, cr.Spec.ForProvider.DatacenterCfg.DatacenterID, networkLoadBalancerID)
	if err != nil {
		if errors.Is(err, networkloadbalancer.ErrNotFound) {
			return managed.ExternalObservation{}, nil
		}
		return managed.ExternalObservation{}, err
	}
	isLateInitialized := networkloadbalancer.LateInitializer(&cr.Spec.ForProvider, observed)
	networkloadbalancer.SetStatus(&cr.Status.AtProvider, observed)
	cr.Status.AtProvider.NetworkLoadBalancerID = networkLoadBalancerID
	listenerLanID, targetLanID, err := getConfiguredLanIDs(cr)
	if err != nil {
		return managed.ExternalObservation{}, err
	}
	publicIPs, err := c.getConfiguredIPs(ctx, cr)
	if err != nil {
		return managed.ExternalObservation{}, err
	}
	c.log.Debug(fmt.Sprintf("Observing state: %v", cr.Status.AtProvider.State))
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
	// Check external name in order to avoid duplicates,
	// since the creation requests take longer than other resources.
	if meta.GetExternalName(cr) != "" {
		return managed.ExternalCreation{}, nil
	}
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
	listenerLanID, targetLanID, err := getConfiguredLanIDs(cr)
	if err != nil {
		return managed.ExternalCreation{}, err
	}
	ips, err := c.getConfiguredIPs(ctx, cr)
	if err != nil {
		return managed.ExternalCreation{}, err
	}
	nlbInput := networkloadbalancer.GenerateCreateInput(cr, listenerLanID, targetLanID, ips)
	newInstance, err := c.service.CreateNetworkLoadBalancer(ctx, datacenterID, nlbInput)
	if err != nil {
		return managed.ExternalCreation{}, err
	}
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
	listenerLanID, targetLanID, err := getConfiguredLanIDs(cr)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}
	publicIPs, err := c.getConfiguredIPs(ctx, cr)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}
	nlbInput := networkloadbalancer.GenerateUpdateInput(cr, listenerLanID, targetLanID, publicIPs)
	_, err = c.service.UpdateNetworkLoadBalancer(ctx, datacenterID, nlbID, nlbInput)

	return managed.ExternalUpdate{}, err
}

func (c *externalNetworkLoadBalancer) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.NetworkLoadBalancer)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotNetworkLoadBalancer)
	}

	cr.SetConditions(xpv1.Deleting())
	if cr.Status.AtProvider.State == compute.DESTROYING {
		return managed.ExternalDelete{}, nil
	}
	datacenterID := cr.Spec.ForProvider.DatacenterCfg.DatacenterID
	nlbID := cr.Status.AtProvider.NetworkLoadBalancerID
	err := c.service.DeleteNetworkLoadBalancer(ctx, datacenterID, nlbID)
	if !errors.Is(err, networkloadbalancer.ErrNotFound) {
		return managed.ExternalDelete{}, err
	}
	return managed.ExternalDelete{}, nil
}

func (c *externalNetworkLoadBalancer) getConfiguredIPs(ctx context.Context, cr *v1alpha1.NetworkLoadBalancer) ([]string, error) {
	if len(cr.Spec.ForProvider.IpsCfg.IPs) == 0 && len(cr.Spec.ForProvider.IpsCfg.IPsBlocksCfg) == 0 {
		return nil, nil
	} else if len(cr.Spec.ForProvider.IpsCfg.IPs) > 0 {
		return cr.Spec.ForProvider.IpsCfg.IPs, nil
	}

	publicIPs := make([]string, 0, len(cr.Spec.ForProvider.IpsCfg.IPsBlocksCfg))
	for _, cfg := range cr.Spec.ForProvider.IpsCfg.IPsBlocksCfg {
		ips, err := c.ipBlockService.GetIPs(ctx, cfg.IPBlock.IPBlockID, cfg.Indexes...)
		if err != nil {
			return nil, err
		}
		publicIPs = append(publicIPs, ips...)
	}
	return publicIPs, nil
}

func getConfiguredLanIDs(cr *v1alpha1.NetworkLoadBalancer) (int32, int32, error) {
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

// Disconnect does nothing because there are no resources to release. Needs to be implemented starting from crossplane-runtime v0.17
func (c *externalNetworkLoadBalancer) Disconnect(_ context.Context) error {
	return nil
}
