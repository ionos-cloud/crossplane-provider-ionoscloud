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

package serverset

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"

	ionoscloud "github.com/ionos-cloud/sdk-go/v6"
)

const (
	errUnexpectedObject = "managed resource is not an Volume resource"
	errTrackPCUsage     = "cannot track ProviderConfig usage"
)

const (
	// indexLabel is the label used to identify the server set by index
	indexLabel = "ionoscloud.com/%s-%s-index"
	// versionLabel is the label used to identify the server set by version
	versionLabel = "ionoscloud.com/%s-%s-version"
	// serverSetLabel is the label used to identify the server set resources. All resources created by a server set will have this label
	serverSetLabel = "ionoscloud.com/serverset"
)

// A connector is expected to produce an ExternalClient when its Connect method
// is called.
type connector struct {
	kube                     client.Client
	bootVolumeController     kubeBootVolumeControlManager
	nicController            kubeNicControlManager
	serverController         kubeServerControlManager
	volumeSelectorController kubeVolumeSelectorManager
	usage                    resource.Tracker
	log                      logging.Logger
}

// Connect typically produces an ExternalClient by:
// 1. Tracking that the managed resource is using a ProviderConfig.
// 2. Getting the managed resource's ProviderConfig.
// 3. Getting the credentials specified by the ProviderConfig.
// 4. Using the credentials to form a client.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	_, ok := mg.(*v1alpha1.ServerSet)
	if !ok {
		return nil, errors.New(errUnexpectedObject)
	}
	var err error
	if err = c.usage.Track(ctx, mg); err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	return &external{
		kube:                     c.kube,
		log:                      c.log,
		bootVolumeController:     c.bootVolumeController,
		nicController:            c.nicController,
		serverController:         c.serverController,
		volumeSelectorController: c.volumeSelectorController,
	}, err
}

// external observes, then either creates, updates, or deletes an
// externalServerSet resource to ensure it reflects the managed resource's desired state.
type external struct {
	kube client.Client
	// A 'client' used to connect to the externalServer resource API. In practice this
	// would be something like an IONOS Cloud SDK client.
	bootVolumeController     kubeBootVolumeControlManager
	nicController            kubeNicControlManager
	serverController         kubeServerControlManager
	volumeSelectorController kubeVolumeSelectorManager
	log                      logging.Logger
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.ServerSet)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errUnexpectedObject)
	}

	if meta.GetExternalName(cr) == "" {
		return managed.ExternalObservation{}, nil
	}

	servers, err := GetServersFromSSet(ctx, e.kube, cr.Name)
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	e.populateCRStatus(cr, servers)

	if len(servers) < cr.Spec.ForProvider.Replicas {
		return managed.ExternalObservation{
			ResourceExists:    false,
			ResourceUpToDate:  false,
			ConnectionDetails: managed.ConnectionDetails{},
		}, nil
	}

	areServersUpToDate := AreServersUpToDate(cr.Spec.ForProvider.Template.Spec, servers)

	volumes, err := GetVolumesFromSSet(ctx, e.kube, cr.Name)
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	areVolumesUpToDate := AreVolumesUpToDate(cr.Spec.ForProvider.BootVolumeTemplate, volumes)

	if !areServersUpToDate || !areVolumesUpToDate {
		return managed.ExternalObservation{
			ResourceExists:    true,
			ResourceUpToDate:  false,
			ConnectionDetails: managed.ConnectionDetails{},
			Diff:              "servers are not up to date",
		}, nil
	}

	areNicsUpToDate := false
	// todo check nic parameters are same as template
	if areNicsUpToDate, err = AreNICsUpToDate(ctx, e.kube, cr.GetName(), cr.Spec.ForProvider.Replicas); err != nil {
		return managed.ExternalObservation{}, err
	}

	if !areNicsUpToDate {
		return managed.ExternalObservation{
			ResourceExists:    false,
			ResourceUpToDate:  false,
			ConnectionDetails: managed.ConnectionDetails{},
		}, nil
	}
	cr.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		// Return false when the externalServerSet resource does not exist. This lets
		// the managed resource reconciler know that it needs to call Create to
		// (re)create the resource, or that it has successfully been deleted.
		ResourceExists: true,

		// Return false when the externalServerSet resource exists, but it not up to date
		// with the desired managed resource state. This lets the managed
		// resource reconciler know that it needs to call Update.
		ResourceUpToDate: true,

		// Return any details that may be required to connect to the externalServerSet
		// resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func didNrOfReplicasChange(cr *v1alpha1.ServerSet, replicas []v1alpha1.Server) bool {
	return len(replicas) != cr.Status.AtProvider.Replicas
}

