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

package firewallrule

import (
	"context"
	"fmt"
	"time"

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

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	apisv1alpha1 "github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute/firewallrule"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute/ipblock"
)

const errNotFirewallRule = "managed resource is not a FirewallRule custom resource"

// Setup adds a controller that reconciles FirewallRule managed resources.
func Setup(mgr ctrl.Manager, l logging.Logger, rl workqueue.RateLimiter, poll time.Duration, creationGracePeriod time.Duration) error {
	name := managed.ControllerName(v1alpha1.FirewallRuleGroupKind)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(controller.Options{
			RateLimiter: ratelimiter.NewDefaultManagedRateLimiter(rl),
		}).
		For(&v1alpha1.FirewallRule{}).
		Complete(managed.NewReconciler(mgr,
			resource.ManagedKind(v1alpha1.FirewallRuleGroupVersionKind),
			managed.WithExternalConnecter(&connectorFirewallRule{
				kube:  mgr.GetClient(),
				usage: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
				log:   l}),
			managed.WithPollInterval(poll),
			managed.WithCreationGracePeriod(creationGracePeriod),
			managed.WithReferenceResolver(managed.NewAPISimpleReferenceResolver(mgr.GetClient())),
			managed.WithLogger(l.WithValues("controller", name)),
			managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name)))))
}

// A connectorFirewallRule is expected to produce an ExternalClient when its Connect method
// is called.
type connectorFirewallRule struct {
	kube  client.Client
	usage resource.Tracker
	log   logging.Logger
}

// Connect typically produces an ExternalClient by:
// 1. Tracking that the managed resource is using a ProviderConfig.
// 2. Getting the managed resource's ProviderConfig.
// 3. Getting the credentials specified by the ProviderConfig.
// 4. Using the credentials to form a client.
func (c *connectorFirewallRule) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	_, ok := mg.(*v1alpha1.FirewallRule)
	if !ok {
		return nil, errors.New(errNotFirewallRule)
	}
	svc, err := clients.ConnectForCRD(ctx, mg, c.kube, c.usage)
	return &externalFirewallRule{
		service:        &firewallrule.APIClient{IonosServices: svc},
		ipBlockService: &ipblock.APIClient{IonosServices: svc},
		log:            c.log}, err
}

// An ExternalClient observes, then either creates, updates, or deletes an
// externalFirewallRule resource to ensure it reflects the managed resource's desired state.
type externalFirewallRule struct {
	// A 'client' used to connect to the externalFirewallRule resource API. In practice this
	// would be something like an IONOS Cloud SDK client.
	service        firewallrule.Client
	ipBlockService ipblock.Client
	log            logging.Logger
}

