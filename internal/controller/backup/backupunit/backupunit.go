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

package backupunit

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/backup/v1alpha1"
	apisv1alpha1 "github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/backup/backupunit"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"
)

const errNotBackupUnit = "managed resource is not a BackupUnit custom resource"

// Setup adds a controller that reconciles BackupUnit managed resources.
func Setup(mgr ctrl.Manager, opts *utils.ConfigurationOptions) error {
	name := managed.ControllerName(v1alpha1.BackupUnitGroupKind)
	logger := opts.CtrlOpts.Logger

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(opts.CtrlOpts.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.BackupUnit{}).
		Complete(managed.NewReconciler(mgr,
			resource.ManagedKind(v1alpha1.BackupUnitGroupVersionKind),
			managed.WithExternalConnecter(&connectorBackupUnit{
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
			managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name)))))
}

// A connectorBackupUnit is expected to produce an ExternalClient when its Connect method
// is called.
type connectorBackupUnit struct {
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
func (c *connectorBackupUnit) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	_, ok := mg.(*v1alpha1.BackupUnit)
	if !ok {
		return nil, errors.New(errNotBackupUnit)
	}
	svc, err := clients.ConnectForCRD(ctx, mg, c.kube, c.usage)
	return &externalBackupUnit{
		service:              &backupunit.APIClient{IonosServices: svc},
		log:                  c.log,
		isUniqueNamesEnabled: c.isUniqueNamesEnabled}, err
}

// An ExternalClient observes, then either creates, updates, or deletes an
// externalBackupUnit resource to ensure it reflects the managed resource's desired state.
type externalBackupUnit struct {
	// A 'client' used to connect to the externalBackupUnit resource API. In practice this
	// would be something like an IONOS Cloud SDK client.
	service              backupunit.Client
	log                  logging.Logger
	isUniqueNamesEnabled bool
}

func (c *externalBackupUnit) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.BackupUnit)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotBackupUnit)
	}

	// External Name of the CR is the BackupUnit ID
	if meta.GetExternalName(cr) == "" {
		return managed.ExternalObservation{}, nil
	}
	instance, apiResponse, err := c.service.GetBackupUnit(ctx, meta.GetExternalName(cr))
	if err != nil {
		retErr := fmt.Errorf("failed to get backup unit by id. error: %w", err)
		return managed.ExternalObservation{}, compute.ErrorUnlessNotFound(apiResponse, retErr)
	}

	cr.Status.AtProvider.BackupUnitID = meta.GetExternalName(cr)
	cr.Status.AtProvider.State = clients.GetCoreResourceState(&instance)
	c.log.Debug(fmt.Sprintf("Observing state: %v", cr.Status.AtProvider.State))
	clients.UpdateCondition(cr, cr.Status.AtProvider.State)

	return managed.ExternalObservation{
		ResourceExists:    true,
		ResourceUpToDate:  backupunit.IsBackupUnitUpToDate(cr, instance),
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *externalBackupUnit) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.BackupUnit)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotBackupUnit)
	}
	cr.SetConditions(xpv1.Creating())
	if cr.Status.AtProvider.State == compute.BUSY {
		return managed.ExternalCreation{}, nil
	}

	if c.isUniqueNamesEnabled {
		// BackupUnits should have unique names per account.
		// Check if there are any existing backup units with the same name.
		// If there are multiple, an error will be returned.
		instance, err := c.service.CheckDuplicateBackupUnit(ctx, cr.Spec.ForProvider.Name, cr.Spec.ForProvider.Email)
		if err != nil {
			return managed.ExternalCreation{}, err
		}
		backupUnitID, err := c.service.GetBackupUnitID(instance)
		if err != nil {
			return managed.ExternalCreation{}, err
		}
		if backupUnitID != "" {
			// "Import" existing backup unit.
			cr.Status.AtProvider.BackupUnitID = backupUnitID
			meta.SetExternalName(cr, backupUnitID)
			return managed.ExternalCreation{}, nil
		}
	}

	instanceInput, err := backupunit.GenerateCreateBackupUnitInput(cr)
	if err != nil {
		return managed.ExternalCreation{}, err
	}
	newInstance, apiResponse, err := c.service.CreateBackupUnit(ctx, *instanceInput)
	creation := managed.ExternalCreation{ConnectionDetails: managed.ConnectionDetails{}}
	if err != nil {
		retErr := fmt.Errorf("failed to create backup unit. error: %w", err)
		return creation, compute.AddAPIResponseInfo(apiResponse, retErr)
	}
	if err = compute.WaitForRequest(ctx, c.service.GetAPIClient(), apiResponse); err != nil {
		return creation, err
	}
	// Set External Name
	cr.Status.AtProvider.BackupUnitID = *newInstance.Id
	meta.SetExternalName(cr, *newInstance.Id)
	return creation, nil
}

func (c *externalBackupUnit) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.BackupUnit)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotBackupUnit)
	}
	if cr.Status.AtProvider.State == compute.BUSY || cr.Status.AtProvider.State == compute.UPDATING {
		return managed.ExternalUpdate{}, nil
	}

	backupUnitID := cr.Status.AtProvider.BackupUnitID
	instanceInput, err := backupunit.GenerateUpdateBackupUnitInput(cr)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	_, apiResponse, err := c.service.UpdateBackupUnit(ctx, backupUnitID, *instanceInput)
	if err != nil {
		retErr := fmt.Errorf("failed to update backup unit. error: %w", err)
		return managed.ExternalUpdate{}, compute.AddAPIResponseInfo(apiResponse, retErr)
	}
	if err = compute.WaitForRequest(ctx, c.service.GetAPIClient(), apiResponse); err != nil {
		return managed.ExternalUpdate{}, err
	}
	return managed.ExternalUpdate{}, nil
}

func (c *externalBackupUnit) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.BackupUnit)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotBackupUnit)
	}

	cr.SetConditions(xpv1.Deleting())
	if cr.Status.AtProvider.State == compute.DESTROYING {
		return managed.ExternalDelete{}, nil
	}

	apiResponse, err := c.service.DeleteBackupUnit(ctx, cr.Status.AtProvider.BackupUnitID)
	if err != nil {
		retErr := fmt.Errorf("failed to delete backup unit. error: %w", err)
		return managed.ExternalDelete{}, compute.ErrorUnlessNotFound(apiResponse, retErr)
	}
	if err = compute.WaitForRequest(ctx, c.service.GetAPIClient(), apiResponse); err != nil {
		return managed.ExternalDelete{}, err
	}
	return managed.ExternalDelete{}, nil
}

// Disconnect does nothing because there are no resources to release. Needs to be implemented starting from crossplane-runtime v0.17
func (c *externalBackupUnit) Disconnect(_ context.Context) error {
	return nil
}