func (e *external) populateCRStatus(cr *v1alpha1.ServerSet, serverSetReplicas []v1alpha1.Server) {
	if cr.Status.AtProvider.ReplicaStatuses == nil || didNrOfReplicasChange(cr, serverSetReplicas) {
		cr.Status.AtProvider.ReplicaStatuses = make([]v1alpha1.ServerSetReplicaStatus, cr.Spec.ForProvider.Replicas)
	}
	cr.Status.AtProvider.Replicas = len(serverSetReplicas)

	for i := range serverSetReplicas {
		replicaStatus := computeStatus(serverSetReplicas[i].Status.AtProvider.State)
		cr.Status.AtProvider.ReplicaStatuses[i] = v1alpha1.ServerSetReplicaStatus{
			Name:         serverSetReplicas[i].Name,
			Status:       replicaStatus,
			ErrorMessage: "",
			LastModified: metav1.Now(),
		}
	}
}

func computeStatus(state string) string {
	// At the moment we compute the status of the Server contained in the ServerSet
	// based on the status of the Server.
	switch state {
	case ionoscloud.Available:
		return statusReady
	case ionoscloud.Failed:
		return statusError
	}
	return statusUnknown
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.ServerSet)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errUnexpectedObject)
	}

	cr.Status.SetConditions(xpv1.Creating())

	// for n times of cr.Spec.Replicas, create a server
	// for each server, create a volume
	e.log.Info("Creating a new ServerSet", "replicas", cr.Spec.ForProvider.Replicas)
	version := 0
	for i := 0; i < cr.Spec.ForProvider.Replicas; i++ {
		if err := e.ensureBootVolumeByIndex(ctx, cr, i, version); err != nil {
			return managed.ExternalCreation{}, err
		}

		if err := e.ensureServerAndNicByIndex(ctx, cr, i, version); err != nil {
			return managed.ExternalCreation{}, err
		}

	}
	// volume selector attaches data volumes to the servers created in the serverset
	if err := e.volumeSelectorController.CreateOrUpdate(ctx, cr); err != nil {
		return managed.ExternalCreation{}, err
	}

	// When all conditions are met, the managed resource is considered available
	meta.SetExternalName(cr, cr.Name)
	return managed.ExternalCreation{
		// Optionally return any details that may be required to connect to the
		// externalServerSet resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.ServerSet)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errUnexpectedObject)
	}

	if meta.GetExternalName(cr) == "" {
		return managed.ExternalUpdate{}, nil
	}
	// how do we know if we want to update servers or nic params?
	err := e.updateServersFromTemplate(ctx, cr)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}
	if err := e.reconcileVolumesFromTemplate(ctx, cr); err != nil {
		return managed.ExternalUpdate{}, err

	}

	return managed.ExternalUpdate{
		// Optionally return any details that may be required to connect to the
		// externalServerSet resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (e *external) updateServersFromTemplate(ctx context.Context, cr *v1alpha1.ServerSet) error {
	servers, err := GetServersFromSSet(ctx, e.kube, cr.Name)
	if err != nil {
		return err
	}
	for idx := range servers {
		update := false
		if servers[idx].Spec.ForProvider.RAM != cr.Spec.ForProvider.Template.Spec.RAM {
			update = true
			servers[idx].Spec.ForProvider.RAM = cr.Spec.ForProvider.Template.Spec.RAM
		}
		if servers[idx].Spec.ForProvider.Cores != cr.Spec.ForProvider.Template.Spec.Cores {
			update = true
			servers[idx].Spec.ForProvider.Cores = cr.Spec.ForProvider.Template.Spec.Cores
		}
		if servers[idx].Spec.ForProvider.CPUFamily != cr.Spec.ForProvider.Template.Spec.CPUFamily {
			update = true
			servers[idx].Spec.ForProvider.CPUFamily = cr.Spec.ForProvider.Template.Spec.CPUFamily
		}
		if update {
			if err := e.kube.Update(ctx, &servers[idx]); err != nil {
				return fmt.Errorf("error updating server %w", err)
			}
		}
	}
	return nil
}

