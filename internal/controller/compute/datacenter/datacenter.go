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

package datacenter

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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	apisv1alpha1 "github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute/datacenter"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"
)

const errNotDatacenter = "managed resource is not a Datacenter custom resource"

// Setup adds a controller that reconciles Datacenter managed resources.
func Setup(mgr ctrl.Manager, opts *utils.ConfigurationOptions) error {
	name := managed.ControllerName(v1alpha1.DatacenterGroupKind)

	logger := opts.CtrlOpts.Logger
	if opts.CtrlOpts.MetricOptions != nil && opts.CtrlOpts.MetricOptions.MRMetrics != nil {
		stateMetricsRecorder := statemetrics.NewMRStateRecorder(
			mgr.GetClient(), opts.CtrlOpts.Logger, opts.CtrlOpts.MetricOptions.MRStateMetrics, &v1alpha1.DatacenterList{}, opts.CtrlOpts.MetricOptions.PollStateMetricInterval,
		)
		if err := mgr.Add(stateMetricsRecorder); err != nil {
			return errors.Wrap(err, "cannot register MR state metrics recorder for kind"+name)
		}
	}
	if opts.CtrlOpts.MetricOptions != nil && opts.CtrlOpts.MetricOptions.MRMetrics != nil {
		stateMetricsRecorder := statemetrics.NewMRStateRecorder(
			mgr.GetClient(), opts.CtrlOpts.Logger, opts.CtrlOpts.MetricOptions.MRStateMetrics, &v1alpha1.DatacenterList{}, opts.CtrlOpts.MetricOptions.PollStateMetricInterval,
		)
		if err := mgr.Add(stateMetricsRecorder); err != nil {
			return errors.Wrap(err, "cannot register MR state metrics recorder for kind"+name)
		}
	}

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(opts.CtrlOpts.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.Datacenter{}).
		Complete(managed.NewReconciler(mgr,
			resource.ManagedKind(v1alpha1.DatacenterGroupVersionKind),
			managed.WithExternalConnecter(&connectorDatacenter{
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

// A connectorDatacenter is expected to produce an ExternalClient when its Connect method
// is called.
type connectorDatacenter struct {
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
func (c *connectorDatacenter) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	_, ok := mg.(*v1alpha1.Datacenter)
	if !ok {
		return nil, errors.New(errNotDatacenter)
	}
	svc, err := clients.ConnectForCRD(ctx, mg, c.kube, c.usage)
	return &externalDatacenter{
		service:              &datacenter.APIClient{IonosServices: svc},
		log:                  c.log,
		isUniqueNamesEnabled: c.isUniqueNamesEnabled}, err
}

// An ExternalClient observes, then either creates, updates, or deletes an
// externalDatacenter resource to ensure it reflects the managed resource's desired state.
type externalDatacenter struct {
	// A 'client' used to connect to the externalDatacenter resource API. In practice this
	// would be something like an IONOS Cloud SDK client.
	service              datacenter.Client
	log                  logging.Logger
	isUniqueNamesEnabled bool
}

func (c *externalDatacenter) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.Datacenter)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotDatacenter)
	}

	// External Name of the CR is the Datacenter ID
	if meta.GetExternalName(cr) == "" {
		return managed.ExternalObservation{}, nil
	}
	instance, apiResponse, err := c.service.GetDatacenter(ctx, meta.GetExternalName(cr))
	if err != nil {
		retErr := fmt.Errorf("failed to get datacenter by id. error: %w", err)
		return managed.ExternalObservation{}, compute.ErrorUnlessNotFound(apiResponse, retErr)
	}

	cr.Status.AtProvider.DatacenterID = meta.GetExternalName(cr)
	cr.Status.AtProvider.State = clients.GetCoreResourceState(&instance)
	cr.Status.AtProvider.AvailableCPUFamilies, _ = c.service.GetCPUFamiliesForDatacenter(ctx, cr.Status.AtProvider.DatacenterID)
	c.log.Debug(fmt.Sprintf("Observing state: %v", cr.Status.AtProvider.State))
	clients.UpdateCondition(cr, cr.Status.AtProvider.State)

	return managed.ExternalObservation{
		ResourceExists:    true,
		ResourceUpToDate:  datacenter.IsDatacenterUpToDate(cr, instance),
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *externalDatacenter) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.Datacenter)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotDatacenter)
	}
	cr.SetConditions(xpv1.Creating())
	if cr.Status.AtProvider.State == compute.BUSY {
		return managed.ExternalCreation{}, nil
	}

	if c.isUniqueNamesEnabled {
		// Datacenters should have unique names per account.
		// Check if there are any existing datacenters with the same name.
		// If there are multiple, an error will be returned.
		instance, err := c.service.CheckDuplicateDatacenter(ctx, cr.Spec.ForProvider.Name, cr.Spec.ForProvider.Location)
		if err != nil {
			return managed.ExternalCreation{}, err
		}
		datacenterID, err := c.service.GetDatacenterID(instance)
		if err != nil {
			return managed.ExternalCreation{}, err
		}
		if datacenterID != "" {
			// "Import" existing datacenter.
			cr.Status.AtProvider.DatacenterID = datacenterID
			meta.SetExternalName(cr, datacenterID)
			return managed.ExternalCreation{}, nil
		}
	}

	// Create new datacenter instance accordingly
	// with the properties set.
	instanceInput, err := datacenter.GenerateCreateDatacenterInput(cr)
	if err != nil {
		return managed.ExternalCreation{}, err
	}
	newInstance, apiResponse, err := c.service.CreateDatacenter(ctx, *instanceInput)
	if err != nil {
		retErr := fmt.Errorf("failed to create datacenter. error: %w", err)
		return managed.ExternalCreation{}, compute.AddAPIResponseInfo(apiResponse, retErr)
	}
	if err = compute.WaitForRequest(ctx, c.service.GetAPIClient(), apiResponse); err != nil {
		return managed.ExternalCreation{}, err
	}
	cr.Status.AtProvider.DatacenterID = *newInstance.Id
	meta.SetExternalName(cr, *newInstance.Id)
	return managed.ExternalCreation{}, nil
}

