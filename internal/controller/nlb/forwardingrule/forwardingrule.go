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
	"math/rand/v2"
	"time"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/nlb/v1alpha1"
	apisv1alpha1 "github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute/ipblock"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/nlb/forwardingrule"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"
)

const (
	errNotForwardingRule = "managed resource is not a NetworkLoadBalancer ForwardingRule"
	errGetListenerIP     = "failed to get forwarding rule listener ip: %w"
	errGetTargetsIPs     = "failed to get forwarding rule targets ips: %w"
)

// Setup adds a controller that reconciles ForwardingRule managed resources.
func Setup(mgr ctrl.Manager, opts *utils.ConfigurationOptions) error {
	name := managed.ControllerName(v1alpha1.ForwardingRuleGroupKind)
	logger := opts.CtrlOpts.Logger

	r := event.NewAPIRecorder(mgr.GetEventRecorderFor(name))
	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(opts.CtrlOpts.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.ForwardingRule{}).
		Complete(managed.NewReconciler(mgr,
			resource.ManagedKind(v1alpha1.ForwardingRuleGroupVersionKind),
			managed.WithExternalConnecter(&connectorForwardingRule{
				eventRecorder:        r,
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
				if mg.GetCondition(xpv1.TypeReady).Status != v1.ConditionTrue {
					// If the resource is not ready, we should poll more frequently not to delay time to readiness.
					pollInterval = 30 * time.Second
				}
				// This is the same as runtime default poll interval with jitter, see:
				// https://github.com/crossplane/crossplane-runtime/blob/7fcb8c5cad6fc4abb6649813b92ab92e1832d368/pkg/reconciler/managed/reconciler.go#L573
				return pollInterval + time.Duration((rand.Float64()-0.5)*2*float64(opts.PollJitter)) //nolint G404 // No need for secure randomness
			}),
			managed.WithRecorder(r)))
}

// A connectorForwardingRule is expected to produce an ExternalClient when its Connect method
// is called.
type connectorForwardingRule struct {
	eventRecorder        event.Recorder
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
		eventRecorder:        c.eventRecorder,
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
	eventRecorder        event.Recorder
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
	ruleID := meta.GetExternalName(cr)
	if ruleID == "" {
		return managed.ExternalObservation{}, nil
	}
	datacenterID := cr.Spec.ForProvider.DatacenterCfg.DatacenterID
	nlbID := cr.Spec.ForProvider.NLBCfg.NetworkLoadBalancerID
	observed, err := c.service.GetForwardingRuleByID(ctx, datacenterID, nlbID, ruleID)
	if err != nil {
		if errors.Is(err, forwardingrule.ErrNotFound) {
			return managed.ExternalObservation{}, nil
		}
		return managed.ExternalObservation{}, err
	}
	forwardingrule.SetStatus(&cr.Status.AtProvider, observed)
	cr.Status.AtProvider.ForwardingRuleID = ruleID

	listenerIP, err := c.getConfiguredListenerIP(ctx, cr)
	if err != nil {
		return managed.ExternalObservation{}, err
	}
	targetsIPs, err := c.getConfiguredTargetsIPs(ctx, cr)
	if err != nil {
		return managed.ExternalObservation{}, err
	}
	c.log.Debug(fmt.Sprintf("Observing state: %v", cr.Status.AtProvider.State))
	clients.UpdateCondition(cr, cr.Status.AtProvider.State)
	return managed.ExternalObservation{
		ResourceExists:    true,
		ResourceUpToDate:  forwardingrule.IsUpToDate(cr, observed, listenerIP, targetsIPs),
		ConnectionDetails: managed.ConnectionDetails{},
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
	if cr.Status.AtProvider.State == compute.BUSY {
		return managed.ExternalCreation{}, nil
	}

	datacenterID := cr.Spec.ForProvider.DatacenterCfg.DatacenterID
	nlbID := cr.Spec.ForProvider.NLBCfg.NetworkLoadBalancerID

	if c.isUniqueNamesEnabled {
		// isUniqueNamesEnabled option enforces ForwardingRule names to be unique per Datacenter and NetworkLoadBalancer
		// Multiple Forwarding Rules with the same name will trigger an error
		// If only one instance is found, it will be "imported"
		ruleDuplicateID, err := c.service.CheckDuplicateForwardingRule(ctx, datacenterID, nlbID, cr.Spec.ForProvider.Name)
		if err != nil {
			return managed.ExternalCreation{}, err
		}
		if ruleDuplicateID != "" {
			cr.Status.AtProvider.ForwardingRuleID = ruleDuplicateID
			meta.SetExternalName(cr, ruleDuplicateID)
			return managed.ExternalCreation{}, nil
		}
	}

	listenerIP, err := c.getConfiguredListenerIP(ctx, cr)
	if err != nil {
		return managed.ExternalCreation{}, err
	}
	targetsIPs, err := c.getConfiguredTargetsIPs(ctx, cr)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	ruleInput := forwardingrule.GenerateCreateInput(cr, listenerIP, targetsIPs)
	newInstance, err := c.service.CreateForwardingRule(ctx, datacenterID, nlbID, ruleInput)
	if err != nil {
		return managed.ExternalCreation{}, err
	}
	meta.SetExternalName(cr, *newInstance.Id)

	return managed.ExternalCreation{}, nil
}

func (c *externalForwardingRule) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.ForwardingRule)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotForwardingRule)
	}
	if cr.Status.AtProvider.State == compute.BUSY {
		return managed.ExternalUpdate{}, nil
	}

	datacenterID := cr.Spec.ForProvider.DatacenterCfg.DatacenterID
	nlbID := cr.Spec.ForProvider.NLBCfg.NetworkLoadBalancerID
	ruleID := cr.Status.AtProvider.ForwardingRuleID
	listenerIP, err := c.getConfiguredListenerIP(ctx, cr)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}
	targetsIPs, err := c.getConfiguredTargetsIPs(ctx, cr)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}
	ruleInput := forwardingrule.GenerateUpdateInput(cr, listenerIP, targetsIPs)
	_, err = c.service.UpdateForwardingRule(ctx, datacenterID, nlbID, ruleID, ruleInput)

	return managed.ExternalUpdate{}, err
}

