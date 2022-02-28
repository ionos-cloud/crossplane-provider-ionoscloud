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

package server

import (
	"context"
	"fmt"
	"reflect"

	sdkgo "github.com/ionos-cloud/sdk-go/v6"

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
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute/server"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"
)

const (
	errNotServer    = "managed resource is not a Server custom resource"
	errTrackPCUsage = "cannot track ProviderConfig usage"
	errGetPC        = "cannot get ProviderConfig"
	errGetCreds     = "cannot get credentials"

	errNewClient = "cannot create new Service"
)

// Setup adds a controller that reconciles Server managed resources.
func Setup(mgr ctrl.Manager, l logging.Logger, rl workqueue.RateLimiter) error {
	name := managed.ControllerName(v1alpha1.ServerGroupKind)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(controller.Options{
			RateLimiter: ratelimiter.NewDefaultManagedRateLimiter(rl),
		}).
		For(&v1alpha1.Server{}).
		Complete(managed.NewReconciler(mgr,
			resource.ManagedKind(v1alpha1.ServerGroupVersionKind),
			managed.WithExternalConnecter(&connectorServer{
				kube:  mgr.GetClient(),
				usage: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
				log:   l}),
			managed.WithReferenceResolver(managed.NewAPISimpleReferenceResolver(mgr.GetClient())),
			managed.WithLogger(l.WithValues("controller", name)),
			managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name)))))
}

// A connectorServer is expected to produce an ExternalClient when its Connect method
// is called.
type connectorServer struct {
	kube  client.Client
	usage resource.Tracker
	log   logging.Logger
}

// Connect typically produces an ExternalClient by:
// 1. Tracking that the managed resource is using a ProviderConfig.
// 2. Getting the managed resource's ProviderConfig.
// 3. Getting the credentials specified by the ProviderConfig.
// 4. Using the credentials to form a client.
func (c *connectorServer) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	_, ok := mg.(*v1alpha1.Server)
	if !ok {
		return nil, errors.New(errNotServer)
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
	return &externalServer{service: &server.APIClient{IonosServices: svc}, log: c.log}, nil
}

// An ExternalClient observes, then either creates, updates, or deletes an
// externalServer resource to ensure it reflects the managed resource's desired state.
type externalServer struct {
	// A 'client' used to connect to the externalServer resource API. In practice this
	// would be something like an AWS SDK client.
	service server.Client
	log     logging.Logger
}

func (c *externalServer) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.Server)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotServer)
	}

	// External Name of the CR is the Server ID
	id := meta.GetExternalName(cr)
	if id == "" {
		return managed.ExternalObservation{
			ResourceExists:    false,
			ResourceUpToDate:  false,
			ConnectionDetails: managed.ConnectionDetails{},
		}, nil
	}
	cr.Status.AtProvider.ServerID = id
	instance, apiResponse, err := c.service.GetServer(ctx, cr.Spec.ForProvider.DatacenterCfg.DatacenterID, id)
	if err != nil {
		retErr := fmt.Errorf("failed to get server by id. error: %w", err)
		retErr = compute.CheckAPIResponseInfo(apiResponse, retErr)
		return managed.ExternalObservation{
			ResourceExists:    false,
			ResourceUpToDate:  false,
			ConnectionDetails: managed.ConnectionDetails{},
		}, retErr
	}
	cr.Status.AtProvider.State = *instance.Metadata.State
	c.log.Debug(fmt.Sprintf("Observing state %v...", cr.Status.AtProvider.State))
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

	return managed.ExternalObservation{
		ResourceExists:    true,
		ResourceUpToDate:  server.IsServerUpToDate(cr, instance),
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

// Create
//nolint
func (c *externalServer) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.Server)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotServer)
	}

	cr.SetConditions(xpv1.Creating())
	if cr.Status.AtProvider.State == compute.BUSY {
		return managed.ExternalCreation{}, nil
	}

	instanceInput, err := server.GenerateCreateServerInput(cr)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	instance, apiResponse, err := c.service.CreateServer(ctx, cr.Spec.ForProvider.DatacenterCfg.DatacenterID, *instanceInput)
	creation := managed.ExternalCreation{
		ConnectionDetails: managed.ConnectionDetails{},
	}
	if err != nil {
		retErr := fmt.Errorf("failed to create server. error: %w", err)
		retErr = compute.AddAPIResponseInfo(apiResponse, retErr)
		return creation, retErr
	}
	if err = compute.WaitForRequest(ctx, c.service.GetAPIClient(), apiResponse); err != nil {
		return managed.ExternalCreation{}, err
	}

	// Set External Name
	cr.Status.AtProvider.ServerID = *instance.Id
	meta.SetExternalName(cr, *instance.Id)
	creation.ExternalNameAssigned = true

	// Attach volume
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.VolumeCfg)) {
		c.log.Debug("Attaching Volume...")
		instanceVolume, apiResponse, err := c.service.AttachVolume(ctx, cr.Spec.ForProvider.DatacenterCfg.DatacenterID, cr.Status.AtProvider.ServerID,
			sdkgo.Volume{Id: &cr.Spec.ForProvider.VolumeCfg.VolumeID})
		if err != nil {
			retErr := fmt.Errorf("failed to attach volume to server. error: %w", err)
			retErr = compute.AddAPIResponseInfo(apiResponse, retErr)
			return creation, retErr
		}
		if err = compute.WaitForRequest(ctx, c.service.GetAPIClient(), apiResponse); err != nil {
			return creation, err
		}
		cr.Status.AtProvider.VolumeID = *instanceVolume.Id
	}
	return creation, nil
}