// reconcileVolumesFromTemplate updates bootvolume, or deletes and re-creates server, volume and nic if something
// immutable changes in a bootvolume
func (e *external) reconcileVolumesFromTemplate(ctx context.Context, cr *v1alpha1.ServerSet) error {
	volumes, err := GetVolumesFromSSet(ctx, e.kube, cr.Name)
	if err != nil {
		return err
	}
	masterIndex, err := e.getIdentityFromConfigMap(ctx, *cr)
	if err != nil {
		return fmt.Errorf("while getting master index %w", err)
	}
	err = e.updateOrRecreateVolumes(ctx, cr, volumes, masterIndex)
	if err != nil {
		return fmt.Errorf("while updating volumes %w", err)
	}
	return nil
}

func (e *external) updateOrRecreateVolumes(ctx context.Context, cr *v1alpha1.ServerSet, volumes []v1alpha1.Volume, masterIndex int) error {
	recreateMaster := false
	for idx := range volumes {
		update := false
		deleteAndCreate := false
		update, deleteAndCreate = updateOrRecreate(&volumes[idx].Spec.ForProvider, cr.Spec.ForProvider.BootVolumeTemplate.Spec)
		if deleteAndCreate {
			// we want to recreate master at the end
			if masterIndex == idx {
				recreateMaster = true
				continue
			}
			err := e.updateByIndex(ctx, idx, cr)
			if err != nil {
				return err
			}
		} else if update {
			if err := e.kube.Update(ctx, &volumes[idx]); err != nil {
				return fmt.Errorf("error updating server %w", err)
			}
		}
	}
	if masterIndex != -1 {
		e.log.Info("updating master with", "index", masterIndex, "template", cr.Spec.ForProvider.BootVolumeTemplate.Spec)
		if recreateMaster {
			err := e.updateByIndex(ctx, masterIndex, cr)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (e *external) updateByIndex(ctx context.Context, idx int, cr *v1alpha1.ServerSet) error {
	volumeVersion, serverVersion, err := getVersionsFromVolumeAndServer(ctx, e.kube, cr.GetName(), idx)
	if err != nil {
		return err
	}
	updater := e.getUpdaterByStrategy(cr.Spec.ForProvider.BootVolumeTemplate.Spec.UpdateStrategy.Stype)
	return updater.update(ctx, cr, idx, volumeVersion, serverVersion)
}

func (e *external) getIdentityFromConfigMap(ctx context.Context, cr v1alpha1.ServerSet) (int, error) {
	configMap := &v1.ConfigMap{}
	if cr.Namespace == "" {
		cr.Namespace = "default"
	}
	err := e.kube.Get(ctx, client.ObjectKey{Namespace: cr.Namespace, Name: "config-lease"}, configMap)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			return -1, nil
		}
		return -1, err
	}

	masterIndex := -1
	if configMap.Data["identity"] != "" {
		sliceOfValues := strings.Split(configMap.Data["identity"], "-")
		if len(sliceOfValues) > 2 {
			tmpIndex, err := strconv.Atoi(sliceOfValues[2])
			// annoying, but we need to check if the index is valid
			if err == nil {
				masterIndex = tmpIndex
			}
		}
	}

	return masterIndex, nil
}

func (e *external) getUpdaterByStrategy(strategyType v1alpha1.UpdateStrategyType) updater {
	switch strategyType {
	case v1alpha1.CreateAllBeforeDestroy:
		return newCreateBeforeDestroy(e.bootVolumeController, e.serverController, e.nicController)
	case v1alpha1.CreateBeforeDestroyBootVolume:
		return newCreateBeforeDestroyOnlyBootVolume(e.bootVolumeController, e.serverController)
	default:
		return newCreateBeforeDestroyOnlyBootVolume(e.bootVolumeController, e.serverController)
	}
}

// updateOrRecreate checks if bootvolume parameters are equal to bootvolume template parameters
// mutates volume parameters if fields are not equal
func updateOrRecreate(volumeParams *v1alpha1.VolumeParameters, volumeSpec v1alpha1.ServerSetBootVolumeSpec) (update bool, deleteAndCreate bool) {
	if volumeParams.Size != volumeSpec.Size {
		update = true
		volumeParams.Size = volumeSpec.Size
	}
	if volumeParams.Type != volumeSpec.Type {
		deleteAndCreate = true
		volumeParams.Type = volumeSpec.Type
	}

	if volumeParams.Image != volumeSpec.Image {
		deleteAndCreate = true
		volumeParams.Image = volumeSpec.Image
	}
	return update, deleteAndCreate
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*v1alpha1.ServerSet)
	if !ok {
		return errors.New(errUnexpectedObject)
	}

	cr.SetConditions(xpv1.Deleting())
	meta.SetExternalName(cr, "")
	if err := e.kube.DeleteAllOf(ctx, &v1alpha1.Nic{}, client.InNamespace(cr.Namespace), client.MatchingLabels{
		serverSetLabel: cr.Name,
	}); err != nil {
		return err
	}

	// delete all servers
	if err := e.kube.DeleteAllOf(ctx, &v1alpha1.Server{}, client.InNamespace(cr.Namespace), client.MatchingLabels{
		serverSetLabel: cr.Name,
	}); err != nil {
		return err
	}

	if err := e.kube.DeleteAllOf(ctx, &v1alpha1.Volume{}, client.InNamespace(cr.Namespace), client.MatchingLabels{
		serverSetLabel: cr.Name,
	}); err != nil {
		return err
	}

	if err := e.kube.DeleteAllOf(ctx, &v1alpha1.Volumeselector{}, client.InNamespace(cr.Namespace), client.MatchingLabels{
		serverSetLabel: cr.Name,
	}); err != nil {
		return err
	}
	return nil
}

