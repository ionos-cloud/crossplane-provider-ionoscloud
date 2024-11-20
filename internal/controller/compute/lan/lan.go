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

package lan

import (
	"context"
	"fmt"

	"github.com/google/go-cmp/cmp"
	sdkgo "github.com/ionos-cloud/sdk-go/v6"
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

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	apisv1alpha1 "github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute/lan"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"
)

const errNotLan = "managed resource is not a Lan custom resource"

// Setup adds a controller that reconciles Lan managed resources.
func Setup(mgr ctrl.Manager, opts *utils.ConfigurationOptions) error {
	name := managed.ControllerName(v1alpha1.LanGroupKind)
	logger := opts.CtrlOpts.Logger

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(controller.Options{
			RateLimiter: ratelimiter.NewController(),
		}).
		For(&v1alpha1.Lan{}).
		Complete(managed.NewReconciler(mgr,
			resource.ManagedKind(v1alpha1.LanGroupVersionKind),
			managed.WithExternalConnecter(&connectorLan{
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

// A connectorLan is expected to produce an ExternalClient when its Connect method
// is called.
type connectorLan struct {
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
func (c *connectorLan) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	_, ok := mg.(*v1alpha1.Lan)
	if !ok {
		return nil, errors.New(errNotLan)
	}
	svc, err := clients.ConnectForCRD(ctx, mg, c.kube, c.usage)
	return &externalLan{
		service:              &lan.APIClient{IonosServices: svc},
		log:                  c.log,
		isUniqueNamesEnabled: c.isUniqueNamesEnabled}, err
}

// An ExternalClient observes, then either creates, updates, or deletes an
// externalLan resource to ensure it reflects the managed resource's desired state.
type externalLan struct {
	// A 'client' used to connect to the externalLan resource API. In practice this
	// would be something like an IONOS Cloud SDK client.
	service              lan.Client
	log                  logging.Logger
	isUniqueNamesEnabled bool
}

func (c *externalLan) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.Lan)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotLan)
	}

	// External Name of the CR is the Lan ID
	if meta.GetExternalName(cr) == "" {
		return managed.ExternalObservation{}, nil
	}
	instance, apiResponse, err := c.service.GetLan(ctx, cr.Spec.ForProvider.DatacenterCfg.DatacenterID, meta.GetExternalName(cr))
	if err != nil {
		retErr := fmt.Errorf("failed to get lan by id. error: %w", err)
		return managed.ExternalObservation{}, compute.ErrorUnlessNotFound(apiResponse, retErr)
	}
	current := cr.Spec.ForProvider.DeepCopy()
	lateInitializer(&cr.Spec.ForProvider, &instance)
	cr.Status.AtProvider.IPFailovers = lan.GetIPFailoverIPs(instance)
	cr.Status.AtProvider.LanID = meta.GetExternalName(cr)
	cr.Status.AtProvider.State = clients.GetCoreResourceState(&instance)
	if instance.Properties != nil {
		cr.Status.AtProvider.Name = *instance.Properties.Name
	}
	c.log.Debug(fmt.Sprintf("Observing state: %v", cr.Status.AtProvider.State))
	// Set Ready condition based on State
	clients.UpdateCondition(cr, cr.Status.AtProvider.State)

	return managed.ExternalObservation{
		ResourceExists:          true,
		ResourceUpToDate:        lan.IsLanUpToDate(cr, instance),
		ConnectionDetails:       managed.ConnectionDetails{},
		ResourceLateInitialized: !cmp.Equal(current, &cr.Spec.ForProvider),
	}, nil
}

// LateInitializer fills the empty fields that are set in the backend
func lateInitializer(in *v1alpha1.LanParameters, lan *sdkgo.Lan) { // nolint:gocyclo
	if lan == nil {
		return
	}
	// Add Properties to the Spec, if they were set by the API
	if lan.Properties != nil {
		if lan.Properties.Ipv6CidrBlock != nil && in.Ipv6Cidr == v1alpha1.LANAuto {
			if in.Ipv6Cidr == "" || in.Ipv6Cidr == v1alpha1.LANAuto {
				in.Ipv6Cidr = *lan.Properties.Ipv6CidrBlock
			}
		}
	}
}
func (c *externalLan) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) { // nolint:gocyclo
	cr, ok := mg.(*v1alpha1.Lan)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotLan)
	}
	cr.SetConditions(xpv1.Creating())
	if cr.Status.AtProvider.State == compute.BUSY {
		return managed.ExternalCreation{}, nil
	}

	if c.isUniqueNamesEnabled {
		// Lans should have unique names per datacenter.
		// Check if there are any existing lans with the same name.
		// If there are multiple, an error will be returned.
		instance, err := c.service.CheckDuplicateLan(ctx, cr.Spec.ForProvider.DatacenterCfg.DatacenterID, cr.Spec.ForProvider.Name)
		if err != nil {
			return managed.ExternalCreation{}, err
		}
		lanID, err := c.service.GetLanID(instance)
		if err != nil {
			return managed.ExternalCreation{}, err
		}
		if lanID != "" {
			// "Import" existing lan.
			cr.Status.AtProvider.LanID = lanID
			meta.SetExternalName(cr, lanID)
			return managed.ExternalCreation{}, nil
		}
	}

	instanceInput, err := lan.GenerateCreateLanInput(cr)
	if err != nil {
		return managed.ExternalCreation{}, err
	}
	newInstance, apiResponse, err := c.service.CreateLan(ctx, cr.Spec.ForProvider.DatacenterCfg.DatacenterID, *instanceInput)
	creation := managed.ExternalCreation{ConnectionDetails: managed.ConnectionDetails{}}
	if err != nil {
		retErr := fmt.Errorf("failed to create lan. error: %w", err)
		return creation, compute.AddAPIResponseInfo(apiResponse, retErr)
	}
	if err = compute.WaitForRequest(ctx, c.service.GetAPIClient(), apiResponse); err != nil {
		return creation, err
	}
	// Set External Name
	cr.Status.AtProvider.LanID = *newInstance.Id
	meta.SetExternalName(cr, *newInstance.Id)
	if newInstance.Properties != nil {
		cr.Status.AtProvider.Name = *newInstance.Properties.Name
	}
	return creation, nil
}

