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

package forwardingrule

import (
	"context"
	"fmt"
	"time"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/google/go-cmp/cmp"
	ionoscloud "github.com/ionos-cloud/sdk-go-bundle/products/dbaas/psql/v2"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/alb/v1alpha1"
	apisv1alpha1 "github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/alb/forwardingrule"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute/ipblock"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"
)

const errNotForwardingRule = "managed resource is not a ApplicationLoadBalancer ForwardingRule custom resource"

// Setup adds a controller that reconciles ForwardingRule managed resources.
func Setup(mgr ctrl.Manager, opts *utils.ConfigurationOptions) error {
	name := managed.ControllerName(v1alpha1.ForwardingRuleGroupKind)
	logger := opts.CtrlOpts.Logger

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(opts.CtrlOpts.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.ForwardingRule{}).
		Complete(managed.NewReconciler(mgr,
			resource.ManagedKind(v1alpha1.ForwardingRuleGroupVersionKind),
			managed.WithExternalConnecter(&connectorForwardingRule{
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
			managed.WithPollIntervalHook(func(mg resource.Managed, pollInterval time.Duration) time.Duration {
				return utils.CalculatePollInterval(mg, pollInterval, opts.PollJitter)
			}),
			managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name)))))
}

// A connectorForwardingRule is expected to produce an ExternalClient when its Connect method
// is called.
type connectorForwardingRule struct {
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
func (c *connectorForwardingRule) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	_, ok := mg.(*v1alpha1.ForwardingRule)
	if !ok {
		return nil, errors.New(errNotForwardingRule)
	}
	svc, err := clients.ConnectForCRD(ctx, mg, c.kube, c.usage)
	return &externalForwardingRule{
		service:              &forwardingrule.APIClient{IonosServices: svc},
		ipBlockService:       &ipblock.APIClient{IonosServices: svc},
		log:                  c.log,
		isUniqueNamesEnabled: c.isUniqueNamesEnabled}, err
}

// An ExternalClient observes, then either creates, updates, or deletes an
// externalForwardingRule resource to ensure it reflects the managed resource's desired state.
type externalForwardingRule struct {
	// A 'client' used to connect to the externalForwardingRule resource API. In practice this
	// would be something like an IONOS Cloud SDK client.
	service              forwardingrule.Client
	ipBlockService       ipblock.Client
	log                  logging.Logger
	isUniqueNamesEnabled bool
}

func (c *externalForwardingRule) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) { // nolint:gocyclo
	cr, ok := mg.(*v1alpha1.ForwardingRule)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotForwardingRule)
	}

	if meta.GetExternalName(cr) == "" {
		return managed.ExternalObservation{}, nil
	}
	observed, apiResponse, err := c.service.GetForwardingRule(ctx, cr.Spec.ForProvider.DatacenterCfg.DatacenterID,
		cr.Spec.ForProvider.ALBCfg.ApplicationLoadBalancerID, meta.GetExternalName(cr))
	if err != nil {
		retErr := fmt.Errorf("failed to get application load balancer forwarding rule by id. error: %w", err)
		return managed.ExternalObservation{}, compute.ErrorUnlessNotFound(apiResponse, retErr)
	}
	current := cr.Spec.ForProvider.DeepCopy()
	forwardingrule.LateInitializer(&cr.Spec.ForProvider, &observed)
	cr.Status.AtProvider.ForwardingRuleID = meta.GetExternalName(cr)
	cr.Status.AtProvider.State = clients.GetCoreResourceState(&observed)
	c.log.Debug(fmt.Sprintf("Observing state: %v", cr.Status.AtProvider.State))
	clients.UpdateCondition(cr, cr.Status.AtProvider.State)
	// Resolve IPs
	listenerIP, err := c.getIPSet(ctx, cr)
	if err != nil {
		return managed.ExternalObservation{}, err
	}
	return managed.ExternalObservation{
		ResourceExists:          true,
		ResourceUpToDate:        forwardingrule.IsForwardingRuleUpToDate(cr, observed, listenerIP),
		ConnectionDetails:       managed.ConnectionDetails{},
		ResourceLateInitialized: !cmp.Equal(current, &cr.Spec.ForProvider),
	}, nil
}

