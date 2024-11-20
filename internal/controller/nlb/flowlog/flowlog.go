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

package flowlog

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

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
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/flowlog"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"
)

var errNotFlowLog = errors.New("managed resource is not a NetworkLoadBalancer FlowLog")

// Setup adds a controller that reconciles FlowLog managed resources.
func Setup(mgr ctrl.Manager, opts *utils.ConfigurationOptions) error {
	name := managed.ControllerName(v1alpha1.FlowLogGroupKind)
	logger := opts.CtrlOpts.Logger

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(controller.Options{
			RateLimiter: ratelimiter.NewController(),
		}).
		For(&v1alpha1.FlowLog{}).
		Complete(managed.NewReconciler(mgr,
			resource.ManagedKind(v1alpha1.FlowLogGroupVersionKind),
			managed.WithExternalConnecter(&connectorFlowLog{
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
			managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name)))))
}

// A connectorFlowLog is expected to produce an ExternalClient when its Connect method
// is called.
type connectorFlowLog struct {
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
func (c *connectorFlowLog) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	_, ok := mg.(*v1alpha1.FlowLog)
	if !ok {
		return nil, errNotFlowLog
	}
	svc, err := clients.ConnectForCRD(ctx, mg, c.kube, c.usage)
	return &externalFlowLog{
		service:              flowlog.NLBClient(svc),
		log:                  c.log,
		isUniqueNamesEnabled: c.isUniqueNamesEnabled}, err
}

// An ExternalClient observes, then either creates, updates, or deletes an
// externalFlowLog resource to ensure it reflects the managed resource's desired state.
type externalFlowLog struct {
	// A 'client' used to connect to the externalFlowLog resource API. In practice this
	// would be something like an IONOS Cloud SDK client.
	service              flowlog.NLBFlowLog
	log                  logging.Logger
	isUniqueNamesEnabled bool
}

func (c *externalFlowLog) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) { // nolint:gocyclo
	cr, ok := mg.(*v1alpha1.FlowLog)
	if !ok {
		return managed.ExternalObservation{}, errNotFlowLog
	}
	flowLogID := meta.GetExternalName(cr)
	if flowLogID == "" {
		return managed.ExternalObservation{}, nil
	}

	datacenterID := cr.Spec.ForProvider.DatacenterCfg.DatacenterID
	nlbID := cr.Spec.ForProvider.NLBCfg.NetworkLoadBalancerID
	observed, err := c.service.GetFlowLogByID(ctx, datacenterID, nlbID, flowLogID)
	if err != nil {
		if errors.Is(err, flowlog.ErrNotFound) {
			return managed.ExternalObservation{}, nil
		}
		return managed.ExternalObservation{}, err
	}
	flowlog.SetStatus(cr, observed)
	cr.Status.AtProvider.FlowLogID = flowLogID
	c.log.Debug(fmt.Sprintf("Observing state: %v", cr.Status.AtProvider.State))
	clients.UpdateCondition(cr, cr.Status.AtProvider.State)
	return managed.ExternalObservation{
		ResourceExists:    true,
		ResourceUpToDate:  flowlog.IsUpToDate(cr, observed),
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *externalFlowLog) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) { // nolint: gocyclo
	cr, ok := mg.(*v1alpha1.FlowLog)
	if !ok {
		return managed.ExternalCreation{}, errNotFlowLog
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
		// isUniqueNamesEnabled option enforces FlowLog names to be unique per Datacenter and NetworkLoadBalancer
		// Multiple Flow Logs with the same name will trigger an error
		// If only one instance is found, it will be "imported"
		flowLogDuplicateID, err := c.service.CheckDuplicateFlowLog(ctx, datacenterID, nlbID, cr.Spec.ForProvider.Name)
		if err != nil {
			return managed.ExternalCreation{}, err
		}
		if flowLogDuplicateID != "" {
			cr.Status.AtProvider.FlowLogID = flowLogDuplicateID
			meta.SetExternalName(cr, flowLogDuplicateID)
			return managed.ExternalCreation{}, nil
		}
	}
	flowLogInput := flowlog.GenerateCreateInput(cr)
	newInstance, err := c.service.CreateFlowLog(ctx, datacenterID, nlbID, flowLogInput)
	if err != nil {
		return managed.ExternalCreation{}, err
	}
	meta.SetExternalName(cr, *newInstance.Id)

	return managed.ExternalCreation{}, nil
}

func (c *externalFlowLog) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.FlowLog)
	if !ok {
		return managed.ExternalUpdate{}, errNotFlowLog
	}
	if cr.Status.AtProvider.State == compute.BUSY {
		return managed.ExternalUpdate{}, nil
	}

	datacenterID := cr.Spec.ForProvider.DatacenterCfg.DatacenterID
	nlbID := cr.Spec.ForProvider.NLBCfg.NetworkLoadBalancerID
	flowLogID := cr.Status.AtProvider.FlowLogID
	flowLogInput := flowlog.GenerateUpdateInput(cr)
	_, err := c.service.UpdateFlowLog(ctx, datacenterID, nlbID, flowLogID, flowLogInput)
	return managed.ExternalUpdate{}, err
}

func (c *externalFlowLog) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.FlowLog)
	if !ok {
		return managed.ExternalDelete{}, errNotFlowLog
	}
	cr.SetConditions(xpv1.Deleting())
	if cr.Status.AtProvider.State == compute.DESTROYING {
		return managed.ExternalDelete{}, nil
	}

	datacenterID := cr.Spec.ForProvider.DatacenterCfg.DatacenterID
	nlbID := cr.Spec.ForProvider.NLBCfg.NetworkLoadBalancerID
	err := c.service.DeleteFlowLog(ctx, datacenterID, nlbID, cr.Status.AtProvider.FlowLogID)
	if !errors.Is(err, flowlog.ErrNotFound) {
		return managed.ExternalDelete{}, err
	}
	return managed.ExternalDelete{}, nil
}

func (c *externalFlowLog) Disconnect(_ context.Context) error {
	return nil
}
