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

package volume

import (
	"context"
	"fmt"
	"net/http"

	"github.com/crossplane/crossplane-runtime/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/pkg/statemetrics"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	sdkgo "github.com/ionos-cloud/sdk-go/v6"

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
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute/volume"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"
)

const errNotVolume = "managed resource is not a Volume custom resource"

// Setup adds a controller that reconciles Volume managed resources.
func Setup(mgr ctrl.Manager, opts *utils.ConfigurationOptions) error {
	name := managed.ControllerName(v1alpha1.VolumeGroupKind)
	logger := opts.CtrlOpts.Logger
	if opts.CtrlOpts.MetricOptions != nil && opts.CtrlOpts.MetricOptions.MRStateMetrics != nil {
		stateMetricsRecorder := statemetrics.NewMRStateRecorder(
			mgr.GetClient(), opts.CtrlOpts.Logger, opts.CtrlOpts.MetricOptions.MRStateMetrics, &v1alpha1.VolumeList{}, opts.CtrlOpts.MetricOptions.PollStateMetricInterval,
		)
		if err := mgr.Add(stateMetricsRecorder); err != nil {
			return errors.Wrap(err, "cannot register MR state metrics recorder for kind"+name)
		}
	}

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: opts.GetMaxConcurrentReconcileRate(v1alpha1.VolumeKind),
			RateLimiter:             ratelimiter.NewController(),
			RecoverPanic:            ptr.To(true),
		}).
		For(&v1alpha1.Volume{}).
		Complete(managed.NewReconciler(mgr,
			resource.ManagedKind(v1alpha1.VolumeGroupVersionKind),
			managed.WithExternalConnecter(&connectorVolume{
				kube:                 mgr.GetClient(),
				usage:                resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
				log:                  logger,
				isUniqueNamesEnabled: opts.GetIsUniqueNamesEnabled()}),
			managed.WithPollInterval(opts.GetPollInterval()),
			managed.WithCreationGracePeriod(opts.GetCreationGracePeriod()),
			managed.WithTimeout(opts.GetTimeout()),
			managed.WithReferenceResolver(managed.NewAPISimpleReferenceResolver(mgr.GetClient())),
			managed.WithLogger(logger.WithValues("controller", name)),
			managed.WithMetricRecorder(opts.CtrlOpts.MetricOptions.MRMetrics),
			managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name)))))
}

// A connectorVolume is expected to produce an ExternalClient when its Connect method
// is called.
type connectorVolume struct {
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
func (c *connectorVolume) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	_, ok := mg.(*v1alpha1.Volume)
	if !ok {
		return nil, errors.New(errNotVolume)
	}
	svc, err := clients.ConnectForCRD(ctx, mg, c.kube, c.usage)
	return &externalVolume{
		service:              &volume.APIClient{IonosServices: svc},
		log:                  c.log,
		isUniqueNamesEnabled: c.isUniqueNamesEnabled}, err
}

// An ExternalClient observes, then either creates, updates, or deletes an
// externalVolume resource to ensure it reflects the managed resource's desired state.
type externalVolume struct {
	// A 'client' used to connect to the externalVolume resource API. In practice this
	// would be something like an IONOS Cloud SDK client.
	service              volume.Client
	log                  logging.Logger
	isUniqueNamesEnabled bool
}

func (c *externalVolume) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.Volume)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotVolume)
	}

	// External Name of the CR is the Volume ID
	if meta.GetExternalName(cr) == "" {
		return managed.ExternalObservation{}, nil
	}
	instance, apiResponse, err := c.service.GetVolume(ctx, cr.Spec.ForProvider.DatacenterCfg.DatacenterID, meta.GetExternalName(cr))
	if err != nil {
		retErr := fmt.Errorf("failed to get volume by id. error: %w", err)
		return managed.ExternalObservation{}, compute.ErrorUnlessNotFound(apiResponse, retErr)
	}

	current := cr.Spec.ForProvider.DeepCopy()
	LateStatusInitializer(&cr.Status.AtProvider, &instance)
	cr.Status.AtProvider.VolumeID = meta.GetExternalName(cr)
	cr.Status.AtProvider.State = clients.GetCoreResourceState(&instance)
	if instance.Properties != nil {
		cr.Status.AtProvider.Name = *instance.Properties.Name
		cr.Status.AtProvider.Size = *instance.Properties.Size
		if instance.Properties.BootServer != nil {

			name, err := c.service.GetServerNameByID(ctx, cr.Spec.ForProvider.DatacenterCfg.DatacenterID, *instance.Properties.BootServer)
			if err != nil {
				return managed.ExternalObservation{}, err
			}
			cr.Status.AtProvider.ServerName = name
		}
	}
	c.log.Debug(fmt.Sprintf("Observing state: %v", cr.Status.AtProvider.State))
	// Set Ready condition based on State
	clients.UpdateCondition(cr, cr.Status.AtProvider.State)

	return managed.ExternalObservation{
		ResourceExists:          true,
		ResourceUpToDate:        volume.IsVolumeUpToDate(cr, &instance),
		ConnectionDetails:       managed.ConnectionDetails{},
		ResourceLateInitialized: !cmp.Equal(current, &cr.Spec.ForProvider),
	}, nil
}

