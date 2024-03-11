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

	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute/server"
)

const (
	errUnexpectedObject = "managed resource is not an Volume resource"
	errTrackPCUsage     = "cannot track ProviderConfig usage"
)

const (
	// indexLabel is the label used to identify the server set by index
	indexLabel = "ionoscloud.com/serverset-%s-index"
	// versionLabel is the label used to identify the server set by version
	versionLabel = "ionoscloud.com/serverset-%s-version"
	// serverSetLabel is the label used to identify the server set resources. All resources created by a server set will have this label
	serverSetLabel = "ionoscloud.com/serverset"
)

// A connector is expected to produce an ExternalClient when its Connect method
// is called.
type connector struct {
	kube                 client.Client
	bootVolumeController kubeBootVolumeControlManager
	nicController        kubeNicControlManager
	serverController     kubeServerControlManager
	usage                resource.Tracker
	log                  logging.Logger
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

	if err := c.usage.Track(ctx, mg); err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	svc, err := clients.ConnectForCRD(ctx, mg, c.kube, c.usage)
	if err != nil {
		return nil, err
	}

	return &external{
		kube:                 c.kube,
		service:              &server.APIClient{IonosServices: svc},
		log:                  c.log,
		bootVolumeController: c.bootVolumeController,
		nicController:        c.nicController,
		serverController:     c.serverController,
	}, err
}

// external observes, then either creates, updates, or deletes an
// externalServerSet resource to ensure it reflects the managed resource's desired state.
type external struct {
	kube client.Client
	// A 'client' used to connect to the externalServer resource API. In practice this
	// would be something like an IONOS Cloud SDK client.
	bootVolumeController kubeBootVolumeControlManager
	nicController        kubeNicControlManager
	serverController     kubeServerControlManager
	service              server.Client
	log                  logging.Logger
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.ServerSet)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errUnexpectedObject)
	}

	if meta.GetExternalName(cr) == "" {
		return managed.ExternalObservation{}, nil
	}

	servers, err := e.getServersFromServerSet(ctx, cr.Name)
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	cr.Status.AtProvider.Replicas = len(servers)
	if len(servers) < cr.Spec.ForProvider.Replicas {
		return managed.ExternalObservation{
			// we need to re-create servers. go to create
			ResourceExists:    false,
			ResourceUpToDate:  false,
			ConnectionDetails: managed.ConnectionDetails{},
		}, nil
	}

	areServersUpToDate := areServersUpToDate(cr.Spec.ForProvider.Template.Spec, servers)

	volumes, err := e.getVolumesFromServerSet(ctx, cr.Name)
	if err != nil {
		return managed.ExternalObservation{}, err
	}
	areVolumesUpToDate := areVolumesUpToDate(cr.Spec.ForProvider, volumes)
	// only update
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
	if areNicsUpToDate, err = e.areNicsUpToDate(ctx, cr); err != nil {
		return managed.ExternalObservation{}, err
	}
	if !areNicsUpToDate {
		return managed.ExternalObservation{
			ResourceExists:    false,
			ResourceUpToDate:  false,
			ConnectionDetails: managed.ConnectionDetails{},
		}, nil
	}

	cr.Status.SetConditions(xpv1.Available())

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
		e.log.Info("Creating a new Server", "index", i)
		err := e.ensureBootVolumeByIndex(ctx, cr, i, version)
		if err != nil {
			return managed.ExternalCreation{}, err
		}

		err = e.ensureServerAndNicByIndex(ctx, cr, i, version)
		if err != nil {
			return managed.ExternalCreation{}, err
		}

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
	servers, err := e.getServersFromServerSet(ctx, cr.Name)
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
				fmt.Printf("error updating server %v", err)
				return err
			}
		}
	}
	return nil
}