func (c *externalFirewallRule) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) { //nolint:gocyclo
	cr, ok := mg.(*v1alpha1.FirewallRule)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotFirewallRule)
	}

	// External Name of the CR is the FirewallRule ID
	if meta.GetExternalName(cr) == "" {
		return managed.ExternalObservation{}, nil
	}
	observed, apiResponse, err := c.service.GetFirewallRule(ctx, cr.Spec.ForProvider.DatacenterCfg.DatacenterID,
		cr.Spec.ForProvider.ServerCfg.ServerID, cr.Spec.ForProvider.NicCfg.NicID, meta.GetExternalName(cr))
	if err != nil {
		retErr := fmt.Errorf("failed to get firewallRule by id. error: %w", err)
		return managed.ExternalObservation{}, compute.CheckAPIResponseInfo(apiResponse, retErr)
	}

	cr.Status.AtProvider.FirewallRuleID = meta.GetExternalName(cr)
	if observed.HasProperties() {
		if observed.Properties.HasSourceIp() {
			cr.Status.AtProvider.SourceIP = *observed.Properties.SourceIp
		} else {
			cr.Status.AtProvider.SourceIP = ""
		}
		if observed.Properties.HasTargetIp() {
			cr.Status.AtProvider.TargetIP = *observed.Properties.TargetIp
		} else {
			cr.Status.AtProvider.TargetIP = ""
		}
	}
	if observed.HasMetadata() {
		if observed.Metadata.HasState() {
			cr.Status.AtProvider.State = *observed.Metadata.State
			c.log.Debug(fmt.Sprintf("Observing state: %v", cr.Status.AtProvider.State))
		}
	}
	// Set Ready condition based on State
	switch cr.Status.AtProvider.State {
	case compute.AVAILABLE, compute.ACTIVE:
		cr.SetConditions(xpv1.Available())
	case compute.BUSY, compute.UPDATING:
		cr.SetConditions(xpv1.Creating())
	case compute.DESTROYING:
		cr.SetConditions(xpv1.Deleting())
	default:
		cr.SetConditions(xpv1.Unavailable())
	}
	sourceIP, err := c.getSourceIPSet(ctx, cr)
	if err != nil {
		return managed.ExternalObservation{}, fmt.Errorf("error getting sourceIP: %v", sourceIP)
	}
	targetIP, err := c.getTargetIPSet(ctx, cr)
	if err != nil {
		return managed.ExternalObservation{}, fmt.Errorf("error getting targetIP: %v", targetIP)
	}
	return managed.ExternalObservation{
		ResourceExists:    true,
		ResourceUpToDate:  firewallrule.IsFirewallRuleUpToDate(cr, observed, sourceIP, targetIP),
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *externalFirewallRule) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.FirewallRule)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotFirewallRule)
	}

	cr.SetConditions(xpv1.Creating())
	if cr.Status.AtProvider.State == compute.BUSY {
		return managed.ExternalCreation{}, nil
	}
	sourceIP, err := c.getSourceIPSet(ctx, cr)
	if err != nil {
		return managed.ExternalCreation{}, fmt.Errorf("error getting sourceIP: %v", sourceIP)
	}
	targetIP, err := c.getTargetIPSet(ctx, cr)
	if err != nil {
		return managed.ExternalCreation{}, fmt.Errorf("error getting targetIP: %v", targetIP)
	}
	instanceInput, err := firewallrule.GenerateCreateFirewallRuleInput(cr, sourceIP, targetIP)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	instance, apiResponse, err := c.service.CreateFirewallRule(ctx, cr.Spec.ForProvider.DatacenterCfg.DatacenterID,
		cr.Spec.ForProvider.ServerCfg.ServerID, cr.Spec.ForProvider.NicCfg.NicID, *instanceInput)
	creation := managed.ExternalCreation{ConnectionDetails: managed.ConnectionDetails{}}
	if err != nil {
		retErr := fmt.Errorf("failed to create firewallRule. error: %w", err)
		return creation, compute.AddAPIResponseInfo(apiResponse, retErr)
	}
	if err = compute.WaitForRequest(ctx, c.service.GetAPIClient(), apiResponse); err != nil {
		return creation, err
	}

	// Set External Name
	cr.Status.AtProvider.FirewallRuleID = *instance.Id
	meta.SetExternalName(cr, *instance.Id)
	return creation, nil
}

func (c *externalFirewallRule) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.FirewallRule)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotFirewallRule)
	}
	if cr.Status.AtProvider.State == compute.BUSY || cr.Status.AtProvider.State == compute.UPDATING {
		return managed.ExternalUpdate{}, nil
	}

	firewallRuleID := cr.Status.AtProvider.FirewallRuleID
	sourceIP, err := c.getSourceIPSet(ctx, cr)
	if err != nil {
		return managed.ExternalUpdate{}, fmt.Errorf("error getting sourceIP: %v", sourceIP)
	}
	targetIP, err := c.getTargetIPSet(ctx, cr)
	if err != nil {
		return managed.ExternalUpdate{}, fmt.Errorf("error getting targetIP: %v", targetIP)
	}
	instanceInput, err := firewallrule.GenerateUpdateFirewallRuleInput(cr, sourceIP, targetIP)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	_, apiResponse, err := c.service.UpdateFirewallRule(ctx, cr.Spec.ForProvider.DatacenterCfg.DatacenterID,
		cr.Spec.ForProvider.ServerCfg.ServerID, cr.Spec.ForProvider.NicCfg.NicID, firewallRuleID, *instanceInput)
	if err != nil {
		retErr := fmt.Errorf("failed to update firewallRule. error: %w", err)
		return managed.ExternalUpdate{}, compute.AddAPIResponseInfo(apiResponse, retErr)
	}
	if err = compute.WaitForRequest(ctx, c.service.GetAPIClient(), apiResponse); err != nil {
		return managed.ExternalUpdate{}, err
	}
	return managed.ExternalUpdate{}, nil
}