func (c *externalForwardingRule) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.ForwardingRule)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotForwardingRule)
	}

	cr.SetConditions(xpv1.Deleting())
	if cr.Status.AtProvider.State == compute.DESTROYING {
		return managed.ExternalDelete{}, nil
	}
	datacenterID := cr.Spec.ForProvider.DatacenterCfg.DatacenterID
	nlbID := cr.Spec.ForProvider.NLBCfg.NetworkLoadBalancerID
	err := c.service.DeleteForwardingRule(ctx, datacenterID, nlbID, cr.Status.AtProvider.ForwardingRuleID)
	if !errors.Is(err, forwardingrule.ErrNotFound) {
		return managed.ExternalDelete{}, err
	}
	return managed.ExternalDelete{}, nil
}

func (c *externalForwardingRule) getConfiguredListenerIP(ctx context.Context, cr *v1alpha1.ForwardingRule) (string, error) {
	if cr.Spec.ForProvider.ListenerIP.IP != "" {
		return cr.Spec.ForProvider.ListenerIP.IP, nil
	}

	ip, err := c.ipBlockService.GetIPs(ctx, cr.Spec.ForProvider.ListenerIP.IPBlockID, int(cr.Spec.ForProvider.ListenerIP.Index))
	if err != nil {
		return "", fmt.Errorf(errGetListenerIP, err)
	}
	return ip[0], nil
}

func (c *externalForwardingRule) getConfiguredTargetsIPs(ctx context.Context, cr *v1alpha1.ForwardingRule) (map[string]v1alpha1.ForwardingRuleTarget, error) {
	if len(cr.Spec.ForProvider.Targets) == 0 {
		return nil, nil
	}

	targets := cr.Spec.ForProvider.Targets
	targetsMap := map[string]v1alpha1.ForwardingRuleTarget{}
	var ip string
	var duplicateTargetIP bool
	for _, t := range targets {
		ip = t.IPCfg.IP

		if ip == "" {
			ips, err := c.ipBlockService.GetIPs(ctx, t.IPCfg.IPBlockID, int(t.IPCfg.Index))
			if err != nil {
				return nil, fmt.Errorf(errGetTargetsIPs, err)
			}
			targetsMap[ips[0]] = t
			continue
		}

		// User specified IPs are deduplicated
		// Log and emit a warning if the same private IP is used for multiple targets
		if !duplicateTargetIP {
			if _, duplicateTargetIP = targetsMap[ip]; duplicateTargetIP {
				msg := fmt.Errorf("duplicate ip for forwarding rule targets: %s", ip)
				c.log.Info(msg.Error())
				c.eventRecorder.Event(cr, event.Warning("DuplicateTargetIp", msg))
			}
		}

		targetsMap[ip] = t
	}
	return targetsMap, nil
}

// Disconnect does nothing because there are no resources to release. Needs to be implemented starting from crossplane-runtime v0.17
func (c *externalForwardingRule) Disconnect(_ context.Context) error {
	return nil
}