// reconcileVolumesFromTemplate updates bootvolume, or deletes and re-creates server, volume and nic if something
// immutable changes in a bootvolume
func (e *external) reconcileVolumesFromTemplate(ctx context.Context, cr *v1alpha1.ServerSet) error {
	volumes, err := e.getVolumesFromServerSet(ctx, cr.Name)
	if err != nil {
		return err
	}

	for idx := range volumes {
		update := false
		deleteAndCreate := false
		if volumes[idx].Spec.ForProvider.Size != cr.Spec.ForProvider.BootVolumeTemplate.Spec.Size {
			update = true
			volumes[idx].Spec.ForProvider.Size = cr.Spec.ForProvider.BootVolumeTemplate.Spec.Size
		}
		if volumes[idx].Spec.ForProvider.Type != cr.Spec.ForProvider.BootVolumeTemplate.Spec.Type {
			deleteAndCreate = true
			volumes[idx].Spec.ForProvider.Type = cr.Spec.ForProvider.BootVolumeTemplate.Spec.Type
		}

		if volumes[idx].Spec.ForProvider.Image != cr.Spec.ForProvider.BootVolumeTemplate.Spec.Image {
			deleteAndCreate = true
			volumes[idx].Spec.ForProvider.Image = cr.Spec.ForProvider.BootVolumeTemplate.Spec.Image
		}

		if deleteAndCreate {
			volumeVersion, serverVersion, err := getVersionsFromVolumeAndServer(ctx, e.kube, idx)
			if err != nil {
				return err
			}
			// creates bootvolume, server, nic
			if err = e.createResources(ctx, cr, idx, volumeVersion+1, serverVersion+1); err != nil {
				return err
			}

			// cleanup - bootvolume, server, nic
			if err = e.cleanupCondemned(ctx, cr, idx, volumeVersion, serverVersion); err != nil {
				return err
			}

		} else if update {
			if err := e.kube.Update(ctx, &volumes[idx]); err != nil {
				fmt.Printf("error updating server %v", err)
				return err
			}
		}
	}

	return nil
}

func (e *external) createResources(ctx context.Context, cr *v1alpha1.ServerSet, index, volumeVersion, serverVersion int) error {
	if err := e.bootVolumeController.EnsureBootVolume(ctx, cr, index, volumeVersion); err != nil {
		return err
	}

	if err := e.serverController.EnsureServer(ctx, cr, index, serverVersion); err != nil {
		return err
	}

	return e.nicController.EnsureNICs(ctx, cr, index, serverVersion)
}

func (e *external) cleanupCondemned(ctx context.Context, cr *v1alpha1.ServerSet, index, volumeVersion, serverVersion int) error {
	err := e.bootVolumeController.Delete(ctx, getNameFromIndex(cr.Name, resourceBootVolume, index, volumeVersion), cr.Namespace)
	if err != nil {
		return err
	}
	err = e.serverController.Delete(ctx, getNameFromIndex(cr.Name, resourceServer, index, serverVersion), cr.Namespace)
	if err != nil {
		return err
	}
	return e.nicController.Delete(ctx, getNameFromIndex(cr.Name, resourceNIC, index, serverVersion), cr.Namespace)
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*v1alpha1.ServerSet)
	if !ok {
		return errors.New(errUnexpectedObject)
	}

	cr.SetConditions(xpv1.Deleting())

	fmt.Printf("Deleting: %+v", cr)

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

	return nil
}

