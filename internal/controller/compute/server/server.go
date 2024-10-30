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
	sdkgo "github.com/ionos-cloud/sdk-go/v6"

	"github.com/google/go-cmp/cmp"
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
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute/server"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"
)

const errNotServer = "managed resource is not a Server custom resource"

// Setup adds a controller that reconciles Server managed resources.
func Setup(mgr ctrl.Manager, opts *utils.ConfigurationOptions) error {
	name := managed.ControllerName(v1alpha1.ServerGroupKind)
	logger := opts.CtrlOpts.Logger

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(controller.Options{
			RateLimiter: ratelimiter.NewController(),
		}).
		For(&v1alpha1.Server{}).
		Complete(managed.NewReconciler(mgr,
			resource.ManagedKind(v1alpha1.ServerGroupVersionKind),
			managed.WithExternalConnecter(&connectorServer{
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

// A connectorServer is expected to produce an ExternalClient when its Connect method
// is called.
type connectorServer struct {
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
func (c *connectorServer) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	_, ok := mg.(*v1alpha1.Server)
	if !ok {
		return nil, errors.New(errNotServer)
	}
	svc, err := clients.ConnectForCRD(ctx, mg, c.kube, c.usage)
	return &externalServer{
		service:              &server.APIClient{IonosServices: svc},
		log:                  c.log,
		isUniqueNamesEnabled: c.isUniqueNamesEnabled}, err
}

// An ExternalClient observes, then either creates, updates, or deletes an
// externalServer resource to ensure it reflects the managed resource's desired state.
type externalServer struct {
	// A 'client' used to connect to the externalServer resource API. In practice this
	// would be something like an IONOS Cloud SDK client.
	service              server.Client
	log                  logging.Logger
	isUniqueNamesEnabled bool
}

func (c *externalServer) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.Server)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotServer)
	}

	// External Name of the CR is the Server ID
	if meta.GetExternalName(cr) == "" {
		return managed.ExternalObservation{}, nil
	}
	observed, apiResponse, err := c.service.GetServer(ctx, cr.Spec.ForProvider.DatacenterCfg.DatacenterID, meta.GetExternalName(cr))
	if err != nil {
		retErr := fmt.Errorf("failed to get server by id. error: %w", err)
		return managed.ExternalObservation{}, compute.ErrorUnlessNotFound(apiResponse, retErr)
	}

	current := cr.Spec.ForProvider.DeepCopy()
	server.LateInitializer(&cr.Spec.ForProvider, &observed)
	server.LateStatusInitializer(&cr.Status.AtProvider, &observed)

	cr.Status.AtProvider.ServerID = meta.GetExternalName(cr)
	cr.Status.AtProvider.State = clients.GetCoreResourceState(&observed)
	if observed.Properties != nil {
		cr.Status.AtProvider.Name = *observed.Properties.Name
		if observed.Properties.BootVolume != nil && observed.Properties.BootVolume.Id != nil {
			cr.Status.AtProvider.VolumeID = *observed.Properties.BootVolume.Id
		}
	}
	c.log.Debug(fmt.Sprintf("Observing state: %v", cr.Status.AtProvider.State))
	clients.UpdateCondition(cr, cr.Status.AtProvider.State)

	return managed.ExternalObservation{
		ResourceExists:          true,
		ResourceUpToDate:        server.IsServerUpToDate(cr, observed),
		ConnectionDetails:       managed.ConnectionDetails{},
		ResourceLateInitialized: !cmp.Equal(current, &cr.Spec.ForProvider),
	}, nil
}

func (c *externalServer) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) { // nolint:gocyclo
	cr, ok := mg.(*v1alpha1.Server)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotServer)
	}
	cr.SetConditions(xpv1.Creating())
	if cr.Status.AtProvider.State == compute.BUSY {
		return managed.ExternalCreation{}, nil
	}

	if c.isUniqueNamesEnabled {
		// Servers should have unique names per datacenter.
		// Check if there are any existing servers with the same name.
		// If there are multiple, an error will be returned.
		instance, err := c.service.CheckDuplicateServer(ctx, cr.Spec.ForProvider.DatacenterCfg.DatacenterID,
			cr.Spec.ForProvider.Name, cr.Spec.ForProvider.CPUFamily)
		if err != nil {
			return managed.ExternalCreation{}, err
		}
		serverID, err := c.service.GetServerID(instance)
		if err != nil {
			return managed.ExternalCreation{}, err
		}
		if serverID != "" {
			// "Import" existing server.
			cr.Status.AtProvider.ServerID = serverID
			meta.SetExternalName(cr, serverID)
			return managed.ExternalCreation{}, nil
		}
	}

	instanceInput := server.GenerateCreateServerInput(cr)
	newInstance, apiResponse, err := c.service.CreateServer(ctx, cr.Spec.ForProvider.DatacenterCfg.DatacenterID, *instanceInput)
	creation := managed.ExternalCreation{ConnectionDetails: managed.ConnectionDetails{}}
	if err != nil {
		retErr := fmt.Errorf("failed to create server. error: %w", err)
		return creation, compute.AddAPIResponseInfo(apiResponse, retErr)
	}
	if err = compute.WaitForRequest(ctx, c.service.GetAPIClient(), apiResponse); err != nil {
		return creation, err
	}
	// Set External Name
	cr.Status.AtProvider.ServerID = *newInstance.Id
	meta.SetExternalName(cr, *newInstance.Id)
	if cr.Spec.ForProvider.VolumeCfg.VolumeID != "" {
		c.log.Debug("Attaching Volume...", "volume", cr.Spec.ForProvider.VolumeCfg.VolumeID)
		_, apiResponse, err = c.service.AttachVolume(ctx, cr.Spec.ForProvider.DatacenterCfg.DatacenterID,
			cr.Status.AtProvider.ServerID, sdkgo.Volume{Id: &cr.Spec.ForProvider.VolumeCfg.VolumeID})
		if err != nil {
			retErr := fmt.Errorf("failed to attach volume to server. error: %w", err)
			return creation, compute.AddAPIResponseInfo(apiResponse, retErr)
		}
		if err = compute.WaitForRequest(ctx, c.service.GetAPIClient(), apiResponse); err != nil {
			return creation, err
		}
		// Set Boot Volume ID
		cr.Status.AtProvider.VolumeID = cr.Spec.ForProvider.VolumeCfg.VolumeID
	}
	return creation, nil
}