func (c *externalFirewallRule) Delete(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*v1alpha1.FirewallRule)
	if !ok {
		return errors.New(errNotFirewallRule)
	}

	cr.SetConditions(xpv1.Deleting())
	if cr.Status.AtProvider.State == compute.DESTROYING {
		return nil
	}

	apiResponse, err := c.service.DeleteFirewallRule(ctx, cr.Spec.ForProvider.DatacenterCfg.DatacenterID,
		cr.Spec.ForProvider.ServerCfg.ServerID, cr.Spec.ForProvider.NicCfg.NicID, cr.Status.AtProvider.FirewallRuleID)
	if err != nil {
		retErr := fmt.Errorf("failed to delete firewallRule. error: %w", err)
		return compute.AddAPIResponseInfo(apiResponse, retErr)
	}
	if err = compute.WaitForRequest(ctx, c.service.GetAPIClient(), apiResponse); err != nil {
		return err
	}
	return nil
}

// getTargetIPSet will return the TargetIP set by the user on targetIpConfig.ip or
// targetIpConfig.ipBlockConfig fields of the spec.
// If both fields are set, only the targetIpConfig.ip field will be considered by
// the Crossplane Provider IONOS Cloud.
func (c *externalFirewallRule) getTargetIPSet(ctx context.Context, cr *v1alpha1.FirewallRule) (string, error) {
	if cr.Spec.ForProvider.TargetIPCfg.IP != "" {
		return cr.Spec.ForProvider.TargetIPCfg.IP, nil
	}
	if cr.Spec.ForProvider.TargetIPCfg.IPBlockCfg.IPBlockID != "" {
		ipsCfg, err := c.ipBlockService.GetIPs(ctx, cr.Spec.ForProvider.TargetIPCfg.IPBlockCfg.IPBlockID, cr.Spec.ForProvider.TargetIPCfg.IPBlockCfg.Index)
		if err != nil {
			return "", err
		}
		if len(ipsCfg) != 1 {
			return "", fmt.Errorf("error getting target IP with index %v from IPBlock %v",
				cr.Spec.ForProvider.TargetIPCfg.IPBlockCfg.Index, cr.Spec.ForProvider.TargetIPCfg.IPBlockCfg.IPBlockID)
		}
		return ipsCfg[0], nil
	}
	// return nil if nothing is set,
	// since TargetIP can be empty
	return "", nil
}

// getSourceIPSet will return the SourceIP set by the user on sourceIpConfig.ip or
// sourceIpConfig.ipBlockConfig fields of the spec.
// If both fields are set, only the sourceIpConfig.ip field will be considered by
// the Crossplane Provider IONOS Cloud.
func (c *externalFirewallRule) getSourceIPSet(ctx context.Context, cr *v1alpha1.FirewallRule) (string, error) {
	if cr.Spec.ForProvider.SourceIPCfg.IP != "" {
		return cr.Spec.ForProvider.TargetIPCfg.IP, nil
	}
	if cr.Spec.ForProvider.SourceIPCfg.IPBlockCfg.IPBlockID != "" {
		ipsCfg, err := c.ipBlockService.GetIPs(ctx, cr.Spec.ForProvider.SourceIPCfg.IPBlockCfg.IPBlockID, cr.Spec.ForProvider.SourceIPCfg.IPBlockCfg.Index)
		if err != nil {
			return "", err
		}
		if len(ipsCfg) != 1 {
			return "", fmt.Errorf("error getting source IP with index %v from IPBlock %v",
				cr.Spec.ForProvider.SourceIPCfg.IPBlockCfg.Index, cr.Spec.ForProvider.SourceIPCfg.IPBlockCfg.IPBlockID)
		}
		return ipsCfg[0], nil
	}
	// return nil if nothing is set,
	// since SourceIP can be empty
	return "", nil
}
