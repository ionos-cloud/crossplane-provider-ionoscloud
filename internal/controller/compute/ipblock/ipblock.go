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

package ipblock

import (
	"context"
	"fmt"
	"net/http"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/crossplane/crossplane-runtime/pkg/statemetrics"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	apisv1alpha1 "github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute/ipblock"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"
)

const errNotIPBlock = "managed resource is not a IPBlock custom resource"

// Setup adds a controller that reconciles IPBlock managed resources.
func Setup(mgr ctrl.Manager, opts *utils.ConfigurationOptions) error {
	name := managed.ControllerName(v1alpha1.IPBlockGroupKind)
	logger := opts.CtrlOpts.Logger
	if opts.CtrlOpts.MetricOptions != nil && opts.CtrlOpts.MetricOptions.MRStateMetrics != nil {
		stateMetricsRecorder := statemetrics.NewMRStateRecorder(
			mgr.GetClient(), opts.CtrlOpts.Logger, opts.CtrlOpts.MetricOptions.MRStateMetrics, &v1alpha1.IPBlockList{}, opts.CtrlOpts.MetricOptions.PollStateMetricInterval,
		)
		if err := mgr.Add(stateMetricsRecorder); err != nil {
			return errors.Wrap(err, "cannot register MR state metrics recorder for kind"+name)
		}
	}

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: opts.GetMaxConcurrentReconcileRate(v1alpha1.IPBlockKind),
			RateLimiter:             ratelimiter.NewController(),
			RecoverPanic:            ptr.To(true),
		}).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.IPBlock{}).
		Complete(managed.NewReconciler(mgr,
			resource.ManagedKind(v1alpha1.IPBlockGroupVersionKind),
			managed.WithExternalConnecter(&connectorIPBlock{
				kube:                 mgr.GetClient(),
				usage:                resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
				log:                  logger,
				isUniqueNamesEnabled: opts.GetIsUniqueNamesEnabled()}),
			managed.WithReferenceResolver(managed.NewAPISimpleReferenceResolver(mgr.GetClient())),
			managed.WithInitializers(),
			managed.WithPollInterval(opts.GetPollInterval()),
			managed.WithTimeout(opts.GetTimeout()),
			managed.WithCreationGracePeriod(opts.GetCreationGracePeriod()),
			managed.WithLogger(logger.WithValues("controller", name)),
			managed.WithMetricRecorder(opts.CtrlOpts.MetricOptions.MRMetrics),
			managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name)))))
}

// A connectorIPBlock is expected to produce an ExternalClient when its Connect method
// is called.
type connectorIPBlock struct {
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
func (c *connectorIPBlock) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	_, ok := mg.(*v1alpha1.IPBlock)
	if !ok {
		return nil, errors.New(errNotIPBlock)
	}
	svc, err := clients.ConnectForCRD(ctx, mg, c.kube, c.usage)
	return &externalIPBlock{
		service:              &ipblock.APIClient{IonosServices: svc},
		log:                  c.log,
		isUniqueNamesEnabled: c.isUniqueNamesEnabled}, err
}

// An ExternalClient observes, then either creates, updates, or deletes an
// externalIPBlock resource to ensure it reflects the managed resource's desired state.
type externalIPBlock struct {
	// A 'client' used to connect to the externalIPBlock resource API. In practice this
	// would be something like an IONOS Cloud SDK client.
	service              ipblock.Client
	log                  logging.Logger
	isUniqueNamesEnabled bool
}

func (c *externalIPBlock) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.IPBlock)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotIPBlock)
	}

	// External Name of the CR is the IPBlock ID
	if meta.GetExternalName(cr) == "" {
		return managed.ExternalObservation{}, nil
	}
	observed, apiResponse, err := c.service.GetIPBlock(ctx, meta.GetExternalName(cr))
	if err != nil {
		retErr := fmt.Errorf("failed to get ipBlock by id. error: %w", err)
		return managed.ExternalObservation{}, compute.ErrorUnlessNotFound(apiResponse, retErr)
	}
	current := cr.Spec.ForProvider.DeepCopy()
	ipblock.LateStatusInitializer(&cr.Status.AtProvider, &observed)

	cr.Status.AtProvider.IPBlockID = meta.GetExternalName(cr)
	cr.Status.AtProvider.State = clients.GetCoreResourceState(&observed)
	c.log.Debug("Observed ipblock: ", "state", cr.Status.AtProvider.State, "external name", meta.GetExternalName(cr), "name", cr.Spec.ForProvider.Name)
	// Set Ready condition based on State
	clients.UpdateCondition(cr, cr.Status.AtProvider.State)

	return managed.ExternalObservation{
		ResourceExists:          true,
		ResourceUpToDate:        ipblock.IsIPBlockUpToDate(cr, observed),
		ConnectionDetails:       managed.ConnectionDetails{},
		ResourceLateInitialized: !cmp.Equal(current, &cr.Spec.ForProvider),
	}, nil
}

