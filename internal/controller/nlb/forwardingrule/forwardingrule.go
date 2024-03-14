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
	ionoscloud "github.com/ionos-cloud/sdk-go-dbaas-postgres"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/nlb/v1alpha1"
	apisv1alpha1 "github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute/ipblock"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/nlb/forwardingrule"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"
)

const errNotForwardingRule = "managed resource is not a NetworkLoadBalancer ForwardingRule"

// Setup adds a controller that reconciles ForwardingRule managed resources.
func Setup(mgr ctrl.Manager, l logging.Logger, rl workqueue.RateLimiter, opts *utils.ConfigurationOptions) error {
	name := managed.ControllerName(v1alpha1.ForwardingRuleGroupKind)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(controller.Options{
			RateLimiter: ratelimiter.NewController(),
		}).
		For(&v1alpha1.ForwardingRule{}).
		Complete(managed.NewReconciler(mgr,
			resource.ManagedKind(v1alpha1.ForwardingRuleGroupVersionKind),
			managed.WithExternalConnecter(&connectorForwardingRule{
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
	ruleID := meta.GetExternalName(cr)
	if ruleID == "" {
		return managed.ExternalObservation{}, nil
	}
	datacenterID := cr.Spec.ForProvider.DatacenterCfg.DatacenterID
	nlbID := cr.Spec.ForProvider.NLBCfg.NetworkLoadBalancerID
	observed, _, err := c.service.GetForwardingRuleByID(ctx, datacenterID, nlbID, ruleID)
	if err != nil {
		if errors.Is(err, forwardingrule.ErrNotFound) {
			return managed.ExternalObservation{}, nil
		}
		return managed.ExternalObservation{}, err
	}
	isLateInitialized := forwardingrule.LateInitializer(&cr.Spec.ForProvider, observed)
	forwardingrule.SetStatus(&cr.Status.AtProvider, observed)
	cr.Status.AtProvider.ForwardingRuleID = ruleID

	listenerIP, err := c.getConfiguredListenerIP(ctx, cr)
	if err != nil {
		return managed.ExternalObservation{}, err
	}
	targetIPs, err := c.getConfiguredTargetsIPs(ctx, cr)
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	clients.UpdateCondition(cr, cr.Status.AtProvider.State)
	return managed.ExternalObservation{
		ResourceExists:          true,
		ResourceUpToDate:        forwardingrule.IsUpToDate(cr, observed, listenerIP, targetIPs),
		ConnectionDetails:       managed.ConnectionDetails{},
		ResourceLateInitialized: isLateInitialized,
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
	if cr.Status.AtProvider.State == string(ionoscloud.BUSY) {
		return managed.ExternalCreation{}, nil
	}

	if c.isUniqueNamesEnabled {
	}
	return managed.ExternalCreation{}, nil
}

func (c *externalForwardingRule) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.ForwardingRule)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotForwardingRule)
	}
	if cr.Status.AtProvider.State == string(ionoscloud.BUSY) {
		return managed.ExternalUpdate{}, nil
	}

	return managed.ExternalUpdate{}, nil
}

func (c *externalForwardingRule) Delete(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*v1alpha1.ForwardingRule)
	if !ok {
		return errors.New(errNotForwardingRule)
	}

	cr.SetConditions(xpv1.Deleting())
	if cr.Status.AtProvider.State == string(ionoscloud.DESTROYING) || cr.Status.AtProvider.State == string(ionoscloud.BUSY) {
		return nil
	}

	return nil
}

func (c *externalForwardingRule) getConfiguredListenerIP(ctx context.Context, cr *v1alpha1.ForwardingRule) (string, error) {
	if cr.Spec.ForProvider.ListenerIP.IP != "" {
		return cr.Spec.ForProvider.ListenerIP.IP, nil
	}

	ip, err := c.ipBlockService.GetIPs(ctx, cr.Spec.ForProvider.ListenerIP.IPBlockID, int(cr.Spec.ForProvider.ListenerIP.Index))
	if err != nil {
		return "", err
	}
	return ip[0], nil
}

func (c *externalForwardingRule) getConfiguredTargetsIPs(ctx context.Context, cr *v1alpha1.ForwardingRule) (map[string]v1alpha1.ForwardingRuleTarget, error) {
	if len(cr.Spec.ForProvider.Targets) == 0 {
		return nil, nil
	}

	targets := cr.Spec.ForProvider.Targets
	targetsMap := map[string]v1alpha1.ForwardingRuleTarget{}
	for _, t := range targets {
		if t.IPCfg.IP != "" {
			targetsMap[t.IPCfg.IP] = t
			continue
		}
		ip, err := c.ipBlockService.GetIPs(ctx, t.IPCfg.IPBlockID, int(t.IPCfg.Index))
		if err != nil {
			return nil, err
		}
		targetsMap[ip[0]] = t
	}
	return targetsMap, nil
}
