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

package applicationloadbalancer

import (
	"context"
	"fmt"

	"github.com/crossplane/crossplane-runtime/pkg/statemetrics"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	"github.com/rung/go-safecast"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"

	ionoscloud "github.com/ionos-cloud/sdk-go-bundle/products/dbaas/psql/v2"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/alb/v1alpha1"
	apisv1alpha1 "github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/alb/applicationloadbalancer"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute/ipblock"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"
)

const errNotApplicationLoadBalancer = "managed resource is not a ApplicationLoadBalancer custom resource"

// Setup adds a controller that reconciles ApplicationLoadBalancer managed resources.
func Setup(mgr ctrl.Manager, opts *utils.ConfigurationOptions) error {
	name := managed.ControllerName(v1alpha1.ApplicationLoadBalancerGroupKind)
	logger := opts.CtrlOpts.Logger
	if opts.CtrlOpts.MetricOptions != nil && opts.CtrlOpts.MetricOptions.MRStateMetrics != nil {
		stateMetricsRecorder := statemetrics.NewMRStateRecorder(
			mgr.GetClient(), opts.CtrlOpts.Logger, opts.CtrlOpts.MetricOptions.MRStateMetrics, &v1alpha1.ApplicationLoadBalancerList{}, opts.CtrlOpts.MetricOptions.PollStateMetricInterval,
		)
		if err := mgr.Add(stateMetricsRecorder); err != nil {
			return errors.Wrap(err, "cannot register MR state metrics recorder for kind"+name)
		}
	}
	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(opts.CtrlOpts.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.ApplicationLoadBalancer{}).
		Complete(managed.NewReconciler(mgr,
			resource.ManagedKind(v1alpha1.ApplicationLoadBalancerGroupVersionKind),
			managed.WithExternalConnecter(&connectorApplicationLoadBalancer{
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

// A connectorApplicationLoadBalancer is expected to produce an ExternalClient when its Connect method
// is called.
type connectorApplicationLoadBalancer struct {
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
func (c *connectorApplicationLoadBalancer) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	_, ok := mg.(*v1alpha1.ApplicationLoadBalancer)
	if !ok {
		return nil, errors.New(errNotApplicationLoadBalancer)
	}
	svc, err := clients.ConnectForCRD(ctx, mg, c.kube, c.usage)
	return &externalApplicationLoadBalancer{
		service:              &applicationloadbalancer.APIClient{IonosServices: svc},
		ipBlockService:       &ipblock.APIClient{IonosServices: svc},
		log:                  c.log,
		isUniqueNamesEnabled: c.isUniqueNamesEnabled}, err
}

// An ExternalClient observes, then either creates, updates, or deletes an
// externalApplicationLoadBalancer resource to ensure it reflects the managed resource's desired state.
type externalApplicationLoadBalancer struct {
	// A 'client' used to connect to the externalApplicationLoadBalancer resource API. In practice this
	// would be something like an IONOS Cloud SDK client.
	service              applicationloadbalancer.Client
	ipBlockService       ipblock.Client
	log                  logging.Logger
	isUniqueNamesEnabled bool
}

func (c *externalApplicationLoadBalancer) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) { // nolint:gocyclo
	cr, ok := mg.(*v1alpha1.ApplicationLoadBalancer)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotApplicationLoadBalancer)
	}

	if meta.GetExternalName(cr) == "" {
		return managed.ExternalObservation{}, nil
	}
	observed, resp, err := c.service.GetApplicationLoadBalancer(ctx, cr.Spec.ForProvider.DatacenterCfg.DatacenterID, meta.GetExternalName(cr))
	if err != nil {
		retErr := fmt.Errorf("failed to get application load balancer by id. error: %w", err)
		return managed.ExternalObservation{}, compute.ErrorUnlessNotFound(resp, retErr)
	}
	current := cr.Spec.ForProvider.DeepCopy()
	applicationloadbalancer.LateInitializer(&cr.Spec.ForProvider, &observed)
	cr.Status.AtProvider.ApplicationLoadBalancerID = meta.GetExternalName(cr)
	cr.Status.AtProvider.State = clients.GetCoreResourceState(&observed)
	if observed.HasProperties() && observed.Properties.HasIps() {
		cr.Status.AtProvider.PublicIPs = *observed.Properties.Ips
	}
	c.log.Debug(fmt.Sprintf("Observing state: %v", cr.Status.AtProvider.State))
	clients.UpdateCondition(cr, cr.Status.AtProvider.State)
	// Resolve IPs and lan IDs
	ips, err := c.getIPsSet(ctx, cr)
	if err != nil {
		return managed.ExternalObservation{}, err
	}
	listenerLanID, err := safecast.Atoi32(cr.Spec.ForProvider.ListenerLanCfg.LanID)
	if err != nil {
		return managed.ExternalObservation{}, err
	}
	targetLanID, err := safecast.Atoi32(cr.Spec.ForProvider.TargetLanCfg.LanID)
	if err != nil {
		return managed.ExternalObservation{}, err
	}
	// Check ExternalObservation
	return managed.ExternalObservation{
		ResourceExists:          true,
		ResourceUpToDate:        applicationloadbalancer.IsApplicationLoadBalancerUpToDate(cr, observed, listenerLanID, targetLanID, ips),
		ConnectionDetails:       managed.ConnectionDetails{},
		ResourceLateInitialized: !cmp.Equal(current, &cr.Spec.ForProvider),
	}, nil
}

func (c *externalApplicationLoadBalancer) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) { // nolint: gocyclo
	cr, ok := mg.(*v1alpha1.ApplicationLoadBalancer)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotApplicationLoadBalancer)
	}
	cr.SetConditions(xpv1.Creating())
	// Check external name in order to avoid duplicates,
	// since the creation requests take longer than other resources.
	if meta.GetExternalName(cr) != "" {
		return managed.ExternalCreation{}, nil
	}
	if cr.Status.AtProvider.State == string(ionoscloud.STATE_BUSY) {
		return managed.ExternalCreation{}, nil
	}

	if c.isUniqueNamesEnabled {
		// ApplicationLoadBalancers should have unique names per datacenter.
		// Check if there are any existing application load balancers with the same name.
		// If there are multiple, an error will be returned.
		instance, err := c.service.CheckDuplicateApplicationLoadBalancer(ctx, cr.Spec.ForProvider.DatacenterCfg.DatacenterID, cr.Spec.ForProvider.Name)
		if err != nil {
			return managed.ExternalCreation{}, err
		}
		applicationLoadBalancerID, err := c.service.GetApplicationLoadBalancerID(instance)
		if err != nil {
			return managed.ExternalCreation{}, err
		}
		if applicationLoadBalancerID != "" {
			// "Import" existing application load balancer.
			cr.Status.AtProvider.ApplicationLoadBalancerID = applicationLoadBalancerID
			meta.SetExternalName(cr, applicationLoadBalancerID)
			return managed.ExternalCreation{}, nil
		}
	}

	ips, err := c.getIPsSet(ctx, cr)
	if err != nil {
		return managed.ExternalCreation{}, fmt.Errorf("failed to get ips: %w", err)
	}
	instanceInput, err := applicationloadbalancer.GenerateCreateApplicationLoadBalancerInput(cr, ips)
	if err != nil {
		return managed.ExternalCreation{}, err
	}
	newInstance, apiResponse, err := c.service.CreateApplicationLoadBalancer(ctx, cr.Spec.ForProvider.DatacenterCfg.DatacenterID, *instanceInput)
	creation := managed.ExternalCreation{ConnectionDetails: managed.ConnectionDetails{}}
	if err != nil {
		retErr := fmt.Errorf("failed to create application load balancer: %w", err)
		if apiResponse != nil && apiResponse.Response != nil {
			retErr = fmt.Errorf("%w API Response Status: %v", retErr, apiResponse.Status)
		}
		return creation, retErr
	}
	// Set External Name
	cr.Status.AtProvider.ApplicationLoadBalancerID = *newInstance.Id
	meta.SetExternalName(cr, *newInstance.Id)
	return creation, nil
}

