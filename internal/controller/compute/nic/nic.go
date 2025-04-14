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

package nic

import (
	"context"
	"fmt"
	"time"

	"github.com/google/go-cmp/cmp"
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
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute/ipblock"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute/nic"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"
)

const errNotNic = "managed resource is not a Nic custom resource"

// Setup adds a controller that reconciles Nic managed resources.
func Setup(mgr ctrl.Manager, opts *utils.ConfigurationOptions) error {
	name := managed.ControllerName(v1alpha1.NicGroupKind)
	logger := opts.CtrlOpts.Logger

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(opts.CtrlOpts.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.Nic{}).
		Complete(managed.NewReconciler(mgr,
			resource.ManagedKind(v1alpha1.NicGroupVersionKind),
			managed.WithExternalConnecter(&connectorNic{
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
			managed.WithPollIntervalHook(func(mg resource.Managed, pollInterval time.Duration) time.Duration {
				return utils.CalculatePollInterval(mg, pollInterval, opts.PollJitter)
			}),
			managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name)))))
}

// A connectorNic is expected to produce an ExternalClient when its Connect method
// is called.
type connectorNic struct {
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
func (c *connectorNic) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	_, ok := mg.(*v1alpha1.Nic)
	if !ok {
		return nil, errors.New(errNotNic)
	}
	svc, err := clients.ConnectForCRD(ctx, mg, c.kube, c.usage)
	return &externalNic{
		service:              &nic.APIClient{IonosServices: svc},
		ipBlockService:       &ipblock.APIClient{IonosServices: svc},
		log:                  c.log,
		isUniqueNamesEnabled: c.isUniqueNamesEnabled}, err
}

// An ExternalClient observes, then either creates, updates, or deletes an
// externalNic resource to ensure it reflects the managed resource's desired state.
type externalNic struct {
	// A 'client' used to connect to the externalNic resource API. In practice this
	// would be something like an IONOS Cloud SDK client.
	service              nic.Client
	ipBlockService       ipblock.Client
	log                  logging.Logger
	isUniqueNamesEnabled bool
}

func (c *externalNic) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) { // nolint: gocyclo
	cr, ok := mg.(*v1alpha1.Nic)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotNic)
	}

	// External Name of the CR is the Nic ID
	if meta.GetExternalName(cr) == "" {
		return managed.ExternalObservation{}, nil
	}
	instance, apiResponse, err := c.service.GetNic(ctx, cr.Spec.ForProvider.DatacenterCfg.DatacenterID,
		cr.Spec.ForProvider.ServerCfg.ServerID, meta.GetExternalName(cr))
	if err != nil {
		retErr := fmt.Errorf("failed to get nic by id. error: %w", err)
		return managed.ExternalObservation{}, compute.ErrorUnlessNotFound(apiResponse, retErr)
	}

	current := cr.Spec.ForProvider.DeepCopy()
	nic.LateInitializer(&cr.Spec.ForProvider, &instance)
	nic.LateStatusInitializer(&cr.Status.AtProvider, &instance)

	cr.Status.AtProvider.NicID = meta.GetExternalName(cr)
	cr.Status.AtProvider.State = clients.GetCoreResourceState(&instance)

	c.log.Debug(fmt.Sprintf("Observing state: %v", cr.Status.AtProvider.State))
	// Set Ready condition based on State
	clients.UpdateCondition(cr, cr.Status.AtProvider.State)

	// Resolve IPs
	ips, err := c.getIpsSet(ctx, cr)
	if err != nil {
		return managed.ExternalObservation{}, fmt.Errorf("failed to get ips: %w", err)
	}
	return managed.ExternalObservation{
		ResourceExists:          true,
		ResourceUpToDate:        nic.IsNicUpToDate(cr, instance, ips),
		ConnectionDetails:       managed.ConnectionDetails{},
		ResourceLateInitialized: !cmp.Equal(current, &cr.Spec.ForProvider),
	}, nil
}