func (c *externalLan) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.Lan)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotLan)
	}
	if cr.Status.AtProvider.State == compute.BUSY || cr.Status.AtProvider.State == compute.UPDATING {
		return managed.ExternalUpdate{}, nil
	}

	instanceInput, err := lan.GenerateUpdateLanInput(cr)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	_, apiResponse, err := c.service.UpdateLan(ctx, cr.Spec.ForProvider.DatacenterCfg.DatacenterID, cr.Status.AtProvider.LanID, *instanceInput)
	if err != nil {
		retErr := fmt.Errorf("failed to update lan. error: %w", err)
		return managed.ExternalUpdate{}, compute.AddAPIResponseInfo(apiResponse, retErr)
	}
	if err = compute.WaitForRequest(ctx, c.service.GetAPIClient(), apiResponse); err != nil {
		return managed.ExternalUpdate{}, err
	}
	return managed.ExternalUpdate{}, nil
}

func (c *externalLan) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.Lan)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotLan)
	}

	cr.SetConditions(xpv1.Deleting())
	if cr.Status.AtProvider.State == compute.DESTROYING {
		return managed.ExternalDelete{}, nil
	}

	apiResponse, err := c.service.DeleteLan(ctx, cr.Spec.ForProvider.DatacenterCfg.DatacenterID, cr.Status.AtProvider.LanID)
	if err != nil {
		retErr := fmt.Errorf("failed to delete lan. error: %w", err)
		return managed.ExternalDelete{}, compute.ErrorUnlessNotFound(apiResponse, retErr)
	}
	if err = compute.WaitForRequest(ctx, c.service.GetAPIClient(), apiResponse); err != nil {
		return managed.ExternalDelete{}, err
	}
	return managed.ExternalDelete{}, nil
}

func (c *externalLan) Disconnect(_ context.Context) error {
	return nil
}