func (c *externalApplicationLoadBalancer) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.ApplicationLoadBalancer)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotApplicationLoadBalancer)
	}
	if cr.Status.AtProvider.State == string(ionoscloud.STATE_BUSY) {
		return managed.ExternalUpdate{}, nil
	}

	ips, err := c.getIPsSet(ctx, cr)
	if err != nil {
		return managed.ExternalUpdate{}, fmt.Errorf("failed to get ips: %w", err)
	}
	instanceInput, err := applicationloadbalancer.GenerateUpdateApplicationLoadBalancerInput(cr, ips)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}
	_, apiResponse, err := c.service.UpdateApplicationLoadBalancer(ctx, cr.Spec.ForProvider.DatacenterCfg.DatacenterID,
		cr.Status.AtProvider.ApplicationLoadBalancerID, *instanceInput)
	if err != nil {
		retErr := fmt.Errorf("failed to update application load balancer: %w", err)
		if apiResponse != nil && apiResponse.Response != nil {
			retErr = fmt.Errorf("%w API Response Status: %v", retErr, apiResponse.Status)
		}
		return managed.ExternalUpdate{}, retErr
	}
	if err = compute.WaitForRequest(ctx, c.service.GetAPIClient(), apiResponse); err != nil {
		return managed.ExternalUpdate{}, err
	}
	return managed.ExternalUpdate{}, nil
}