// AreServersUpToDate checks if replicas and template params are equal to server obj params
func AreServersUpToDate(templateParams v1alpha1.ServerSetTemplateSpec, servers []v1alpha1.Server) bool {
	for _, serverObj := range servers {
		if serverObj.Spec.ForProvider.Cores != templateParams.Cores {
			return false
		}
		if serverObj.Spec.ForProvider.RAM != templateParams.RAM {
			return false
		}
		if serverObj.Spec.ForProvider.CPUFamily != templateParams.CPUFamily {
			return false
		}
	}

	return true
}

// AreVolumesUpToDate checks if replicas and template params are equal to volume obj params
func AreVolumesUpToDate(templateParams v1alpha1.BootVolumeTemplate, volumes []v1alpha1.Volume) bool {
	for _, volumeObj := range volumes {
		if volumeObj.Spec.ForProvider.Size != templateParams.Spec.Size {
			return false
		}
		if volumeObj.Spec.ForProvider.Image != templateParams.Spec.Image {
			return false
		}
		if volumeObj.Spec.ForProvider.Type != templateParams.Spec.Type {
			return false
		}
	}

	return true
}

// AreNICsUpToDate - gets nic k8s crs and checks if the correct number of NICs are created
func AreNICsUpToDate(ctx context.Context, kube client.Client, serversetName string, noOfReplicas int) (bool, error) {
	nicList := &v1alpha1.NicList{}
	if err := kube.List(ctx, nicList, client.MatchingLabels{
		serverSetLabel: serversetName,
	}); err != nil {
		return false, err
	}

	if len(nicList.Items) != noOfReplicas {
		return false, nil
	}

	return true, nil
}

// GetServersFromSSet - gets all servers from a server set
func GetServersFromSSet(ctx context.Context, kube client.Client, name string) ([]v1alpha1.Server, error) {
	serverList := &v1alpha1.ServerList{}
	if err := kube.List(ctx, serverList, client.MatchingLabels{
		serverSetLabel: name,
	}); err != nil {
		return nil, err
	}

	return serverList.Items, nil
}

// GetVolumesFromSSet - gets all volumes from a server set
func GetVolumesFromSSet(ctx context.Context, kube client.Client, name string) ([]v1alpha1.Volume, error) {
	volumeList := &v1alpha1.VolumeList{}
	if err := kube.List(ctx, volumeList, client.MatchingLabels{
		serverSetLabel: name,
	}); err != nil {
		return nil, err
	}

	return volumeList.Items, nil
}

// ListResFromSSetWithIndex - lists resources from a server set with a specific index label
func ListResFromSSetWithIndex(ctx context.Context, kube client.Client, serversetName, resType string, index int, list client.ObjectList) error {
	return kube.List(ctx, list, client.MatchingLabels{
		fmt.Sprintf(indexLabel, serversetName, resType): strconv.Itoa(index),
	})
}

// listResFromSSetWithIndexAndVersion - lists resources from a server set with a specific index and version label
func listResFromSSetWithIndexAndVersion(ctx context.Context, kube client.Client, serversetName, resType string, index, version int, list client.ObjectList) error {
	return kube.List(ctx, list, client.MatchingLabels{
		fmt.Sprintf(versionLabel, serversetName, resType): strconv.Itoa(version),
		fmt.Sprintf(indexLabel, serversetName, resType):   strconv.Itoa(index),
	})
}