// areServersUpToDate checks if replicas and template params are equal to server obj params
func areServersUpToDate(templateParams v1alpha1.ServerSetTemplateSpec, servers []v1alpha1.Server) bool {

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

// areVolumesUpToDate
func areVolumesUpToDate(templateParams v1alpha1.ServerSetParameters, volumes []v1alpha1.Volume) bool {

	for _, volumeObj := range volumes {
		if volumeObj.Spec.ForProvider.Size != templateParams.BootVolumeTemplate.Spec.Size {
			return false
		}
		if volumeObj.Spec.ForProvider.Image != templateParams.BootVolumeTemplate.Spec.Image {
			return false
		}
		if volumeObj.Spec.ForProvider.Type != templateParams.BootVolumeTemplate.Spec.Type {
			return false
		}
	}

	return true
}

// areNicsUpToDate gets nic k8s crs and checks if the correct number of NICs are created
func (e *external) areNicsUpToDate(ctx context.Context, cr *v1alpha1.ServerSet) (bool, error) {
	e.log.Info("Ensuring NIC")

	nicList := &v1alpha1.NicList{}
	if err := e.kube.List(ctx, nicList, client.MatchingLabels{
		serverSetLabel: cr.Name,
	}); err != nil {
		return false, err
	}

	if len(nicList.Items) != cr.Spec.ForProvider.Replicas {
		return false, nil
	}

	return true, nil
}

func (e *external) getServersFromServerSet(ctx context.Context, name string) ([]v1alpha1.Server, error) {
	serverList := &v1alpha1.ServerList{}
	if err := e.kube.List(ctx, serverList, client.MatchingLabels{
		serverSetLabel: name,
	}); err != nil {
		return nil, err
	}

	return serverList.Items, nil
}

func (e *external) getVolumesFromServerSet(ctx context.Context, name string) ([]v1alpha1.Volume, error) {
	volumeList := &v1alpha1.VolumeList{}
	if err := e.kube.List(ctx, volumeList, client.MatchingLabels{
		serverSetLabel: name,
	}); err != nil {
		return nil, err
	}

	return volumeList.Items, nil
}

// ListResFromSSetWithIndex - lists resources from a server set with a specific index label
func ListResFromSSetWithIndex(ctx context.Context, kube client.Client, resType string, index int, list client.ObjectList) error {
	return kube.List(ctx, list, client.MatchingLabels{

		fmt.Sprintf(indexLabel, resType): strconv.Itoa(index),
	})
}

// ListResFromSSetWithIndexAndVersion - lists resources from a server set with a specific index and version label
func ListResFromSSetWithIndexAndVersion(ctx context.Context, kube client.Client, resType string, index, version int, list client.ObjectList) error {
	return kube.List(ctx, list, client.MatchingLabels{
		fmt.Sprintf(versionLabel, resType): strconv.Itoa(version),
		fmt.Sprintf(indexLabel, resType):   strconv.Itoa(index),
	})
}

// getVersionsFromVolumeAndServer checks that there is only one server and volume and returns their version
func getVersionsFromVolumeAndServer(ctx context.Context, kube client.Client, replicaIndex int) (volumeVersion int, serverVersion int, err error) {
	volumeResources := &v1alpha1.VolumeList{}
	err = ListResFromSSetWithIndex(ctx, kube, resourceBootVolume, replicaIndex, volumeResources)
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
	err = ListResFromSSetWithIndex(ctx, kube, resourceServer, replicaIndex, serverResources)
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
	volumeVersion, err = strconv.Atoi(condemnedVolume.Labels[fmt.Sprintf(versionLabel, resourceBootVolume)])
	if err != nil {
		return volumeVersion, serverVersion, err
	}

	servers := serverResources.Items
	serverVersion, err = strconv.Atoi(servers[0].Labels[fmt.Sprintf(versionLabel, resourceServer)])
	if err != nil {
		return volumeVersion, serverVersion, err
	}
	return volumeVersion, serverVersion, nil
}

func (e *external) ensureServerAndNicByIndex(ctx context.Context, cr *v1alpha1.ServerSet, replicaIndex, version int) error {
	resSrv := &v1alpha1.ServerList{}
	if err := ListResFromSSetWithIndex(ctx, e.kube, resourceServer, replicaIndex, resSrv); err != nil {
		return err
	}
	if len(resSrv.Items) > 1 {
		return fmt.Errorf("found too many servers for index %d ", replicaIndex)
	}
	if len(resSrv.Items) == 0 {
		if err := e.serverController.EnsureServer(ctx, cr, replicaIndex, version); err != nil {
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
	if err := ListResFromSSetWithIndex(ctx, e.kube, resourceBootVolume, replicaIndex, res); err != nil {
		return err
	}
	if len(res.Items) > 1 {
		return fmt.Errorf("found too many volumes for index %d ", replicaIndex)
	}
	if len(res.Items) == 0 {
		if err := e.bootVolumeController.EnsureBootVolume(ctx, cr, replicaIndex, version); err != nil {
			return err
		}
	}
	return nil
}
