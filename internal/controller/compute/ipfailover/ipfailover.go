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

package ipfailover

import (
	"context"
	"fmt"
	"strings"

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
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute/ipblock"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute/lan"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"
)

const errNotIPFailover = "managed resource is not a IPFailover custom resource"

// Setup adds a controller that reconciles IPFailover managed resources.
func Setup(mgr ctrl.Manager, opts *utils.ConfigurationOptions) error {
	name := managed.ControllerName(v1alpha1.IPFailoverGroupKind)
	logger := opts.CtrlOpts.Logger

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(controller.Options{
			RateLimiter: ratelimiter.NewController(),
		}).
		For(&v1alpha1.IPFailover{}).
		Complete(managed.NewReconciler(mgr,
			resource.ManagedKind(v1alpha1.IPFailoverGroupVersionKind),
			managed.WithExternalConnecter(&connectorIPFailover{
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

// A connectorIPFailover is expected to produce an ExternalClient when its Connect method
// is called.
type connectorIPFailover struct {
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
func (c *connectorIPFailover) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	_, ok := mg.(*v1alpha1.IPFailover)
	if !ok {
		return nil, errors.New(errNotIPFailover)
	}
	svc, err := clients.ConnectForCRD(ctx, mg, c.kube, c.usage)
	return &externalIPFailover{
		service:              &lan.APIClient{IonosServices: svc},
		ipBlockService:       &ipblock.APIClient{IonosServices: svc},
		log:                  c.log,
		isUniqueNamesEnabled: c.isUniqueNamesEnabled}, err
}

// An ExternalClient observes, then either creates, updates, or deletes an
// externalIPFailover resource to ensure it reflects the managed resource's desired state.
type externalIPFailover struct {
	// A 'client' used to connect to the externalIPFailover resource API. In practice this
	// would be something like an IONOS Cloud SDK client.
	service              lan.Client
	ipBlockService       ipblock.Client
	log                  logging.Logger
	isUniqueNamesEnabled bool
}

var (
	available ipFailoverState = "AVAILABLE"
	creating  ipFailoverState = "CREATING"
	updating  ipFailoverState = "UPDATING"
	deleting  ipFailoverState = "DELETING"
)

type ipFailoverState string

func (c *externalIPFailover) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) { // nolint:gocyclo
	cr, ok := mg.(*v1alpha1.IPFailover)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotIPFailover)
	}

	if meta.GetExternalName(cr) == "" {
		return managed.ExternalObservation{}, nil
	}
	// Observe IPFailovers present on specified Lan
	instanceIPFailovers, err := c.service.GetLanIPFailovers(ctx, cr.Spec.ForProvider.DatacenterCfg.DatacenterID, cr.Spec.ForProvider.LanCfg.LanID)
	if err != nil {
		if cr.Status.AtProvider.State == string(deleting) || strings.Contains(err.Error(), "404 Not Found") {
			return managed.ExternalObservation{}, nil
		}
		return managed.ExternalObservation{}, fmt.Errorf("failed to get lan ipfailovers by id. error: %w", err)
	}
	ipSetByUser, err := c.getIPSet(ctx, cr)
	if err != nil {
		return managed.ExternalObservation{}, fmt.Errorf("failed to get ip: %w", err)
	}
	// Check if the IP Failover is created and present
	if lan.IsIPFailoverPresent(instanceIPFailovers, ipSetByUser, cr.Spec.ForProvider.NicCfg.NicID) {
		cr.Status.AtProvider.IP = ipSetByUser
		cr.Status.AtProvider.State = string(available)
	}
	c.log.Debug(fmt.Sprintf("Observing state: %v", cr.Status.AtProvider.State))
	// Set Ready condition based on State
	clients.UpdateCondition(cr, cr.Status.AtProvider.State)

	return managed.ExternalObservation{
		ResourceExists:    true,
		ResourceUpToDate:  lan.IsIPFailoverUpToDate(cr, instanceIPFailovers, ipSetByUser),
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *externalIPFailover) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.IPFailover)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotIPFailover)
	}

	cr.SetConditions(xpv1.Creating())
	instanceIPFailovers, err := c.service.GetLanIPFailovers(ctx, cr.Spec.ForProvider.DatacenterCfg.DatacenterID, cr.Spec.ForProvider.LanCfg.LanID)
	if err != nil && instanceIPFailovers != nil {
		return managed.ExternalCreation{}, fmt.Errorf("failed to get lan ipfailovers by id. error: %w", err)
	}
	ipSetByUser, err := c.getIPSet(ctx, cr)
	if err != nil {
		return managed.ExternalCreation{}, fmt.Errorf("failed to get ip: %w", err)
	}
	if lan.IsIPFailoverPresent(instanceIPFailovers, ipSetByUser, cr.Spec.ForProvider.NicCfg.NicID) {
		return managed.ExternalCreation{}, nil
	}
	instanceInput, err := lan.GenerateCreateIPFailoverInput(instanceIPFailovers, ipSetByUser, cr.Spec.ForProvider.NicCfg.NicID)
	if err != nil {
		return managed.ExternalCreation{}, fmt.Errorf("failed to generate input for ipfailover creation: %w", err)
	}
	// Create IPFailover - Update Lan with the new IP Failover
	_, apiResponse, err := c.service.UpdateLan(ctx, cr.Spec.ForProvider.DatacenterCfg.DatacenterID, cr.Spec.ForProvider.LanCfg.LanID, *instanceInput)
	if err != nil {
		retErr := fmt.Errorf("failed to update lan to create ipfailover. error: %w", err)
		return managed.ExternalCreation{}, compute.AddAPIResponseInfo(apiResponse, retErr)
	}
	if err = compute.WaitForRequest(ctx, c.service.GetAPIClient(), apiResponse); err != nil {
		return managed.ExternalCreation{}, fmt.Errorf("failed waiting for request on creating ipfailover: %w", err)
	}

	cr.Status.AtProvider.IP = ipSetByUser
	cr.Status.AtProvider.State = string(creating)
	meta.SetExternalName(cr, cr.ObjectMeta.Name)
	return managed.ExternalCreation{}, nil
}

