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
	"time"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
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
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute/lan"
)

const (
	errNotIPFailover = "managed resource is not a IPFailover custom resource"
	errTrackPCUsage  = "cannot track ProviderConfig usage"
	errGetPC         = "cannot get ProviderConfig"
	errGetCreds      = "cannot get credentials"

	errNewClient = "cannot create new Service"
)

// Setup adds a controller that reconciles IPFailover managed resources.
func Setup(mgr ctrl.Manager, l logging.Logger, rl workqueue.RateLimiter, poll time.Duration) error {
	name := managed.ControllerName(v1alpha1.IPFailoverGroupKind)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(controller.Options{
			RateLimiter: ratelimiter.NewDefaultManagedRateLimiter(rl),
		}).
		For(&v1alpha1.IPFailover{}).
		Complete(managed.NewReconciler(mgr,
			resource.ManagedKind(v1alpha1.IPFailoverGroupVersionKind),
			managed.WithExternalConnecter(&connectorIPFailover{
				kube:  mgr.GetClient(),
				usage: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
				log:   l}),
			managed.WithReferenceResolver(managed.NewAPISimpleReferenceResolver(mgr.GetClient())),
			managed.WithInitializers(),
			managed.WithPollInterval(poll),
			managed.WithLogger(l.WithValues("controller", name)),
			managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name)))))
}

// A connectorIPFailover is expected to produce an ExternalClient when its Connect method
// is called.
type connectorIPFailover struct {
	kube  client.Client
	usage resource.Tracker
	log   logging.Logger
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

	if err := c.usage.Track(ctx, mg); err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	pc := &apisv1alpha1.ProviderConfig{}
	if err := c.kube.Get(ctx, types.NamespacedName{Name: mg.GetProviderConfigReference().Name}, pc); err != nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	cd := pc.Spec.Credentials
	data, err := resource.CommonCredentialExtractor(ctx, cd.Source, c.kube, cd.CommonCredentialSelectors)
	if err != nil {
		return nil, errors.Wrap(err, errGetCreds)
	}

	svc, err := clients.NewIonosClients(data)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}
	return &externalIPFailover{service: &lan.APIClient{IonosServices: svc}, ipBlockService: &ipblock.APIClient{IonosServices: svc}, log: c.log}, nil
}

// An ExternalClient observes, then either creates, updates, or deletes an
// externalIPFailover resource to ensure it reflects the managed resource's desired state.
type externalIPFailover struct {
	// A 'client' used to connect to the externalIPFailover resource API. In practice this
	// would be something like an IONOS Cloud SDK client.
	service        lan.Client
	ipBlockService ipblock.Client
	log            logging.Logger
}

func (c *externalIPFailover) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.IPFailover)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotIPFailover)
	}

	// External Name of the CR is the IPFailover IP
	if meta.GetExternalName(cr) == "" {
		return managed.ExternalObservation{}, nil
	}
	instance, apiResponse, err := c.service.GetLan(ctx, cr.Spec.ForProvider.DatacenterCfg.DatacenterID, cr.Spec.ForProvider.LanCfg.LanID)
	if err != nil {
		retErr := fmt.Errorf("failed to get lan by id. error: %w", err)
		return managed.ExternalObservation{}, compute.CheckAPIResponseInfo(apiResponse, retErr)
	}

	ipSetByUser, err := c.getIPSet(ctx, cr)
	if err != nil {
		return managed.ExternalObservation{}, fmt.Errorf("failed to get ip: %w", err)
	}
	cr.Status.AtProvider.State = *instance.Metadata.State
	if lan.IsIPFailoverPresent(cr, instance, ipSetByUser) {
		cr.Status.AtProvider.IP = ipSetByUser
		cr.SetConditions(xpv1.Available())
	} else {
		cr.SetConditions(xpv1.Unavailable())
	}

	return managed.ExternalObservation{
		ResourceExists:    true,
		ResourceUpToDate:  lan.IsIPFailoverUpToDate(cr, instance, ipSetByUser),
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *externalIPFailover) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.IPFailover)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotIPFailover)
	}

	cr.SetConditions(xpv1.Creating())
	instance, apiResponse, err := c.service.GetLan(ctx, cr.Spec.ForProvider.DatacenterCfg.DatacenterID, cr.Spec.ForProvider.LanCfg.LanID)
	if err != nil {
		retErr := fmt.Errorf("failed to get lan by id. error: %w", err)
		return managed.ExternalCreation{}, compute.CheckAPIResponseInfo(apiResponse, retErr)
	}
	ip, err := c.getIPSet(ctx, cr)
	if err != nil {
		return managed.ExternalCreation{}, fmt.Errorf("failed to get ip: %w", err)
	}
	if lan.IsIPFailoverPresent(cr, instance, ip) {
		return managed.ExternalCreation{}, nil
	}

	// Generate IPFailover Input
	instanceInput, err := lan.GenerateCreateIPFailoverInput(cr, instance.Properties, ip)
	if err != nil {
		return managed.ExternalCreation{}, err
	}
	// Create IPFailover
	_, apiResponse, err = c.service.UpdateLan(ctx, cr.Spec.ForProvider.DatacenterCfg.DatacenterID, cr.Spec.ForProvider.LanCfg.LanID, *instanceInput)
	creation := managed.ExternalCreation{ConnectionDetails: managed.ConnectionDetails{}}
	if err != nil {
		retErr := fmt.Errorf("failed to update lan to create ipfailover. error: %w", err)
		return creation, compute.AddAPIResponseInfo(apiResponse, retErr)
	}
	if err = compute.WaitForRequest(ctx, c.service.GetAPIClient(), apiResponse); err != nil {
		return creation, err
	}

	// Set External Name
	meta.SetExternalName(cr, ip)
	creation.ExternalNameAssigned = true
	return creation, nil
}

