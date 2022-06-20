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
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute/ipblock"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute/nic"
)

const errNotNic = "managed resource is not a Nic custom resource"

// Setup adds a controller that reconciles Nic managed resources.
func Setup(mgr ctrl.Manager, l logging.Logger, rl workqueue.RateLimiter, poll time.Duration, creationGracePeriod time.Duration) error {
	name := managed.ControllerName(v1alpha1.NicGroupKind)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(controller.Options{
			RateLimiter: ratelimiter.NewDefaultManagedRateLimiter(rl),
		}).
		For(&v1alpha1.Nic{}).
		Complete(managed.NewReconciler(mgr,
			resource.ManagedKind(v1alpha1.NicGroupVersionKind),
			managed.WithExternalConnecter(&connectorNic{
				kube:  mgr.GetClient(),
				usage: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
				log:   l}),
			managed.WithReferenceResolver(managed.NewAPISimpleReferenceResolver(mgr.GetClient())),
			managed.WithInitializers(),
			managed.WithPollInterval(poll),
			managed.WithCreationGracePeriod(creationGracePeriod),
			managed.WithLogger(l.WithValues("controller", name)),
			managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name)))))
}

// A connectorNic is expected to produce an ExternalClient when its Connect method
// is called.
type connectorNic struct {
	kube  client.Client
	usage resource.Tracker
	log   logging.Logger
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
		service:        &nic.APIClient{IonosServices: svc},
		ipblockService: &ipblock.APIClient{IonosServices: svc},
		log:            c.log}, err
}

// An ExternalClient observes, then either creates, updates, or deletes an
// externalNic resource to ensure it reflects the managed resource's desired state.
type externalNic struct {
	// A 'client' used to connect to the externalNic resource API. In practice this
	// would be something like an IONOS Cloud SDK client.
	service        nic.Client
	ipblockService ipblock.Client
	log            logging.Logger
}

// Keep old value of Nic's IPs to correctly handle the update.
// The user can set an IP and after that to unset it.
var oldIPsNic []string

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
		return managed.ExternalObservation{}, compute.CheckAPIResponseInfo(apiResponse, retErr)
	}

	current := cr.Spec.ForProvider.DeepCopy()
	nic.LateInitializer(&cr.Spec.ForProvider, &instance)

	cr.Status.AtProvider.NicID = meta.GetExternalName(cr)
	if instance.HasMetadata() {
		if instance.Metadata.HasState() {
			cr.Status.AtProvider.State = *instance.Metadata.State
		}
	}
	if instance.HasProperties() {
		if instance.Properties.HasIps() {
			cr.Status.AtProvider.IPs = *instance.Properties.Ips
		}
	}
	c.log.Debug(fmt.Sprintf("Observing state: %v", cr.Status.AtProvider.State))
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

	// Resolve IPs
	ips, err := c.getIpsSet(ctx, cr)
	if err != nil {
		return managed.ExternalObservation{}, fmt.Errorf("failed to get ips: %w", err)
	}
	return managed.ExternalObservation{
		ResourceExists:          true,
		ResourceUpToDate:        nic.IsNicUpToDate(cr, instance, ips, oldIPsNic),
		ConnectionDetails:       managed.ConnectionDetails{},
		ResourceLateInitialized: !cmp.Equal(current, &cr.Spec.ForProvider),
	}, nil
}

func (c *externalNic) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.Nic)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotNic)
	}

	cr.SetConditions(xpv1.Creating())
	if cr.Status.AtProvider.State == compute.BUSY {
		return managed.ExternalCreation{}, nil
	}

	ips, err := c.getIpsSet(ctx, cr)
	if err != nil {
		return managed.ExternalCreation{}, fmt.Errorf("failed to get ips: %w", err)
	}
	oldIPsNic = ips
	instanceInput, err := nic.GenerateCreateNicInput(cr, ips)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	instance, apiResponse, err := c.service.CreateNic(ctx, cr.Spec.ForProvider.DatacenterCfg.DatacenterID,
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
	cr.Status.AtProvider.NicID = *instance.Id
	meta.SetExternalName(cr, *instance.Id)
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
	oldIPsNic = ips
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

func (c *externalNic) Delete(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*v1alpha1.Nic)
	if !ok {
		return errors.New(errNotNic)
	}

	cr.SetConditions(xpv1.Deleting())
	if cr.Status.AtProvider.State == compute.DESTROYING {
		return nil
	}

	apiResponse, err := c.service.DeleteNic(ctx, cr.Spec.ForProvider.DatacenterCfg.DatacenterID,
		cr.Spec.ForProvider.ServerCfg.ServerID, cr.Status.AtProvider.NicID)
	if err != nil {
		retErr := fmt.Errorf("failed to delete nic. error: %w", err)
		return compute.AddAPIResponseInfo(apiResponse, retErr)
	}
	if err = compute.WaitForRequest(ctx, c.service.GetAPIClient(), apiResponse); err != nil {
		return err
	}
	return nil
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
			ipsCfg, err := c.ipblockService.GetIPs(ctx, cfg.IPBlockID, cfg.Indexes...)
			if err != nil {
				return nil, err
			}
			ips = append(ips, ipsCfg...)
		}
	}
	return ips, nil
}