func (c *externalIPFailover) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.IPFailover)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotIPFailover)
	}
	if cr.Status.AtProvider.State == string(updating) {
		return managed.ExternalUpdate{}, nil
	}

	instanceIPFailovers, err := c.service.GetLanIPFailovers(ctx, cr.Spec.ForProvider.DatacenterCfg.DatacenterID, cr.Spec.ForProvider.LanCfg.LanID)
	if err != nil && instanceIPFailovers != nil {
		return managed.ExternalUpdate{}, fmt.Errorf("failed to get lan ipfailovers by id. error: %w", err)
	}
	ipSetByUser, err := c.getIPSet(ctx, cr)
	if err != nil {
		return managed.ExternalUpdate{}, fmt.Errorf("failed to get ip: %w", err)
	}
	if lan.IsIPFailoverPresent(instanceIPFailovers, ipSetByUser, cr.Spec.ForProvider.NicCfg.NicID) {
		return managed.ExternalUpdate{}, nil
	}
	instanceInput, err := lan.GenerateUpdateIPFailoverInput(instanceIPFailovers, ipSetByUser, cr.Status.AtProvider.IP, cr.Spec.ForProvider.NicCfg.NicID)
	if err != nil {
		return managed.ExternalUpdate{}, fmt.Errorf("failed to generate input for ipfailover update: %w", err)
	}
	// Update IPFailover - Update Lan with the new ipfailover, replacing the old one
	_, apiResponse, err := c.service.UpdateLan(ctx, cr.Spec.ForProvider.DatacenterCfg.DatacenterID, cr.Spec.ForProvider.LanCfg.LanID, *instanceInput)
	if err != nil {
		retErr := fmt.Errorf("failed to update lan to update ipfailover. error: %w", err)
		return managed.ExternalUpdate{}, compute.AddAPIResponseInfo(apiResponse, retErr)
	}
	if err = compute.WaitForRequest(ctx, c.service.GetAPIClient(), apiResponse); err != nil {
		return managed.ExternalUpdate{}, fmt.Errorf("failed to wait for request on updating ipfailover: %w", err)
	}

	cr.Status.AtProvider.IP = ipSetByUser
	cr.Status.AtProvider.State = string(updating)
	return managed.ExternalUpdate{}, nil
}

func (c *externalIPFailover) Delete(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*v1alpha1.IPFailover)
	if !ok {
		return errors.New(errNotIPFailover)
	}

	cr.SetConditions(xpv1.Deleting())
	if cr.Status.AtProvider.State == string(deleting) {
		return nil
	}
	instanceIPFailovers, err := c.service.GetLanIPFailovers(ctx, cr.Spec.ForProvider.DatacenterCfg.DatacenterID, cr.Spec.ForProvider.LanCfg.LanID)
	if err != nil {
		if strings.Contains(err.Error(), "404 Not Found") {
			return nil
		}
		return fmt.Errorf("failed to get lan ipfailovers. error: %w", err)
	}
	if !lan.IsIPFailoverPresent(instanceIPFailovers, cr.Status.AtProvider.IP, cr.Spec.ForProvider.NicCfg.NicID) {
		return nil
	}
	instanceInput, err := lan.GenerateRemoveIPFailoverInput(instanceIPFailovers, cr.Status.AtProvider.IP)
	if err != nil {
		return fmt.Errorf("failed to generate input for ipfailover deletion: %w", err)
	}
	// Remove IPFailover - Update Lan with the new ipfailovers - without the one needed to be deleted
	_, apiResponse, err := c.service.UpdateLan(ctx, cr.Spec.ForProvider.DatacenterCfg.DatacenterID, cr.Spec.ForProvider.LanCfg.LanID, *instanceInput)
	if err != nil {
		retErr := fmt.Errorf("failed to update lan to remove ipfailover. error: %w", err)
		return compute.ErrorUnlessNotFound(apiResponse, retErr)
	}
	if err = compute.WaitForRequest(ctx, c.service.GetAPIClient(), apiResponse); err != nil {
		return fmt.Errorf("failed to wait for request on removing ipfailover: %w", err)
	}

	cr.Status.AtProvider.State = string(deleting)
	return nil
}

// getIPSet will return ip set by the user on ip or ipConfig fields of the spec.
// If both fields are set, only the ip field will be considered by the Crossplane
// Provider IONOS Cloud.
func (c *externalIPFailover) getIPSet(ctx context.Context, cr *v1alpha1.IPFailover) (string, error) {
	if cr.Spec.ForProvider.IPCfg.IP != "" {
		return cr.Spec.ForProvider.IPCfg.IP, nil
	}
	if cr.Spec.ForProvider.IPCfg.IPBlockCfg.IPBlockID != "" {
		ipsCfg, err := c.ipBlockService.GetIPs(ctx, cr.Spec.ForProvider.IPCfg.IPBlockCfg.IPBlockID, cr.Spec.ForProvider.IPCfg.IPBlockCfg.Index)
		if err != nil {
			return "", err
		}
		if len(ipsCfg) != 1 {
			return "", fmt.Errorf("error getting IP with index %v from IPBlock %v",
				cr.Spec.ForProvider.IPCfg.IPBlockCfg.Index, cr.Spec.ForProvider.IPCfg.IPBlockCfg.IPBlockID)
		}
		return ipsCfg[0], nil
	}
	return "", fmt.Errorf("error getting IP set")
}
