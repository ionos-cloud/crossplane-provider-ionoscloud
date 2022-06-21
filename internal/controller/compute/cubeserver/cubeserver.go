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

package cubeserver

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
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute/server"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute/volume"
)

const errNotCubeServer = "managed resource is not a Cube Server custom resource"

// Setup adds a controller that reconciles Server managed resources.
func Setup(mgr ctrl.Manager, l logging.Logger, rl workqueue.RateLimiter, poll time.Duration, creationGracePeriod time.Duration) error {
	name := managed.ControllerName(v1alpha1.CubeServerGroupKind)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(controller.Options{
			RateLimiter: ratelimiter.NewDefaultManagedRateLimiter(rl),
		}).
		For(&v1alpha1.CubeServer{}).
		Complete(managed.NewReconciler(mgr,
			resource.ManagedKind(v1alpha1.CubeServerGroupVersionKind),
			managed.WithExternalConnecter(&connectorServer{
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
	_, ok := mg.(*v1alpha1.CubeServer)
	if !ok {
		return nil, errors.New(errNotCubeServer)
	}
	svc, err := clients.ConnectForCRD(ctx, mg, c.kube, c.usage)
	return &externalServer{
		service:       &server.APIClient{IonosServices: svc},
		serviceVolume: &volume.APIClient{IonosServices: svc},
		log:           c.log}, err
}

// An ExternalClient observes, then either creates, updates, or deletes an
// externalServer resource to ensure it reflects the managed resource's desired state.
type externalServer struct {
	// A 'client' used to connect to the externalServer resource API. In practice this
	// would be something like an IONOS Cloud SDK client.
	service       server.Client
	serviceVolume volume.Client
	log           logging.Logger
}

func (c *externalServer) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) { // nolint:gocyclo
	cr, ok := mg.(*v1alpha1.CubeServer)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotCubeServer)
	}

	// External Name of the CR is the Cube Server ID
	if meta.GetExternalName(cr) == "" {
		return managed.ExternalObservation{}, nil
	}
	instance, apiResponse, err := c.service.GetServer(ctx, cr.Spec.ForProvider.DatacenterCfg.DatacenterID, meta.GetExternalName(cr))
	if err != nil {
		retErr := fmt.Errorf("failed to get cube server by id. error: %w", err)
		return managed.ExternalObservation{}, compute.CheckAPIResponseInfo(apiResponse, retErr)
	}
	if instance.Entities != nil && instance.Entities.Volumes != nil && instance.Entities.Volumes.Items != nil {
		if len(*instance.Entities.Volumes.Items) > 0 {
			items := *instance.Entities.Volumes.Items
			cr.Status.AtProvider.VolumeID = *items[0].Id
		}
	}
	current := cr.Spec.ForProvider.DeepCopy()
	server.LateInitializerCube(&cr.Spec.ForProvider, &instance)
	server.LateStatusInitializer(&cr.Status.AtProvider, &instance)

	cr.Status.AtProvider.ServerID = meta.GetExternalName(cr)
	cr.Status.AtProvider.State = clients.GetCoreResourceState(&instance)
	c.log.Debug(fmt.Sprintf("Observing state %v...", cr.Status.AtProvider.State))
	// Set Ready condition based on State
	clients.UpdateCondition(cr, cr.Status.AtProvider.State)

	return managed.ExternalObservation{
		ResourceExists:          true,
		ResourceUpToDate:        server.IsCubeServerUpToDate(cr, instance),
		ConnectionDetails:       managed.ConnectionDetails{},
		ResourceLateInitialized: !cmp.Equal(current, &cr.Spec.ForProvider),
	}, nil
}

func (c *externalServer) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.CubeServer)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotCubeServer)
	}

	cr.SetConditions(xpv1.Creating())
	if cr.Status.AtProvider.State == compute.BUSY {
		return managed.ExternalCreation{}, nil
	}

	instanceInput, err := server.GenerateCreateCubeServerInput(cr, c.service.GetAPIClient())
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	instance, apiResponse, err := c.service.CreateServer(ctx, cr.Spec.ForProvider.DatacenterCfg.DatacenterID, *instanceInput)
	creation := managed.ExternalCreation{ConnectionDetails: managed.ConnectionDetails{}}
	if err != nil {
		retErr := fmt.Errorf("failed to create cube server. error: %w", err)
		return creation, compute.AddAPIResponseInfo(apiResponse, retErr)
	}
	if err = compute.WaitForRequest(ctx, c.service.GetAPIClient(), apiResponse); err != nil {
		return creation, err
	}

	// Set External Name
	cr.Status.AtProvider.ServerID = *instance.Id
	meta.SetExternalName(cr, *instance.Id)
	return creation, nil
}

func (c *externalServer) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.CubeServer)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotCubeServer)
	}
	if cr.Status.AtProvider.State == compute.BUSY || cr.Status.AtProvider.State == compute.UPDATING {
		return managed.ExternalUpdate{}, nil
	}

	serverID := cr.Status.AtProvider.ServerID
	instanceInput, err := server.GenerateUpdateCubeServerInput(cr)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}
	_, apiResponse, err := c.service.UpdateServer(ctx, cr.Spec.ForProvider.DatacenterCfg.DatacenterID, serverID, *instanceInput)
	update := managed.ExternalUpdate{ConnectionDetails: managed.ConnectionDetails{}}
	if err != nil {
		retErr := fmt.Errorf("failed to update cube server. error: %w", err)
		return update, compute.AddAPIResponseInfo(apiResponse, retErr)
	}
	if err = compute.WaitForRequest(ctx, c.service.GetAPIClient(), apiResponse); err != nil {
		return update, err
	}
	instanceVolumeInput, err := server.GenerateUpdateVolumeInput(cr)
	if err != nil {
		return update, err
	}
	_, apiResponse, err = c.serviceVolume.UpdateVolume(ctx, cr.Spec.ForProvider.DatacenterCfg.DatacenterID, cr.Status.AtProvider.VolumeID, *instanceVolumeInput)
	if err != nil {
		retErr := fmt.Errorf("failed to update das volume. error: %w", err)
		return update, compute.AddAPIResponseInfo(apiResponse, retErr)
	}
	if err = compute.WaitForRequest(ctx, c.serviceVolume.GetAPIClient(), apiResponse); err != nil {
		return update, err
	}

	return update, nil
}

func (c *externalServer) Delete(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*v1alpha1.CubeServer)
	if !ok {
		return errors.New(errNotCubeServer)
	}

	cr.SetConditions(xpv1.Deleting())
	if cr.Status.AtProvider.State == compute.DESTROYING {
		return nil
	}

	// Deleting the CUBE Server will also delete the DAS Volume
	apiResponse, err := c.service.DeleteServer(ctx, cr.Spec.ForProvider.DatacenterCfg.DatacenterID, cr.Status.AtProvider.ServerID)
	if err != nil {
		retErr := fmt.Errorf("failed to delete cube server. error: %w", err)
		return compute.CheckAPIResponseInfo(apiResponse, retErr)
	}
	if err = compute.WaitForRequest(ctx, c.service.GetAPIClient(), apiResponse); err != nil {
		return err
	}
	return nil
}
