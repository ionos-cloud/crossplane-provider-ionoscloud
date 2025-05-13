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

package pcc

import (
	"context"
	"fmt"

	"github.com/crossplane/crossplane-runtime/pkg/statemetrics"
	"github.com/pkg/errors"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	apisv1alpha1 "github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute/pcc"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"
)

const errNotPrivateCrossConnect = "managed resource is not a pcc custom resource"

// Setup adds a controller that reconciles pcc managed resources.
func Setup(mgr ctrl.Manager, opts *utils.ConfigurationOptions) error {
	name := managed.ControllerName(v1alpha1.PrivateCrossConnectGroupKind)
	logger := opts.CtrlOpts.Logger
	if opts.CtrlOpts.MetricOptions != nil && opts.CtrlOpts.MetricOptions.MRStateMetrics != nil {
		stateMetricsRecorder := statemetrics.NewMRStateRecorder(
			mgr.GetClient(), opts.CtrlOpts.Logger, opts.CtrlOpts.MetricOptions.MRStateMetrics, &v1alpha1.PccList{}, opts.CtrlOpts.MetricOptions.PollStateMetricInterval,
		)
		if err := mgr.Add(stateMetricsRecorder); err != nil {
			return errors.Wrap(err, "cannot register MR state metrics recorder for kind"+name)
		}
	}

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(opts.CtrlOpts.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.Pcc{}).
		Complete(managed.NewReconciler(mgr,
			resource.ManagedKind(v1alpha1.PrivateCrossConnectGroupVersionKind),
			managed.WithExternalConnecter(&connectorPrivateCrossConnect{
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

// A connectorPrivateCrossConnect is expected to produce an ExternalClient when its Connect method
// is called.
type connectorPrivateCrossConnect struct {
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
func (c *connectorPrivateCrossConnect) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	_, ok := mg.(*v1alpha1.Pcc)
	if !ok {
		return nil, errors.New(errNotPrivateCrossConnect)
	}
	svc, err := clients.ConnectForCRD(ctx, mg, c.kube, c.usage)
	return &externalPrivateCrossConnect{
		service:              &pcc.APIClient{IonosServices: svc},
		log:                  c.log,
		isUniqueNamesEnabled: c.isUniqueNamesEnabled}, err
}

// An ExternalClient observes, then either creates, updates, or deletes an
// externalPrivateCrossConnect resource to ensure it reflects the managed resource's desired state.
type externalPrivateCrossConnect struct {
	// A 'client' used to connect to the externalPrivateCrossConnect resource API. In practice this
	// would be something like an IONOS Cloud SDK client.
	service              pcc.Client
	log                  logging.Logger
	isUniqueNamesEnabled bool
}

func (c *externalPrivateCrossConnect) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.Pcc)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotPrivateCrossConnect)
	}

	// External Name of the CR is the pcc ID
	if meta.GetExternalName(cr) == "" {
		return managed.ExternalObservation{}, nil
	}
	instance, apiResponse, err := c.service.GetPrivateCrossConnect(ctx, meta.GetExternalName(cr))
	if err != nil {
		retErr := fmt.Errorf("failed to get privateCrossConnect by id. error: %w", err)
		return managed.ExternalObservation{}, compute.ErrorUnlessNotFound(apiResponse, retErr)
	}

	cr.Status.AtProvider.PrivateCrossConnectID = meta.GetExternalName(cr)
	cr.Status.AtProvider.State = clients.GetCoreResourceState(&instance)
	c.log.Debug(fmt.Sprintf("Observing state: %v", cr.Status.AtProvider.State))
	clients.UpdateCondition(cr, cr.Status.AtProvider.State)

	return managed.ExternalObservation{
		ResourceExists:    true,
		ResourceUpToDate:  pcc.IsPrivateCrossConnectUpToDate(cr, instance),
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *externalPrivateCrossConnect) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.Pcc)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotPrivateCrossConnect)
	}
	cr.SetConditions(xpv1.Creating())
	if cr.Status.AtProvider.State == compute.BUSY {
		return managed.ExternalCreation{}, nil
	}

	if c.isUniqueNamesEnabled {
		// PrivateCrossConnects should have unique names per account.
		// Check if there are any existing privateCrossConnects with the same name.
		// If there are multiple, an error will be returned.
		instance, err := c.service.CheckDuplicatePrivateCrossConnect(ctx, cr.Spec.ForProvider.Name)
		if err != nil {
			return managed.ExternalCreation{}, err
		}
		privateCrossConnectID, err := c.service.GetPrivateCrossConnectID(instance)
		if err != nil {
			return managed.ExternalCreation{}, err
		}
		if privateCrossConnectID != "" {
			// "Import" existing privateCrossConnect.
			cr.Status.AtProvider.PrivateCrossConnectID = privateCrossConnectID
			meta.SetExternalName(cr, privateCrossConnectID)
			return managed.ExternalCreation{}, nil
		}
	}

	// Create new privateCrossConnect instance accordingly
	// with the properties set.
	instanceInput, err := pcc.GenerateCreatePrivateCrossConnectInput(cr)
	if err != nil {
		return managed.ExternalCreation{}, err
	}
	newInstance, apiResponse, err := c.service.CreatePrivateCrossConnect(ctx, *instanceInput)
	if err != nil {
		retErr := fmt.Errorf("failed to create privateCrossConnect. error: %w", err)
		return managed.ExternalCreation{}, compute.AddAPIResponseInfo(apiResponse, retErr)
	}
	if err = compute.WaitForRequest(ctx, c.service.GetAPIClient(), apiResponse); err != nil {
		return managed.ExternalCreation{}, err
	}
	cr.Status.AtProvider.PrivateCrossConnectID = *newInstance.Id
	meta.SetExternalName(cr, *newInstance.Id)
	return managed.ExternalCreation{}, nil
}