// LateStatusInitializer fills the empty fields in *v1alpha1.VolumeObservation with
// the values seen in sdkgo.Volume.
func LateStatusInitializer(in *v1alpha1.VolumeObservation, volume *sdkgo.Volume) {
	if volume == nil {
		return
	}
	// Add options to the Spec, if they were updated by the API
	if propertiesOk, ok := volume.GetPropertiesOk(); ok && propertiesOk != nil {
		if PCISlotOk, ok := propertiesOk.GetPciSlotOk(); ok && PCISlotOk != nil {
			if in.PCISlot == 0 {
				in.PCISlot = *PCISlotOk
			}
		}
	}
}

func (c *externalVolume) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.Volume)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotVolume)
	}
	cr.SetConditions(xpv1.Creating())
	if cr.Status.AtProvider.State == compute.BUSY {
		return managed.ExternalCreation{}, nil
	}

	if externalName := meta.GetExternalName(cr); externalName != "" && externalName != cr.Name {
		isDone, err := compute.IsRequestDone(ctx, c.service.GetAPIClient(), externalName, http.MethodPost)
		if err != nil {
			return managed.ExternalCreation{}, err
		}

		if isDone {
			return managed.ExternalCreation{ConnectionDetails: managed.ConnectionDetails{}}, nil
		}

		return managed.ExternalCreation{}, nil
	}

	if c.isUniqueNamesEnabled {
		// Volumes should have unique names per datacenter.
		// Check if there are any existing volumes with the same name.
		// If there are multiple, an error will be returned.
		instance, err := c.service.CheckDuplicateVolume(ctx, cr.Spec.ForProvider.DatacenterCfg.DatacenterID,
			cr.Spec.ForProvider.Name, cr.Spec.ForProvider.Type, cr.Spec.ForProvider.AvailabilityZone,
			cr.Spec.ForProvider.LicenceType, cr.Spec.ForProvider.Image)
		if err != nil {
			return managed.ExternalCreation{}, err
		}
		volumeID, err := c.service.GetVolumeID(instance)
		if err != nil {
			return managed.ExternalCreation{}, err
		}
		if volumeID != "" {
			// "Import" existing volume.
			cr.Status.AtProvider.VolumeID = volumeID
			meta.SetExternalName(cr, volumeID)
			return managed.ExternalCreation{}, nil
		}
	}

	instanceInput, err := volume.GenerateCreateVolumeInput(cr)
	if err != nil {
		return managed.ExternalCreation{}, err
	}
	newInstance, apiResponse, err := c.service.CreateVolume(ctx, cr.Spec.ForProvider.DatacenterCfg.DatacenterID, *instanceInput)
	creation := managed.ExternalCreation{ConnectionDetails: managed.ConnectionDetails{}}
	if err != nil {
		retErr := fmt.Errorf("failed to create volume. error: %w", err)
		return creation, compute.AddAPIResponseInfo(apiResponse, retErr)
	}

	// Set External Name
	cr.Status.AtProvider.VolumeID = *newInstance.Id
	meta.SetExternalName(cr, *newInstance.Id)

	if err = compute.WaitForRequest(ctx, c.service.GetAPIClient(), apiResponse); err != nil {
		return creation, err
	}
	return creation, nil
}

func (c *externalVolume) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.Volume)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotVolume)
	}
	if cr.Status.AtProvider.State == compute.BUSY || cr.Status.AtProvider.State == compute.UPDATING {
		return managed.ExternalUpdate{}, nil
	}

	volumeID := cr.Status.AtProvider.VolumeID
	// Get the current Volume
	instanceObserved, apiResponse, err := c.service.GetVolume(ctx, cr.Spec.ForProvider.DatacenterCfg.DatacenterID, volumeID)
	if err != nil {
		retErr := fmt.Errorf("failed to get volume by id. error: %w", err)
		return managed.ExternalUpdate{}, compute.ErrorUnlessNotFound(apiResponse, retErr)
	}
	instanceInput, err := volume.GenerateUpdateVolumeInput(cr, instanceObserved.Properties)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}
	_, apiResponse, err = c.service.UpdateVolume(ctx, cr.Spec.ForProvider.DatacenterCfg.DatacenterID, volumeID, *instanceInput)
	if err != nil {
		retErr := fmt.Errorf("failed to update volume. error: %w", err)
		return managed.ExternalUpdate{}, compute.AddAPIResponseInfo(apiResponse, retErr)
	}
	if err = compute.WaitForRequest(ctx, c.service.GetAPIClient(), apiResponse); err != nil {
		return managed.ExternalUpdate{}, err
	}
	return managed.ExternalUpdate{}, nil
}

func (c *externalVolume) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.Volume)
	if cr.Status.AtProvider.State == compute.DESTROYING {
		return managed.ExternalDelete{}, nil
	}
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotVolume)
	}

	cr.SetConditions(xpv1.Deleting())
	if cr.Status.AtProvider.State == compute.DESTROYING {
		return managed.ExternalDelete{}, nil
	}

	apiResponse, err := c.service.DeleteVolume(ctx, cr.Spec.ForProvider.DatacenterCfg.DatacenterID, cr.Status.AtProvider.VolumeID)
	if err != nil {
		retErr := fmt.Errorf("failed to delete volume. error: %w", err)
		return managed.ExternalDelete{}, compute.ErrorUnlessNotFound(apiResponse, retErr)
	}
	if err = compute.WaitForRequest(ctx, c.service.GetAPIClient(), apiResponse); err != nil {
		return managed.ExternalDelete{}, err
	}
	return managed.ExternalDelete{}, nil
}

// Disconnect does nothing because there are no resources to release. Needs to be implemented starting from crossplane-runtime v0.17
func (c *externalVolume) Disconnect(_ context.Context) error {
	return nil
}