// Update
//nolint
func (c *externalServer) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.Server)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotServer)
	}
	if cr.Status.AtProvider.State == compute.BUSY {
		return managed.ExternalUpdate{}, nil
	}
	serverID := cr.Status.AtProvider.ServerID
	// Attach volume
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.VolumeCfg)) {
		c.log.Debug("Attaching Volume...")
		instanceVolume, apiResponse, err := c.service.AttachVolume(ctx, cr.Spec.ForProvider.DatacenterCfg.DatacenterID, cr.Status.AtProvider.ServerID,
			sdkgo.Volume{Id: &cr.Spec.ForProvider.VolumeCfg.VolumeID})
		if err != nil {
			retErr := fmt.Errorf("failed to attach volume to server. error: %w", err)
			retErr = compute.AddAPIResponseInfo(apiResponse, retErr)
			return managed.ExternalUpdate{}, retErr
		}
		if err = compute.WaitForRequest(ctx, c.service.GetAPIClient(), apiResponse); err != nil {
			return managed.ExternalUpdate{}, err
		}
		cr.Status.AtProvider.VolumeID = *instanceVolume.Id
	} else if cr.Status.AtProvider.VolumeID != "" {
		apiResponse, err := c.service.DetachVolume(ctx, cr.Spec.ForProvider.DatacenterCfg.DatacenterID,
			cr.Status.AtProvider.ServerID, cr.Status.AtProvider.VolumeID)
		if err != nil {
			retErr := fmt.Errorf("failed to detach volume from server. error: %w", err)
			retErr = compute.AddAPIResponseInfo(apiResponse, retErr)
			return managed.ExternalUpdate{}, retErr
		}
		if err = compute.WaitForRequest(ctx, c.service.GetAPIClient(), apiResponse); err != nil {
			return managed.ExternalUpdate{}, err
		}
		cr.Status.AtProvider.VolumeID = ""
	}

	instanceInput, err := server.GenerateUpdateServerInput(cr)
	if err != nil {
		return managed.ExternalUpdate{}, nil
	}

	_, apiResponse, err := c.service.UpdateServer(ctx, cr.Spec.ForProvider.DatacenterCfg.DatacenterID, serverID, *instanceInput)
	update := managed.ExternalUpdate{
		ConnectionDetails: managed.ConnectionDetails{},
	}
	if err != nil {
		retErr := fmt.Errorf("failed to update server. error: %w", err)
		retErr = compute.AddAPIResponseInfo(apiResponse, retErr)
		return update, retErr
	}
	if err = compute.WaitForRequest(ctx, c.service.GetAPIClient(), apiResponse); err != nil {
		return managed.ExternalUpdate{}, err
	}
	return update, nil
}

func (c *externalServer) Delete(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*v1alpha1.Server)
	if !ok {
		return errors.New(errNotServer)
	}

	cr.SetConditions(xpv1.Deleting())
	if cr.Status.AtProvider.State == compute.DESTROYING {
		return nil
	}

	apiResponse, err := c.service.DeleteServer(ctx, cr.Spec.ForProvider.DatacenterCfg.DatacenterID, cr.Status.AtProvider.ServerID)
	if err != nil {
		retErr := fmt.Errorf("failed to delete server. error: %w", err)
		retErr = compute.AddAPIResponseInfo(apiResponse, retErr)
		return retErr
	}
	if err = compute.WaitForRequest(ctx, c.service.GetAPIClient(), apiResponse); err != nil {
		return err
	}
	return nil
}