func (c *externalIPBlock) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.IPBlock)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotIPBlock)
	}
	cr.SetConditions(xpv1.Creating())
	if cr.Status.AtProvider.State == compute.BUSY {
		return managed.ExternalCreation{}, nil
	}

	if externalName := meta.GetExternalName(cr); externalName != "" && externalName != cr.Name {
		isDone, err := compute.IsRequestDone(ctx, c.service.GetAPIClient(), externalName, http.MethodPost)
		if err != nil {
			return managed.ExternalCreation{}, err
		}

		if isDone {
			return managed.ExternalCreation{ConnectionDetails: managed.ConnectionDetails{}}, nil
		}

		return managed.ExternalCreation{}, nil
	}

	if c.isUniqueNamesEnabled {
		// IPBlocks should have unique names per account.
		// Check if there are any existing volumes with the same name.
		// If there are multiple, an error will be returned.
		instance, err := c.service.CheckDuplicateIPBlock(ctx, cr.Spec.ForProvider.Name, cr.Spec.ForProvider.Location)
		if err != nil {
			return managed.ExternalCreation{}, err
		}
		ipBlockID, err := c.service.GetIPBlockID(instance)
		if err != nil {
			return managed.ExternalCreation{}, err
		}
		if ipBlockID != "" {
			// "Import" existing IPBlock.
			cr.Status.AtProvider.IPBlockID = ipBlockID
			meta.SetExternalName(cr, ipBlockID)
			return managed.ExternalCreation{}, nil
		}
	}

	instanceInput, err := ipblock.GenerateCreateIPBlockInput(cr)
	if err != nil {
		return managed.ExternalCreation{}, err
	}
	newInstance, apiResponse, err := c.service.CreateIPBlock(ctx, *instanceInput)
	creation := managed.ExternalCreation{ConnectionDetails: managed.ConnectionDetails{}}
	if err != nil {
		retErr := fmt.Errorf("failed to create ipBlock. error: %w", err)
		return creation, compute.AddAPIResponseInfo(apiResponse, retErr)
	}

	// Set External Name
	cr.Status.AtProvider.IPBlockID = *newInstance.Id
	meta.SetExternalName(cr, *newInstance.Id)
	if err = compute.WaitForRequest(ctx, c.service.GetAPIClient(), apiResponse); err != nil {
		return creation, err
	}

	return creation, nil
}

func (c *externalIPBlock) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.IPBlock)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotIPBlock)
	}
	if cr.Status.AtProvider.State == compute.BUSY || cr.Status.AtProvider.State == compute.UPDATING {
		return managed.ExternalUpdate{}, nil
	}

	ipBlockID := cr.Status.AtProvider.IPBlockID
	instanceInput, err := ipblock.GenerateUpdateIPBlockInput(cr)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	_, apiResponse, err := c.service.UpdateIPBlock(ctx, ipBlockID, *instanceInput)
	if err != nil {
		retErr := fmt.Errorf("failed to update ipBlock. error: %w", err)
		return managed.ExternalUpdate{}, compute.AddAPIResponseInfo(apiResponse, retErr)
	}
	if err = compute.WaitForRequest(ctx, c.service.GetAPIClient(), apiResponse); err != nil {
		return managed.ExternalUpdate{}, err
	}
	return managed.ExternalUpdate{}, nil
}

func (c *externalIPBlock) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.IPBlock)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotIPBlock)
	}

	cr.SetConditions(xpv1.Deleting())
	if cr.Status.AtProvider.State == compute.DESTROYING {
		return managed.ExternalDelete{}, nil
	}

	apiResponse, err := c.service.DeleteIPBlock(ctx, cr.Status.AtProvider.IPBlockID)
	if err != nil {
		retErr := fmt.Errorf("failed to delete ipBlock. error: %w", err)
		return managed.ExternalDelete{}, compute.ErrorUnlessNotFound(apiResponse, retErr)
	}
	if err = compute.WaitForRequest(ctx, c.service.GetAPIClient(), apiResponse); err != nil {
		return managed.ExternalDelete{}, err
	}
	return managed.ExternalDelete{}, nil
}

// Disconnect does nothing because there are no resources to release. Needs to be implemented starting from crossplane-runtime v0.17
func (c *externalIPBlock) Disconnect(_ context.Context) error {
	return nil
}