func (c *externalApplicationLoadBalancer) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.ApplicationLoadBalancer)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotApplicationLoadBalancer)
	}

	cr.SetConditions(xpv1.Deleting())
	if cr.Status.AtProvider.State == string(ionoscloud.STATE_DESTROYING) || cr.Status.AtProvider.State == string(ionoscloud.STATE_BUSY) {
		return managed.ExternalDelete{}, nil
	}
	apiResponse, err := c.service.DeleteApplicationLoadBalancer(ctx, cr.Spec.ForProvider.DatacenterCfg.DatacenterID, cr.Status.AtProvider.ApplicationLoadBalancerID)
	if err != nil {
		retErr := fmt.Errorf("failed to delete application load balancer. error: %w", err)
		return managed.ExternalDelete{}, compute.ErrorUnlessNotFound(apiResponse, retErr)
	}
	if err = compute.WaitForRequest(ctx, c.service.GetAPIClient(), apiResponse); err != nil {
		return managed.ExternalDelete{}, err
	}
	return managed.ExternalDelete{}, nil
}

func (c *externalApplicationLoadBalancer) getIPsSet(ctx context.Context, cr *v1alpha1.ApplicationLoadBalancer) ([]string, error) {
	if len(cr.Spec.ForProvider.IpsCfg.IPs) == 0 && len(cr.Spec.ForProvider.IpsCfg.IPBlockCfgs) == 0 {
		return nil, nil
	}
	if len(cr.Spec.ForProvider.IpsCfg.IPs) > 0 {
		return cr.Spec.ForProvider.IpsCfg.IPs, nil
	}
	ips := make([]string, 0)
	if len(cr.Spec.ForProvider.IpsCfg.IPBlockCfgs) > 0 {
		for _, cfg := range cr.Spec.ForProvider.IpsCfg.IPBlockCfgs {
			ipsCfg, err := c.ipBlockService.GetIPs(ctx, cfg.IPBlockID, cfg.Indexes...)
			if err != nil {
				return nil, err
			}
			ips = append(ips, ipsCfg...)
		}
	}
	return ips, nil
}

// Disconnect does nothing because there are no resources to release. Needs to be implemented starting from crossplane-runtime v0.17
func (c *externalApplicationLoadBalancer) Disconnect(_ context.Context) error {
	return nil
}