func (c *externalPrivateCrossConnect) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.Pcc)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotPrivateCrossConnect)
	}
	if cr.Status.AtProvider.State == compute.BUSY || cr.Status.AtProvider.State == compute.UPDATING {
		return managed.ExternalUpdate{}, nil
	}

	privateCrossConnectID := cr.Status.AtProvider.PrivateCrossConnectID
	instanceInput, err := pcc.GenerateUpdatePrivateCrossConnectInput(cr)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	_, apiResponse, err := c.service.UpdatePrivateCrossConnect(ctx, privateCrossConnectID, *instanceInput)
	if err != nil {
		retErr := fmt.Errorf("failed to update privateCrossConnect. error: %w", err)
		return managed.ExternalUpdate{}, compute.AddAPIResponseInfo(apiResponse, retErr)
	}
	if err = compute.WaitForRequest(ctx, c.service.GetAPIClient(), apiResponse); err != nil {
		return managed.ExternalUpdate{}, err
	}
	return managed.ExternalUpdate{}, nil
}

func (c *externalPrivateCrossConnect) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.Pcc)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotPrivateCrossConnect)
	}

	cr.SetConditions(xpv1.Deleting())
	if cr.Status.AtProvider.State == compute.DESTROYING {
		return managed.ExternalDelete{}, nil
	}

	apiResponse, err := c.service.DeletePrivateCrossConnect(ctx, cr.Status.AtProvider.PrivateCrossConnectID)
	if err != nil {
		retErr := fmt.Errorf("failed to delete privateCrossConnect. error: %w", err)
		return managed.ExternalDelete{}, compute.ErrorUnlessNotFound(apiResponse, retErr)
	}
	if err = compute.WaitForRequest(ctx, c.service.GetAPIClient(), apiResponse); err != nil {
		return managed.ExternalDelete{}, err
	}
	return managed.ExternalDelete{}, nil
}

// Disconnect does nothing because there are no resources to release. Needs to be implemented starting from crossplane-runtime v0.17
func (c *externalPrivateCrossConnect) Disconnect(_ context.Context) error {
	return nil
}