func (c *externalServer) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) { // nolint:gocyclo
	cr, ok := mg.(*v1alpha1.Server)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotServer)
	}
	c.log.Debug("Update, started updating server", "name", cr.Spec.ForProvider.Name)
	if cr.Status.AtProvider.State == compute.BUSY {
		return managed.ExternalUpdate{}, nil
	}
	// Attach or Detach Volume
	if cr.Spec.ForProvider.VolumeCfg.VolumeID != "" {
		c.log.Debug("Update, attaching Volume", "volume", cr.Spec.ForProvider.VolumeCfg.VolumeID)
		_, apiResponse, err := c.service.AttachVolume(ctx, cr.Spec.ForProvider.DatacenterCfg.DatacenterID, cr.Status.AtProvider.ServerID,
			sdkgo.Volume{Id: &cr.Spec.ForProvider.VolumeCfg.VolumeID})
		if err != nil {
			retErr := fmt.Errorf("failed to attach volume to server. error: %w", err)
			return managed.ExternalUpdate{}, compute.AddAPIResponseInfo(apiResponse, retErr)
		}
		if err = compute.WaitForRequest(ctx, c.service.GetAPIClient(), apiResponse); err != nil {
			return managed.ExternalUpdate{}, err
		}
		c.log.Debug("Update, finished attaching Volume", "volume", cr.Spec.ForProvider.VolumeCfg.VolumeID)

	} else if cr.Status.AtProvider.VolumeID != "" {
		c.log.Debug("Detaching Volume...")
		apiResponse, err := c.service.DetachVolume(ctx, cr.Spec.ForProvider.DatacenterCfg.DatacenterID,
			cr.Status.AtProvider.ServerID, cr.Status.AtProvider.VolumeID)
		if err != nil {
			retErr := fmt.Errorf("failed to detach volume from server. error: %w", err)
			return managed.ExternalUpdate{}, compute.AddAPIResponseInfo(apiResponse, retErr)
		}
		if err = compute.WaitForRequest(ctx, c.service.GetAPIClient(), apiResponse); err != nil {
			return managed.ExternalUpdate{}, err
		}
	}
	// Set Boot Volume ID
	cr.Status.AtProvider.VolumeID = cr.Spec.ForProvider.VolumeCfg.VolumeID

	// Enterprise Server
	instanceInput := server.GenerateUpdateServerInput(cr)
	_, apiResponse, err := c.service.UpdateServer(ctx, cr.Spec.ForProvider.DatacenterCfg.DatacenterID, cr.Status.AtProvider.ServerID, *instanceInput)
	if err != nil {
		retErr := fmt.Errorf("failed to update server. error: %w", err)
		return managed.ExternalUpdate{}, compute.AddAPIResponseInfo(apiResponse, retErr)
	}
	if err = compute.WaitForRequest(ctx, c.service.GetAPIClient(), apiResponse); err != nil {
		return managed.ExternalUpdate{}, err
	}
	c.log.Debug("Update, finished updating server", "name", cr.Spec.ForProvider.Name)

	return managed.ExternalUpdate{}, nil
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
		return compute.ErrorUnlessNotFound(apiResponse, retErr)
	}
	if err = compute.WaitForRequest(ctx, c.service.GetAPIClient(), apiResponse); err != nil {
		return err
	}
	return nil
}