func (c *externalIPFailover) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.IPFailover)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotIPFailover)
	}

	instance, apiResponse, err := c.service.GetLan(ctx, cr.Spec.ForProvider.DatacenterCfg.DatacenterID, cr.Spec.ForProvider.LanCfg.LanID)
	if err != nil {
		retErr := fmt.Errorf("failed to get lan by id. error: %w", err)
		return managed.ExternalUpdate{}, compute.CheckAPIResponseInfo(apiResponse, retErr)
	}
	ip, err := c.getIPSet(ctx, cr)
	if err != nil {
		return managed.ExternalUpdate{}, fmt.Errorf("failed to get ip: %w", err)
	}
	if lan.IsIPFailoverPresent(cr, instance, ip) {
		return managed.ExternalUpdate{}, nil
	}

	instanceInput, err := lan.GenerateUpdateIPFailoverInput(cr, instance.Properties, ip)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}
	// Update IPFailover
	_, apiResponse, err = c.service.UpdateLan(ctx, cr.Spec.ForProvider.DatacenterCfg.DatacenterID, cr.Spec.ForProvider.LanCfg.LanID, *instanceInput)
	if err != nil {
		retErr := fmt.Errorf("failed to update lan to update ipfailover. error: %w", err)
		return managed.ExternalUpdate{}, compute.AddAPIResponseInfo(apiResponse, retErr)
	}
	if err = compute.WaitForRequest(ctx, c.service.GetAPIClient(), apiResponse); err != nil {
		return managed.ExternalUpdate{}, err
	}
	return managed.ExternalUpdate{}, nil
}

func (c *externalIPFailover) Delete(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*v1alpha1.IPFailover)
	if !ok {
		return errors.New(errNotIPFailover)
	}

	cr.SetConditions(xpv1.Deleting())
	// Get Lan
	instance, apiResponse, err := c.service.GetLan(ctx, cr.Spec.ForProvider.DatacenterCfg.DatacenterID, cr.Spec.ForProvider.LanCfg.LanID)
	if err != nil {
		retErr := fmt.Errorf("failed to get lan by id. error: %w", err)
		return compute.CheckAPIResponseInfo(apiResponse, retErr)
	}
	ip, err := c.getIPSet(ctx, cr)
	if err != nil {
		return fmt.Errorf("failed to get ip: %w", err)
	}
	if !lan.IsIPFailoverPresent(cr, instance, ip) || cr.Status.AtProvider.State == compute.DESTROYING {
		cr.Status.AtProvider.IP = ""
		return nil
	}

	// Generate IPFailover Input to Remove
	instanceInput, err := lan.GenerateRemoveIPFailoverInput(cr, instance.Properties)
	if err != nil {
		return err
	}
	// Remove IPFailover
	_, apiResponse, err = c.service.UpdateLan(ctx, cr.Spec.ForProvider.DatacenterCfg.DatacenterID, cr.Spec.ForProvider.LanCfg.LanID, *instanceInput)
	if err != nil {
		retErr := fmt.Errorf("failed to update lan to remove ipfailover. error: %w", err)
		return compute.AddAPIResponseInfo(apiResponse, retErr)
	}
	if err = compute.WaitForRequest(ctx, c.service.GetAPIClient(), apiResponse); err != nil {
		return err
	}
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
