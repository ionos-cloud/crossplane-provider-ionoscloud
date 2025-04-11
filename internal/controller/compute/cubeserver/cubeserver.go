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
	"math/rand/v2"
	"time"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	apisv1alpha1 "github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute/server"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute/template"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute/volume"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"
)

const errNotCubeServer = "managed resource is not a Cube Server custom resource"

// Setup adds a controller that reconciles Server managed resources.
func Setup(mgr ctrl.Manager, opts *utils.ConfigurationOptions) error {
	name := managed.ControllerName(v1alpha1.CubeServerGroupKind)
	logger := opts.CtrlOpts.Logger

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(opts.CtrlOpts.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.CubeServer{}).
		Complete(managed.NewReconciler(mgr,
			resource.ManagedKind(v1alpha1.CubeServerGroupVersionKind),
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
			managed.WithPollIntervalHook(func(mg resource.Managed, pollInterval time.Duration) time.Duration {
				if mg.GetCondition(xpv1.TypeReady).Status != v1.ConditionTrue {
					// If the resource is not ready, we should poll more frequently not to delay time to readiness.
					pollInterval = 30 * time.Second
				}
				// This is the same as runtime default poll interval with jitter, see:
				// https://github.com/crossplane/crossplane-runtime/blob/7fcb8c5cad6fc4abb6649813b92ab92e1832d368/pkg/reconciler/managed/reconciler.go#L573
				return pollInterval + time.Duration((rand.Float64()-0.5)*2*float64(opts.PollJitter)) //nolint G404 // No need for secure randomness
			}),
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
	_, ok := mg.(*v1alpha1.CubeServer)
	if !ok {
		return nil, errors.New(errNotCubeServer)
	}
	svc, err := clients.ConnectForCRD(ctx, mg, c.kube, c.usage)
	return &externalServer{
		service:              &server.APIClient{IonosServices: svc},
		serviceVolume:        &volume.APIClient{IonosServices: svc},
		serviceTemplate:      &template.APIClient{IonosServices: svc},
		log:                  c.log,
		isUniqueNamesEnabled: c.isUniqueNamesEnabled}, err
}

// An ExternalClient observes, then either creates, updates, or deletes an
// externalServer resource to ensure it reflects the managed resource's desired state.
type externalServer struct {
	// A 'client' used to connect to the externalServer resource API. In practice this
	// would be something like an IONOS Cloud SDK client.
	service              server.Client
	serviceVolume        volume.Client
	serviceTemplate      template.Client
	log                  logging.Logger
	isUniqueNamesEnabled bool
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
		return managed.ExternalObservation{}, compute.ErrorUnlessNotFound(apiResponse, retErr)
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
	if instance.Properties != nil {
		cr.Status.AtProvider.Name = *instance.Properties.Name
	}
	c.log.Debug(fmt.Sprintf("Observing state %v.te"+
		"..", cr.Status.AtProvider.State))
	// Set Ready condition based on State
	clients.UpdateCondition(cr, cr.Status.AtProvider.State)

	return managed.ExternalObservation{
		ResourceExists:          true,
		ResourceUpToDate:        server.IsCubeServerUpToDate(cr, instance),
		ConnectionDetails:       managed.ConnectionDetails{},
		ResourceLateInitialized: !cmp.Equal(current, &cr.Spec.ForProvider),
	}, nil
}

func (c *externalServer) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) { // nolint: gocyclo
	cr, ok := mg.(*v1alpha1.CubeServer)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotCubeServer)
	}
	cr.SetConditions(xpv1.Creating())
	if cr.Status.AtProvider.State == compute.BUSY {
		return managed.ExternalCreation{}, nil
	}
	// Resolve TemplateID
	templateID, err := getTemplateID(ctx, c, cr)
	if err != nil {
		return managed.ExternalCreation{}, err
	}
	if c.isUniqueNamesEnabled {
		// Servers should have unique names per datacenter.
		// Check if there are any existing servers with the same name.
		// If there are multiple, an error will be returned.
		cubeDuplicateID, err := c.service.CheckDuplicateCubeServer(ctx, cr.Spec.ForProvider.DatacenterCfg.DatacenterID, cr.Spec.ForProvider.Name, templateID)
		if err != nil {
			return managed.ExternalCreation{}, err
		}
		if cubeDuplicateID != "" {
			// "Import" existing server.
			cr.Status.AtProvider.ServerID = cubeDuplicateID
			meta.SetExternalName(cr, cubeDuplicateID)
			return managed.ExternalCreation{}, nil
		}
	}

	// Create new cube server based on the properties set
	instanceInput, err := server.GenerateCreateCubeServerInput(cr, templateID)
	if err != nil {
		return managed.ExternalCreation{}, err
	}
	newInstance, apiResponse, err := c.service.CreateServer(ctx, cr.Spec.ForProvider.DatacenterCfg.DatacenterID, *instanceInput)
	creation := managed.ExternalCreation{ConnectionDetails: managed.ConnectionDetails{}}
	if err != nil {
		retErr := fmt.Errorf("failed to create cube server. error: %w", err)
		return creation, compute.AddAPIResponseInfo(apiResponse, retErr)
	}
	if err = compute.WaitForRequest(ctx, c.service.GetAPIClient(), apiResponse); err != nil {
		return creation, err
	}
	// Set External Name
	cr.Status.AtProvider.ServerID = *newInstance.Id
	meta.SetExternalName(cr, *newInstance.Id)
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

func (c *externalServer) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.CubeServer)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotCubeServer)
	}

	cr.SetConditions(xpv1.Deleting())
	if cr.Status.AtProvider.State == compute.DESTROYING {
		return managed.ExternalDelete{}, nil
	}

	// Deleting the CUBE Server will also delete the DAS Volume
	apiResponse, err := c.service.DeleteServer(ctx, cr.Spec.ForProvider.DatacenterCfg.DatacenterID, cr.Status.AtProvider.ServerID)
	if err != nil {
		retErr := fmt.Errorf("failed to delete cube server. error: %w", err)
		return managed.ExternalDelete{}, compute.ErrorUnlessNotFound(apiResponse, retErr)
	}
	if err = compute.WaitForRequest(ctx, c.service.GetAPIClient(), apiResponse); err != nil {
		return managed.ExternalDelete{}, err
	}
	return managed.ExternalDelete{}, nil
}

func getTemplateID(ctx context.Context, c *externalServer, cr *v1alpha1.CubeServer) (string, error) {
	if cr.Spec.ForProvider.Template.TemplateID != "" {
		return cr.Spec.ForProvider.Template.TemplateID, nil
	} else if cr.Spec.ForProvider.Template.Name != "" {
		if templateID, err := c.serviceTemplate.GetTemplateIDByName(ctx, cr.Spec.ForProvider.Template.Name); err == nil && templateID != "" {
			return templateID, nil
		}
	}
	return "", fmt.Errorf("error getting template ID")
}

// Disconnect does nothing because there are no resources to release. Needs to be implemented starting from crossplane-runtime v0.17
func (c *externalServer) Disconnect(_ context.Context) error {
	return nil
}