func (c *externalDatacenter) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.Datacenter)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotDatacenter)
	}
	if cr.Status.AtProvider.State == compute.BUSY || cr.Status.AtProvider.State == compute.UPDATING {
		return managed.ExternalUpdate{}, nil
	}

	datacenterID := cr.Status.AtProvider.DatacenterID
	instanceInput, err := datacenter.GenerateUpdateDatacenterInput(cr)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	_, apiResponse, err := c.service.UpdateDatacenter(ctx, datacenterID, *instanceInput)
	if err != nil {
		retErr := fmt.Errorf("failed to update datacenter. error: %w", err)
		return managed.ExternalUpdate{}, compute.AddAPIResponseInfo(apiResponse, retErr)
	}
	if err = compute.WaitForRequest(ctx, c.service.GetAPIClient(), apiResponse); err != nil {
		return managed.ExternalUpdate{}, err
	}
	return managed.ExternalUpdate{}, nil
}

func (c *externalDatacenter) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.Datacenter)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotDatacenter)
	}

	cr.SetConditions(xpv1.Deleting())
	if cr.Status.AtProvider.State == compute.DESTROYING {
		return managed.ExternalDelete{}, nil
	}

	apiResponse, err := c.service.DeleteDatacenter(ctx, cr.Status.AtProvider.DatacenterID)
	if err != nil {
		retErr := fmt.Errorf("failed to delete datacenter. error: %w", err)
		return managed.ExternalDelete{}, compute.ErrorUnlessNotFound(apiResponse, retErr)
	}
	if err = compute.WaitForRequest(ctx, c.service.GetAPIClient(), apiResponse); err != nil {
		return managed.ExternalDelete{}, err
	}
	return managed.ExternalDelete{}, nil
}

// Disconnect does nothing because there are no resources to release. Needs to be implemented starting from crossplane-runtime v0.17
func (c *externalDatacenter) Disconnect(_ context.Context) error {
	return nil
}