func (c *externalForwardingRule) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) { // nolint: gocyclo
	cr, ok := mg.(*v1alpha1.ForwardingRule)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotForwardingRule)
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
		// ForwardingRules should have unique names per application load balancer.
		// Check if there are any existing forwarding rules with the same name.
		// If there are multiple, an error will be returned.
		instance, err := c.service.CheckDuplicateForwardingRule(ctx, cr.Spec.ForProvider.DatacenterCfg.DatacenterID,
			cr.Spec.ForProvider.ALBCfg.ApplicationLoadBalancerID, cr.Spec.ForProvider.Name)
		if err != nil {
			return managed.ExternalCreation{}, err
		}
		forwardingRuleID, err := c.service.GetForwardingRuleID(instance)
		if err != nil {
			return managed.ExternalCreation{}, err
		}
		if forwardingRuleID != "" {
			// "Import" existing forwarding rule.
			cr.Status.AtProvider.ForwardingRuleID = forwardingRuleID
			meta.SetExternalName(cr, forwardingRuleID)
			return managed.ExternalCreation{}, nil
		}
	}

	listenerIP, err := c.getIPSet(ctx, cr)
	if err != nil {
		return managed.ExternalCreation{}, fmt.Errorf("failed to get listener ip: %w", err)
	}
	instanceInput, err := forwardingrule.GenerateCreateForwardingRuleInput(cr, listenerIP)
	if err != nil {
		return managed.ExternalCreation{}, err
	}
	newInstance, apiResponse, err := c.service.CreateForwardingRule(ctx, cr.Spec.ForProvider.DatacenterCfg.DatacenterID,
		cr.Spec.ForProvider.ALBCfg.ApplicationLoadBalancerID, *instanceInput)
	creation := managed.ExternalCreation{ConnectionDetails: managed.ConnectionDetails{}}
	if err != nil {
		retErr := fmt.Errorf("failed to create application load balancer forwarding rule: %w", err)
		if apiResponse != nil && apiResponse.Response != nil {
			retErr = fmt.Errorf("%w API Response Status: %v", retErr, apiResponse.Status)
		}
		return creation, retErr
	}
	// Set External Name
	cr.Status.AtProvider.ForwardingRuleID = *newInstance.Id
	meta.SetExternalName(cr, *newInstance.Id)
	return creation, nil
}

func (c *externalForwardingRule) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.ForwardingRule)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotForwardingRule)
	}
	if cr.Status.AtProvider.State == string(ionoscloud.STATE_BUSY) {
		return managed.ExternalUpdate{}, nil
	}
	listenerIP, err := c.getIPSet(ctx, cr)
	if err != nil {
		return managed.ExternalUpdate{}, fmt.Errorf("failed to get listener ip: %w", err)
	}
	instanceInput, err := forwardingrule.GenerateUpdateForwardingRuleInput(cr, listenerIP)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}
	_, apiResponse, err := c.service.UpdateForwardingRule(ctx, cr.Spec.ForProvider.DatacenterCfg.DatacenterID,
		cr.Spec.ForProvider.ALBCfg.ApplicationLoadBalancerID, cr.Status.AtProvider.ForwardingRuleID, *instanceInput)
	if err != nil {
		retErr := fmt.Errorf("failed to update application load balancer forwarding rule: %w", err)
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

func (c *externalForwardingRule) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.ForwardingRule)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotForwardingRule)
	}
	cr.SetConditions(xpv1.Deleting())
	if cr.Status.AtProvider.State == string(ionoscloud.STATE_DESTROYING) || cr.Status.AtProvider.State == string(ionoscloud.STATE_BUSY) {
		return managed.ExternalDelete{}, nil
	}
	apiResponse, err := c.service.DeleteForwardingRule(ctx, cr.Spec.ForProvider.DatacenterCfg.DatacenterID,
		cr.Spec.ForProvider.ALBCfg.ApplicationLoadBalancerID, cr.Status.AtProvider.ForwardingRuleID)
	if err != nil {
		retErr := fmt.Errorf("failed to delete application load balancer forwarding rule. error: %w", err)
		return managed.ExternalDelete{}, compute.ErrorUnlessNotFound(apiResponse, retErr)
	}
	if err = compute.WaitForRequest(ctx, c.service.GetAPIClient(), apiResponse); err != nil {
		return managed.ExternalDelete{}, err
	}
	return managed.ExternalDelete{}, nil
}

func (c *externalForwardingRule) getIPSet(ctx context.Context, cr *v1alpha1.ForwardingRule) (string, error) {
	if len(cr.Spec.ForProvider.ListenerIP.IP) == 0 && len(cr.Spec.ForProvider.ListenerIP.IPBlockCfg.IPBlockID) == 0 {
		return "", nil
	}
	if len(cr.Spec.ForProvider.ListenerIP.IP) > 0 {
		return cr.Spec.ForProvider.ListenerIP.IP, nil
	}
	if len(cr.Spec.ForProvider.ListenerIP.IPBlockCfg.IPBlockID) > 0 {
		ipsCfg, err := c.ipBlockService.GetIPs(ctx, cr.Spec.ForProvider.ListenerIP.IPBlockCfg.IPBlockID, cr.Spec.ForProvider.ListenerIP.IPBlockCfg.Index)
		if err != nil {
			return "", err
		}
		return ipsCfg[0], nil
	}
	return "", nil
}

// Disconnect does nothing because there are no resources to release. Needs to be implemented starting from crossplane-runtime v0.17
func (c *externalForwardingRule) Disconnect(_ context.Context) error {
	return nil
}