func (c *externalNic) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) { // nolint: gocyclo
	cr, ok := mg.(*v1alpha1.Nic)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotNic)
	}
	cr.SetConditions(xpv1.Creating())
	if cr.Status.AtProvider.State == compute.BUSY {
		return managed.ExternalCreation{}, nil
	}

	if c.isUniqueNamesEnabled {
		// NICs should have unique names per server.
		// Check if there are any existing nics with the same name.
		// If there are multiple, an error will be returned.
		instance, err := c.service.CheckDuplicateNic(ctx, cr.Spec.ForProvider.DatacenterCfg.DatacenterID,
			cr.Spec.ForProvider.ServerCfg.ServerID, cr.Spec.ForProvider.Name)
		if err != nil {
			return managed.ExternalCreation{}, err
		}
		nicID, err := c.service.GetNicID(instance)
		if err != nil {
			return managed.ExternalCreation{}, err
		}
		if nicID != "" {
			// "Import" existing Nic.
			cr.Status.AtProvider.NicID = nicID
			meta.SetExternalName(cr, nicID)
			return managed.ExternalCreation{}, nil
		}
	}

	ips, err := c.getIpsSet(ctx, cr)
	if err != nil {
		return managed.ExternalCreation{}, fmt.Errorf("failed to get ips: %w", err)
	}
	instanceInput, err := nic.GenerateCreateNicInput(cr, ips)
	if err != nil {
		return managed.ExternalCreation{}, err
	}
	if cr.Spec.ForProvider.ServerCfg.ServerID == "" {
		return managed.ExternalCreation{}, fmt.Errorf("serverId is required")
	}
	newInstance, apiResponse, err := c.service.CreateNic(ctx, cr.Spec.ForProvider.DatacenterCfg.DatacenterID,
		cr.Spec.ForProvider.ServerCfg.ServerID, *instanceInput)
	creation := managed.ExternalCreation{ConnectionDetails: managed.ConnectionDetails{}}
	if err != nil {
		retErr := fmt.Errorf("failed to create nic. error: %w", err)
		return creation, compute.AddAPIResponseInfo(apiResponse, retErr)
	}
	if err = compute.WaitForRequest(ctx, c.service.GetAPIClient(), apiResponse); err != nil {
		return creation, err
	}
	// Set External Name
	cr.Status.AtProvider.NicID = *newInstance.Id
	meta.SetExternalName(cr, *newInstance.Id)
	return creation, nil
}

func (c *externalNic) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.Nic)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotNic)
	}
	if cr.Status.AtProvider.State == compute.BUSY || cr.Status.AtProvider.State == compute.UPDATING {
		return managed.ExternalUpdate{}, nil
	}

	ips, err := c.getIpsSet(ctx, cr)
	if err != nil {
		return managed.ExternalUpdate{}, fmt.Errorf("failed to get ips: %w", err)
	}
	instanceInput, err := nic.GenerateUpdateNicInput(cr, ips)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	_, apiResponse, err := c.service.UpdateNic(ctx, cr.Spec.ForProvider.DatacenterCfg.DatacenterID,
		cr.Spec.ForProvider.ServerCfg.ServerID, cr.Status.AtProvider.NicID, *instanceInput)
	if err != nil {
		retErr := fmt.Errorf("failed to update nic. error: %w", err)
		return managed.ExternalUpdate{}, compute.AddAPIResponseInfo(apiResponse, retErr)
	}
	if err = compute.WaitForRequest(ctx, c.service.GetAPIClient(), apiResponse); err != nil {
		return managed.ExternalUpdate{}, err
	}
	return managed.ExternalUpdate{}, nil
}

func (c *externalNic) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.Nic)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotNic)
	}

	cr.SetConditions(xpv1.Deleting())
	if cr.Status.AtProvider.State == compute.DESTROYING {
		return managed.ExternalDelete{}, nil
	}

	apiResponse, err := c.service.DeleteNic(ctx, cr.Spec.ForProvider.DatacenterCfg.DatacenterID,
		cr.Spec.ForProvider.ServerCfg.ServerID, cr.Status.AtProvider.NicID)
	if err != nil {
		retErr := fmt.Errorf("failed to delete nic. error: %w", err)
		return managed.ExternalDelete{}, compute.ErrorUnlessNotFound(apiResponse, retErr)
	}
	if err = compute.WaitForRequest(ctx, c.service.GetAPIClient(), apiResponse); err != nil {
		return managed.ExternalDelete{}, err
	}
	return managed.ExternalDelete{}, nil
}

// getIpsSet will return ips set by the user on ips or ipsConfig fields of the spec.
// If both fields are set, only the ips field will be considered by the Crossplane
// Provider IONOS Cloud.
func (c *externalNic) getIpsSet(ctx context.Context, cr *v1alpha1.Nic) ([]string, error) {
	if len(cr.Spec.ForProvider.IpsCfg.IPs) == 0 && len(cr.Spec.ForProvider.IpsCfg.IPBlockCfgs) == 0 {
		return nil, nil
	}
	if len(cr.Spec.ForProvider.IpsCfg.IPs) > 0 {
		return cr.Spec.ForProvider.IpsCfg.IPs, nil
	}
	ips := make([]string, 0)
	if len(cr.Spec.ForProvider.IpsCfg.IPBlockCfgs) > 0 {
		for i, cfg := range cr.Spec.ForProvider.IpsCfg.IPBlockCfgs {
			if cfg.IPBlockID == "" {
				return nil, fmt.Errorf("error resolving references for ipblock at index: %v", i)
			}
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
func (c *externalNic) Disconnect(_ context.Context) error {
	return nil
}