// getVersionsFromVolumeAndServer checks that there is only one server and volume and returns their version
func getVersionsFromVolumeAndServer(ctx context.Context, kube client.Client, serversetName string, replicaIndex int) (volumeVersion int, serverVersion int, err error) {
	volumeResources := &v1alpha1.VolumeList{}
	err = ListResFromSSetWithIndex(ctx, kube, serversetName, resourceBootVolume, replicaIndex, volumeResources)
	if err != nil {
		return volumeVersion, serverVersion, err
	}
	if len(volumeResources.Items) > 1 {
		return volumeVersion, serverVersion, fmt.Errorf("found too many volumes for index %d ", replicaIndex)
	}
	if len(volumeResources.Items) == 0 {
		return volumeVersion, serverVersion, fmt.Errorf("found no volumes for index %d ", replicaIndex)
	}
	serverResources := &v1alpha1.ServerList{}
	err = ListResFromSSetWithIndex(ctx, kube, serversetName, ResourceServer, replicaIndex, serverResources)
	if err != nil {
		return volumeVersion, serverVersion, err
	}
	if len(serverResources.Items) > 1 {
		return volumeVersion, serverVersion, fmt.Errorf("found too many servers for index %d ", replicaIndex)
	}
	if len(serverResources.Items) == 0 {
		return volumeVersion, serverVersion, fmt.Errorf("found no servers for index %d ", replicaIndex)
	}

	condemnedVolume := volumeResources.Items[0]
	volumeVersion, err = strconv.Atoi(condemnedVolume.Labels[fmt.Sprintf(versionLabel, serversetName, resourceBootVolume)])
	if err != nil {
		return volumeVersion, serverVersion, err
	}

	servers := serverResources.Items
	serverVersion, err = strconv.Atoi(servers[0].Labels[fmt.Sprintf(versionLabel, serversetName, ResourceServer)])
	if err != nil {
		return volumeVersion, serverVersion, err
	}
	return volumeVersion, serverVersion, nil
}

func (e *external) ensureServerAndNicByIndex(ctx context.Context, cr *v1alpha1.ServerSet, replicaIndex, version int) error {
	resSrv := &v1alpha1.ServerList{}
	if err := ListResFromSSetWithIndex(ctx, e.kube, cr.GetName(), ResourceServer, replicaIndex, resSrv); err != nil {
		return err
	}
	if len(resSrv.Items) > 1 {
		return fmt.Errorf("found too many servers for index %d ", replicaIndex)
	}
	if len(resSrv.Items) == 0 {
		res := &v1alpha1.VolumeList{}
		volumeVersion := version
		if err := ListResFromSSetWithIndex(ctx, e.kube, cr.GetName(), resourceBootVolume, replicaIndex, res); err != nil {
			return err
		}
		if len(res.Items) > 0 {
			var err error
			volumeVersion, err = strconv.Atoi(res.Items[0].Labels[fmt.Sprintf(versionLabel, cr.GetName(), resourceBootVolume)])
			if err != nil {
				return err
			}
		}
		if err := e.serverController.Ensure(ctx, cr, replicaIndex, version, volumeVersion); err != nil {
			return err
		}
		if err := e.nicController.EnsureNICs(ctx, cr, replicaIndex, version); err != nil {
			return err
		}
	}
	return nil
}

// ensureBootVolumeByIndex - ensures boot volume created for a specific index. After checking for index, it checks for index and version
func (e *external) ensureBootVolumeByIndex(ctx context.Context, cr *v1alpha1.ServerSet, replicaIndex, version int) error {
	res := &v1alpha1.VolumeList{}
	if err := ListResFromSSetWithIndex(ctx, e.kube, cr.GetName(), resourceBootVolume, replicaIndex, res); err != nil {
		return err
	}
	if len(res.Items) > 1 {
		return fmt.Errorf("found too many volumes for index %d ", replicaIndex)
	}
	if len(res.Items) == 0 {
		if err := e.bootVolumeController.Ensure(ctx, cr, replicaIndex, version); err != nil {
			return err
		}
	}
	return nil
}

// getNameFromIndex - generates name consisting of name, kind and index
func getNameFromIndex(resourceName, resourceType string, idx, version int) string {
	return fmt.Sprintf("%s-%s-%d-%d", resourceName, resourceType, idx, version)
}
