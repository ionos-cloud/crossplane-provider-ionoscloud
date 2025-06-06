/*
Copyright 2022 The Crossplane Authors.

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

package statefulserverset

import (
	"context"
	"fmt"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	ionoscloud "github.com/ionos-cloud/sdk-go/v6"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute/server"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/controller/serverset"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/controller/volumeselector"
)

const (
	errNotStatefulServerSet = "managed resource is not a StatefulServerSet custom resource"
	errTrackPCUsage         = "cannot track ProviderConfig usage"
)

const statefulServerSetLabel = "statefulServerSet"

// A NoOpService does nothing.
type NoOpService struct{}

// A connector is expected to produce an ExternalClient when its Connect method
// is called.
type connector struct {
	kube                     client.Client
	usage                    resource.Tracker
	log                      logging.Logger
	dataVolumeController     kubeDataVolumeControlManager
	LANController            kubeLANControlManager
	SSetController           kubeSSetControlManager
	volumeSelectorController kubeVolumeSelectorManager
}

// Connect typically produces an ExternalClient by:
// 1. Tracking that the managed resource is using a ProviderConfig.
// 2. Getting the managed resource's ProviderConfig.
// 3. Getting the credentials specified by the ProviderConfig.
// 4. Using the credentials to form a client.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	_, ok := mg.(*v1alpha1.StatefulServerSet)
	if !ok {
		return nil, errors.New(errNotStatefulServerSet)
	}

	if err := c.usage.Track(ctx, mg); err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	svc, err := clients.ConnectForCRD(ctx, mg, c.kube, c.usage)
	if err != nil {
		return nil, err
	}
	return &external{
		kube:                     c.kube,
		service:                  &server.APIClient{IonosServices: svc},
		dataVolumeController:     c.dataVolumeController,
		LANController:            c.LANController,
		SSetController:           c.SSetController,
		volumeSelectorController: c.volumeSelectorController,
		log:                      c.log,
	}, err
}

// external observes, then either creates, updates, or deletes an
// externalServerSet resource to ensure it reflects the managed resource's desired state.
type external struct {
	kube                     client.Client
	service                  server.Client
	dataVolumeController     kubeDataVolumeControlManager
	LANController            kubeLANControlManager
	SSetController           kubeSSetControlManager
	volumeSelectorController kubeVolumeSelectorManager
	log                      logging.Logger
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.StatefulServerSet)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotStatefulServerSet)
	}

	if meta.GetExternalName(cr) == "" {
		return managed.ExternalObservation{}, nil
	}

	if meta.WasDeleted(cr) {
		return managed.ExternalObservation{}, nil
	}

	areResourcesCreated, areResourcesUpdated, areResourcesAvailable, err := e.observeResourcesUpdateStatus(ctx, cr)
	if err != nil {
		return managed.ExternalObservation{}, err
	}
	if areResourcesCreated && areResourcesUpdated && areResourcesAvailable {
		cr.SetConditions(xpv1.Available())
	} else {
		cr.SetConditions(xpv1.Creating())
	}

	return managed.ExternalObservation{
		// Return false when the externalStatefulServerSet resource does not exist. This lets
		// the managed resource reconciler know that it needs to call Create to
		// (re)create the resource, or that it has successfully been deleted.
		ResourceExists: areResourcesCreated,

		// Return false when the externalStatefulServerSet resource exists, but it not up to date
		// with the desired managed resource state. This lets the managed
		// resource reconciler know that it needs to call Update.
		ResourceUpToDate: areResourcesUpdated,

		// Return any details that may be required to connect to the externalStatefulServerSet
		// resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (e *external) observeResourcesUpdateStatus(ctx context.Context, cr *v1alpha1.StatefulServerSet) (areResourcesCreated, areResourcesUpdated, areResourcesAvailable bool, err error) { // nolint:gocyclo

	// ******************* LANS *******************
	lans, err := e.LANController.ListLans(ctx, cr)
	if err != nil {
		return false, false, false, fmt.Errorf("while listing lans %w", err)
	}
	creationLansUpToDate, lansUpToDate := areLansUpToDate(cr, lans.Items)
	cr.Status.AtProvider.LanStatuses = computeLanStatuses(lans.Items)

	// ******************* VOLUMES *******************
	volumes, err := e.dataVolumeController.ListVolumes(ctx, cr)
	if err != nil {
		return false, false, false, fmt.Errorf("while listing volumes %w", err)
	}
	creationVolumesUpToDate, areVolumesUpToDate, areVolumesAvailable := areDataVolumesUpToDateAndAvailable(cr, volumes.Items, e.log)
	cr.Status.AtProvider.DataVolumeStatuses = computeVolumeStatuses(e.log, cr.Spec.ForProvider.Template.Metadata.Name, volumes.Items)

	// ******************* SERVERSET *******************
	creationSSetUpToDate, isSSetUpToDate, isSSetAvailable, err := e.isServerSetUpToDate(ctx, cr)
	if err != nil {
		return false, false, false, err
	}
	err = e.setSSetStatusOnCR(ctx, cr)
	if err != nil {
		return false, false, false, err
	}

	// ******************* VOLUMESELECTOR *******************
	creationVSUpToDate, err := e.isVolumeSelectorUpToDate(ctx, cr)
	if err != nil {
		return false, false, false, err
	}
	e.log.Info("Observing the StatefulServerSet", "name", cr.Name, "creationLansUpToDate", creationLansUpToDate, "lansUpToDate", lansUpToDate, "creationVolumesUpToDate", creationVolumesUpToDate,
		"areVolumesUpToDate", areVolumesUpToDate, "creationSSetUpToDate", creationSSetUpToDate, "isSSetUpToDate", isSSetUpToDate, "creationVSUpToDate", creationVSUpToDate, "areVolumesAvailable", areVolumesAvailable)
	areResourcesCreated = creationLansUpToDate && creationVolumesUpToDate && creationSSetUpToDate && creationVSUpToDate
	areResourcesUpdated = lansUpToDate && areVolumesUpToDate && isSSetUpToDate
	areResourcesAvailable = areVolumesAvailable && isSSetAvailable
	return areResourcesCreated, areResourcesUpdated, areResourcesAvailable, nil
}

func (e *external) isServerSetUpToDate(ctx context.Context, cr *v1alpha1.StatefulServerSet) (creationServerUpToDate, serversetUpToDate, ssetAvailable bool, err error) {
	_, err = e.SSetController.Get(ctx, getSSetName(cr), cr.Namespace)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			return false, false, false, nil
		}
	}
	serversetUpToDate, ssetAvailable, err = areSSetResourcesReady(ctx, e.kube, cr)
	if err != nil {
		return false, false, false, err
	}
	return true, serversetUpToDate, ssetAvailable, err
}

func (e *external) isVolumeSelectorUpToDate(ctx context.Context, cr *v1alpha1.StatefulServerSet) (creationVSUpToDate bool, err error) {
	vsName := fmt.Sprintf(volumeSelectorName, cr.Name)
	_, err = e.volumeSelectorController.Get(ctx, vsName, cr.Namespace)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return e.volumeSelectorController.IsAvailable(ctx, vsName, cr.Namespace)
}

func (e *external) setSSetStatusOnCR(ctx context.Context, cr *v1alpha1.StatefulServerSet) error {
	sSet, err := e.SSetController.Get(ctx, getSSetName(cr), cr.Namespace)
	if err != nil {
		if !apiErrors.IsNotFound(err) {
			return err
		}
	}
	if sSet != nil {
		cr.Status.AtProvider.ReplicaStatuses = sSet.Status.AtProvider.ReplicaStatuses
		cr.Status.AtProvider.Replicas = sSet.Status.AtProvider.Replicas
	}
	return nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.StatefulServerSet)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotStatefulServerSet)
	}

	cr.SetConditions(xpv1.Creating())

	// volume selector attaches data volumes to the servers created in the serverset
	if err := e.volumeSelectorController.CreateOrUpdate(ctx, cr); err != nil {
		return managed.ExternalCreation{}, err
	}

	e.log.Info("Creating a new StatefulServerSet", "name", cr.Name, "replicas", cr.Spec.ForProvider.Replicas)
	for replicaIndex := 0; replicaIndex < cr.Spec.ForProvider.Replicas; replicaIndex++ {
		err := e.ensureDataVolumes(ctx, cr, replicaIndex)
		if err != nil {
			return managed.ExternalCreation{}, fmt.Errorf("while ensuring DataVolumes %w", err)
		}

	}
	if err := e.ensureLans(ctx, cr); err != nil {
		return managed.ExternalCreation{}, fmt.Errorf("while ensuring LANs %w", err)
	}

	if err := e.SSetController.Ensure(ctx, cr); err != nil {
		return managed.ExternalCreation{}, fmt.Errorf("while ensuring ServerSet %w", err)
	}

	// When all conditions are met, the managed resource is considered available
	meta.SetExternalName(cr, cr.Name)
	return managed.ExternalCreation{
		// Optionally return any details that may be required to connect to the
		// externalStatefulServerSet resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.StatefulServerSet)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotStatefulServerSet)
	}
	for replicaIndex := 0; replicaIndex < cr.Spec.ForProvider.Replicas; replicaIndex++ {
		for volumeIndex := range cr.Spec.ForProvider.Volumes {
			_, err := e.dataVolumeController.Update(ctx, cr, replicaIndex, volumeIndex)
			if err != nil {
				return managed.ExternalUpdate{}, err
			}
		}
	}
	for lanIndex := range cr.Spec.ForProvider.Lans {
		_, err := e.LANController.Update(ctx, cr, lanIndex)
		if err != nil {
			return managed.ExternalUpdate{}, err
		}
	}
	_, err := e.SSetController.Update(ctx, cr, false)
	if err != nil {
		return managed.ExternalUpdate{}, fmt.Errorf("while updating ServerSet CR %w", err)
	}
	return managed.ExternalUpdate{
		// Optionally return any details that may be required to connect to the
		// externalStatefulServerSet resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (e *external) Delete(_ context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.StatefulServerSet)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotStatefulServerSet)
	}
	cr.SetConditions(xpv1.Deleting())

	e.log.Info("Successfully deleted StatefulServerSet", "name", cr.Name)
	return managed.ExternalDelete{}, nil
}

func (e *external) ensureDataVolumes(ctx context.Context, cr *v1alpha1.StatefulServerSet, replicaIndex int) error {
	e.log.Info("Ensuring the DataVolumes for ", "name", cr.Name)
	errGroup, ctx := errgroup.WithContext(ctx)
	for volumeIndex := range cr.Spec.ForProvider.Volumes {
		errGroup.Go(func() error { return e.dataVolumeController.Ensure(ctx, cr, replicaIndex, volumeIndex) })
	}
	return errGroup.Wait()
}

func (e *external) ensureLans(ctx context.Context, cr *v1alpha1.StatefulServerSet) error {
	e.log.Info("Ensuring the LANs for", "name", cr.Name)
	errGroup, ctx := errgroup.WithContext(ctx)
	for lanIndex := range cr.Spec.ForProvider.Lans {
		errGroup.Go(func() error { return e.LANController.Ensure(ctx, cr, lanIndex) })
	}
	return errGroup.Wait()
}

func areLansUpToDate(cr *v1alpha1.StatefulServerSet, lans []v1alpha1.Lan) (bool, bool) {
	if len(lans) != len(cr.Spec.ForProvider.Lans) {
		return false, false
	}

	for _, gotLan := range lans {
		for _, specLan := range cr.Spec.ForProvider.Lans {
			if isALanFieldNotUpToDate(specLan, gotLan) {
				return true, false
			}
		}
	}

	return true, true
}

func isALanFieldNotUpToDate(specLan v1alpha1.StatefulServerSetLan, gotLan v1alpha1.Lan) bool {
	if specLan.Metadata.Name != gotLan.Spec.ForProvider.Name {
		return false
	}
	if gotLan.Spec.ForProvider.Public != specLan.Spec.Public {
		return true
	}
	if specLan.Spec.IPv6cidr != v1alpha1.LANAuto && gotLan.Spec.ForProvider.Ipv6Cidr != specLan.Spec.IPv6cidr {
		return true
	}
	return false
}

func areDataVolumesUpToDateAndAvailable(cr *v1alpha1.StatefulServerSet, volumes []v1alpha1.Volume, log logging.Logger) (creationVolumesUpToDate, areVolumesUpToDate, volumesAvailable bool) {
	crExpectedNrOfVolumes := len(cr.Spec.ForProvider.Volumes) * cr.Spec.ForProvider.Replicas
	if len(volumes) != crExpectedNrOfVolumes {
		return false, false, false
	}
	for volumeIndex := range volumes {
		for _, specVolume := range cr.Spec.ForProvider.Volumes {
			// there can be multiple volumes, so we need to match names before checking size and updating
			idxLabel := fmt.Sprintf(volumeselector.IndexLabel, cr.Spec.ForProvider.Template.Metadata.Name, volumeselector.ResourceDataVolume)
			volVersionLabel := fmt.Sprintf(volumeselector.VolumeIndexLabel, cr.Spec.ForProvider.Template.Metadata.Name, volumeselector.ResourceDataVolume)
			replicaIndex := serverset.ComputeReplicaIdx(log, idxLabel, volumes[volumeIndex].Labels)
			version := serverset.ComputeReplicaIdx(log, volVersionLabel, volumes[volumeIndex].Labels)
			if replicaIndex == -1 || version == -1 {
				log.Info("DataVolume does not have the right labels", "name", volumes[volumeIndex].ObjectMeta.Name)
				return true, false, false
			}
			generatedName := generateNameFrom(specVolume.Metadata.Name, replicaIndex, version)

			if volumes[volumeIndex].ObjectMeta.Name == generatedName {
				if volumes[volumeIndex].Spec.ForProvider.Size != specVolume.Spec.Size {
					log.Info("DataVolume size is not up to date, triggering an update for", "name", volumes[volumeIndex].ObjectMeta.Name, "expectedSize", specVolume.Spec.Size, "actualSize", volumes[volumeIndex].Spec.ForProvider.Size)
					return true, false, false
				}
				if volumes[volumeIndex].Status.AtProvider.State != ionoscloud.Available {
					log.Info("DataVolume is not available", "name", volumes[volumeIndex].ObjectMeta.Name, "expectedState", ionoscloud.Available, "actualState", volumes[volumeIndex].Status.AtProvider.State)
					return true, true, false
				}
			}
		}
	}
	return true, true, true
}

func areSSetResourcesReady(ctx context.Context, kube client.Client, cr *v1alpha1.StatefulServerSet) (isSsetUpToDate, isSsetAvailable bool, err error) {
	serversUpToDate, areServersAvailable, err := areServersUpToDate(ctx, kube, cr)
	if !serversUpToDate {
		return false, false, err
	}

	bootVolumesUpToDate, areBootVolumesAvailable, err := areBootVolumesUpToDate(ctx, kube, cr)
	if !bootVolumesUpToDate {
		return false, false, err
	}

	areNICSUpToDate, err := areNICsUpToDate(ctx, kube, cr)
	if !areNICSUpToDate {
		return false, false, err
	}

	return true, areServersAvailable && areBootVolumesAvailable, nil
}

func areServersUpToDate(ctx context.Context, kube client.Client, cr *v1alpha1.StatefulServerSet) (areServersUpToDate, areServersAvailable bool, err error) {
	servers, err := serverset.GetServersOfSSet(ctx, kube, getSSetName(cr))
	if err != nil {
		return false, false, err
	}

	if len(servers) < cr.Spec.ForProvider.Replicas {
		return false, false, nil
	}
	areServersUpToDate, areServersAvailable = serverset.AreServersReady(cr.Spec.ForProvider.Template.Spec, servers)
	return areServersUpToDate, areServersAvailable, nil
}

func areBootVolumesUpToDate(ctx context.Context, kube client.Client, cr *v1alpha1.StatefulServerSet) (areUpToDate, areAvailable bool, err error) {
	volumes, err := serverset.GetVolumesOfSSet(ctx, kube, getSSetName(cr))
	if err != nil {
		return false, false, err
	}
	areUpToDate, areAvailable = serverset.AreBootVolumesReady(cr.Spec.ForProvider.BootVolumeTemplate, volumes)
	return areUpToDate, areAvailable, nil
}

func areNICsUpToDate(ctx context.Context, kube client.Client, cr *v1alpha1.StatefulServerSet) (bool, error) {
	nics, err := serverset.GetNICsOfSSet(ctx, kube, getSSetName(cr))
	if err != nil {
		return false, err
	}

	crExpectedNrOfNICs := len(cr.Spec.ForProvider.Template.Spec.NICs) * cr.Spec.ForProvider.Replicas
	if len(nics) != crExpectedNrOfNICs {
		return false, nil
	}

	return true, nil
}

func computeVolumeStatuses(log logging.Logger, serverName string, volumes []v1alpha1.Volume) []v1alpha1.StatefulServerSetVolumeStatus {
	if len(volumes) == 0 {
		return nil
	}
	status := make([]v1alpha1.StatefulServerSetVolumeStatus, len(volumes))
	for idx := range volumes {
		status[idx].VolumeStatus = volumes[idx].Status
		idxLabel := fmt.Sprintf(volumeselector.IndexLabel, serverName, volumeselector.ResourceDataVolume)
		status[idx].ReplicaIndex = serverset.ComputeReplicaIdx(log, idxLabel, volumes[idx].Labels)
	}
	return status
}

func computeLanStatuses(lans []v1alpha1.Lan) []v1alpha1.StatefulServerSetLanStatus {
	if len(lans) == 0 {
		return nil
	}
	status := make([]v1alpha1.StatefulServerSetLanStatus, len(lans))
	for idx := range lans {
		status[idx].LanStatus = lans[idx].Status
		status[idx].IPv6CIDRBlock = lans[idx].Spec.ForProvider.Ipv6Cidr
	}
	return status
}

// Disconnect does nothing because there are no resources to release. Needs to be implemented starting from crossplane-runtime v0.17
func (e *external) Disconnect(_ context.Context) error {
	return nil
}
