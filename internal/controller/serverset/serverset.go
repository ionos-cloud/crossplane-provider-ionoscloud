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
	ionoscloud "github.com/ionos-cloud/sdk-go/v6"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute/server"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/controller/kube"
)

const (
	errUnexpectedObject = "managed resource is not an Volume resource"

	errTrackPCUsage = "cannot track ProviderConfig usage"
)

// A connector is expected to produce an ExternalClient when its Connect method
// is called.
type connector struct {
	kubeWrapper kube.Wrapper
	usage       resource.Tracker
	log         logging.Logger
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

	svc, err := clients.ConnectForCRD(ctx, mg, c.kubeWrapper.Kube, c.usage)
	if err != nil {
		return nil, err
	}

	return &external{
		kubeWrapper: c.kubeWrapper,
		service:     &server.APIClient{IonosServices: svc},
		log:         c.log,
	}, err
}

// external observes, then either creates, updates, or deletes an
// externalServerSet resource to ensure it reflects the managed resource's desired state.
type external struct {
	kubeWrapper kube.Wrapper
	// A 'client' used to connect to the externalServer resource API. In practice this
	// would be something like an IONOS Cloud SDK client.

	service server.Client
	log     logging.Logger
}

func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.ServerSet)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errUnexpectedObject)
	}

	if meta.GetExternalName(cr) == "" {
		return managed.ExternalObservation{}, nil
	}

	servers, err := c.getServersFromServerSet(ctx, cr.Name)
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	cr.Status.AtProvider.Replicas = len(servers)
	// we need to re-create servers. go to create
	if len(servers) < cr.Spec.ForProvider.Replicas {
		return managed.ExternalObservation{
			ResourceExists:    false,
			ResourceUpToDate:  false,
			ConnectionDetails: managed.ConnectionDetails{},
		}, nil
	}

	areServersUpToDate := areServersUpToDate(cr.Spec.ForProvider.Template.Spec, servers)

	volumes, err := c.getVolumesFromServerSet(ctx, cr.Name)
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
	if areNicsUpToDate, err = c.areNicsUpToDate(ctx, cr); err != nil {
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

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.ServerSet)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errUnexpectedObject)
	}

	cr.Status.SetConditions(xpv1.Creating())

	// for n times of cr.Spec.Replicas, create a server
	// for each server, create a volume
	c.log.Info("Creating a new ServerSet", "replicas", cr.Spec.ForProvider.Replicas)
	version := 0
	for i := 0; i < cr.Spec.ForProvider.Replicas; i++ {
		c.log.Info("Creating a new Server", "index", i)
		if err := c.ensureBootVolume(ctx, cr, kube.GetNameFromIndex(cr.Name, "bootvolume", i, version), i, version); err != nil {
			return managed.ExternalCreation{}, err
		}

		if err := c.ensureServer(ctx, cr, i, version); err != nil {
			return managed.ExternalCreation{}, err
		}

		if err := c.ensureVolumeClaim(); err != nil {
			return managed.ExternalCreation{}, err
		}

		if err := c.ensureNICs(ctx, cr, i, version); err != nil {
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

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.ServerSet)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errUnexpectedObject)
	}
	// how do we know if we want to update servers or nic params?
	err := c.updateServersFromTemplate(ctx, cr)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}
	if err := c.reconcileVolumesFromTemplate(ctx, cr); err != nil {
		return managed.ExternalUpdate{}, err

	}

	return managed.ExternalUpdate{
		// Optionally return any details that may be required to connect to the
		// externalServerSet resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) updateServersFromTemplate(ctx context.Context, cr *v1alpha1.ServerSet) error {
	servers, err := c.getServersFromServerSet(ctx, cr.Name)
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
			if err := c.kubeWrapper.Kube.Update(ctx, &servers[idx]); err != nil {
				fmt.Printf("error updating server %v", err)
				return err
			}
		}
	}
	return nil
}

// reconcileVolumesFromTemplate updates bootvolume, or deletes and re-creates server, volume and nic if something
// immutable changes in a bootvolume
func (c *external) reconcileVolumesFromTemplate(ctx context.Context, cr *v1alpha1.ServerSet) error {
	volumes, err := c.getVolumesFromServerSet(ctx, cr.Name)
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
			res := &v1alpha1.VolumeList{}
			err := kube.ListResourceFromLabelIndex(ctx, c.kubeWrapper.Kube, "bootvolume", idx, res)
			if err != nil {
				return err
			}
			if len(res.Items) > 1 {
				return fmt.Errorf("found too many volumes for index %d ", idx)
			}
			if len(res.Items) > 0 {
				condemnedVolume := res.Items[0]
				volumeVersion, _ := strconv.Atoi(condemnedVolume.Labels[fmt.Sprintf(kube.ServersetVersionLabel, "bootvolume")])
				newVolumeVersion := volumeVersion + 1

				if _, err = c.createBootVolume(ctx, cr, kube.GetNameFromIndex(cr.Name, "bootvolume", idx, newVolumeVersion), idx, newVolumeVersion); err != nil {
					return err
				}
				res := &v1alpha1.ServerList{}
				err := kube.ListResourceFromLabelIndex(ctx, c.kubeWrapper.Kube, "server", idx, res)
				if err != nil {
					return err
				}
				servers := res.Items
				if len(servers) > 0 {
					serverVersion, _ := strconv.Atoi(servers[0].Labels[fmt.Sprintf(kube.ServersetVersionLabel, "server")])
					newServerVersion := serverVersion + 1
					err = c.createServer(ctx, cr, idx, newServerVersion, newVolumeVersion)
					if err != nil {
						return err
					}
					createdServer, err := c.getServer(ctx, kube.GetNameFromIndex(cr.Name, "server", idx, newServerVersion), cr.Namespace)
					if err != nil {
						return err
					}

					for nicx := range cr.Spec.ForProvider.Template.Spec.NICs {
						if err := c.kubeWrapper.CreateNic(ctx, cr, createdServer.Status.AtProvider.ServerID,
							cr.Spec.ForProvider.Template.Spec.NICs[nicx].Reference, idx, newServerVersion); err != nil {
							return err
						}
					}

					// wait for server to become ready again after re-attaching volume
					err = kube.WaitForKubeResource(ctx, kube.ResourceReadyTimeout, c.kubeWrapper.IsServerAvailable, kube.GetNameFromIndex(cr.Name, "server", idx, volumeVersion), cr.Namespace)
					if err != nil {
						return err
					}

					if err := c.kubeWrapper.Kube.Delete(ctx, &condemnedVolume); err != nil {
						fmt.Printf("error deleting volume %v", err)
						return err
					}
					err = kube.WaitForKubeResource(ctx, kube.ResourceReadyTimeout, c.kubeWrapper.IsVolumeDeleted, condemnedVolume.Name, cr.Namespace)
					if err != nil {
						return err
					}

					condemnedServer, err := c.getServer(ctx, kube.GetNameFromIndex(cr.Name, "server", idx, serverVersion), cr.Namespace)
					if err != nil {
						return err
					}
					if err := c.kubeWrapper.Kube.Delete(ctx, condemnedServer); err != nil {
						fmt.Printf("error deleting server %v", err)
						return err
					}

					condemnedNic, err := c.kubeWrapper.GetNic(ctx, kube.GetNameFromIndex(cr.Name, "nic", idx, serverVersion), cr.Namespace)
					if err != nil {
						return err
					}
					if err := c.kubeWrapper.Kube.Delete(ctx, condemnedNic); err != nil {
						fmt.Printf("error deleting nic %v", err)
						return err
					}

					// todo change to wait for server deletion
					// err = kubeWrapper.WaitForKubeResource(ctx, resourceReadyTimeout, kubeWrapper.IsVolumeDeleted, c.kubeWrapper.Kube, volumes[idx].Name, cr.Namespace)
					// if err != nil {
					// 	return err
					// }
				}
			}
		} else if update {
			if err := c.kubeWrapper.Kube.Update(ctx, &volumes[idx]); err != nil {
				fmt.Printf("error updating server %v", err)
				return err
			}
		}
	}

	return nil
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*v1alpha1.ServerSet)
	if !ok {
		return errors.New(errUnexpectedObject)
	}

	cr.SetConditions(xpv1.Deleting())

	fmt.Printf("Deleting: %+v", cr)

	if err := c.kubeWrapper.Kube.DeleteAllOf(ctx, &v1alpha1.Nic{}, client.InNamespace(cr.Namespace), client.MatchingLabels{
		kube.ServerSetLabel: cr.Name,
	}); err != nil {
		return err
	}

	// delete all servers
	if err := c.kubeWrapper.Kube.DeleteAllOf(ctx, &v1alpha1.Server{}, client.InNamespace(cr.Namespace), client.MatchingLabels{
		kube.ServerSetLabel: cr.Name,
	}); err != nil {
		return err
	}

	if err := c.kubeWrapper.Kube.DeleteAllOf(ctx, &v1alpha1.Volume{}, client.InNamespace(cr.Namespace), client.MatchingLabels{
		kube.ServerSetLabel: cr.Name,
	}); err != nil {
		return err
	}

	return nil
}

func (c *external) ensureBootVolume(ctx context.Context, cr *v1alpha1.ServerSet, name string, replicaIndex, version int) error {
	c.log.Info("Ensuring BootVolume", "name", name)
	// ns := cr.Namespace
	resourceType := "bootvolume"
	res := &v1alpha1.VolumeList{}
	err := kube.ListResourceFromLabelIndex(ctx, c.kubeWrapper.Kube, resourceType, replicaIndex, res)
	if err != nil {
		return err
	}
	volumes := res.Items
	if len(volumes) > 0 {
		version, _ = strconv.Atoi(volumes[0].Labels[fmt.Sprintf(kube.ServersetVersionLabel, "bootvolume")])
	} else {
		_, err := c.createBootVolume(ctx, cr, name, replicaIndex, version)
		return err
	}
	// c.log.Info(fmt.Sprintf("Volume State: %s", volume.Status.AtProvider.State))

	return nil
}

func (c *external) ensureVolumeClaim() error {
	c.log.Info("Ensuring Volume")

	return nil
}

func (c *external) ensureServer(ctx context.Context, cr *v1alpha1.ServerSet, replicaIndex, version int) error {
	c.log.Info("Ensuring Server")

	resourceType := "server"
	res := &v1alpha1.ServerList{}
	err := kube.ListResourceFromLabelIndex(ctx, c.kubeWrapper.Kube, resourceType, replicaIndex, res)
	if err != nil {
		return err
	}
	servers := res.Items
	if len(servers) > 0 {
		c.log.Info("Server already exists", "name", servers[0].Name)
		// version, _ = strconv.Atoi(servers[0].Labels[serversetVersionLabel])
	} else {
		// server and volume have the same version(?) for now
		return c.createServer(ctx, cr, replicaIndex, version, version)
	}
	// fmt.Println("Server State: ", obj.Status.AtProvider.State)

	return nil
}

func (c *external) getServer(ctx context.Context, name, ns string) (*v1alpha1.Server, error) {
	obj := &v1alpha1.Server{}
	if err := c.kubeWrapper.Kube.Get(ctx, types.NamespacedName{
		Namespace: ns,
		Name:      name,
	}, obj); err != nil {
		return nil, err
	}

	return obj, nil
}

func (c *external) createServer(ctx context.Context, cr *v1alpha1.ServerSet, replicaIndex, version, volumeVersion int) error {
	c.log.Info("Creating Server")
	serverType := "server"
	serverObj := kube.FromServerSetToServer(cr, replicaIndex, version, volumeVersion)

	serverObj.SetProviderConfigReference(cr.Spec.ProviderConfigReference)
	if err := c.kubeWrapper.Kube.Create(ctx, &serverObj); err != nil {
		fmt.Println("error creating server")
		fmt.Println(err.Error())
		return err
	}
	if err := kube.WaitForKubeResource(ctx, kube.ResourceReadyTimeout, c.kubeWrapper.IsServerAvailable, kube.GetNameFromIndex(cr.Name, serverType, replicaIndex, version), cr.Namespace); err != nil {
		return fmt.Errorf("while waiting for server to be populated %w ", err)
	}
	return nil
}

// createBootVolume creates a volume CR and waits until in reaches AVAILABLE state
func (c *external) createBootVolume(ctx context.Context, cr *v1alpha1.ServerSet, name string, replicaIndex, version int) (v1alpha1.Volume, error) {
	c.log.Info("Creating Volume")
	var volumeObj = kube.FromServerSetToVolume(cr, name, replicaIndex, version)
	volumeObj.SetProviderConfigReference(cr.Spec.ProviderConfigReference)
	if err := c.kubeWrapper.Kube.Create(ctx, &volumeObj); err != nil {
		return v1alpha1.Volume{}, err
	}
	if err := kube.WaitForKubeResource(ctx, kube.ResourceReadyTimeout, c.kubeWrapper.IsVolumeAvailable, name, cr.Namespace); err != nil {
		return v1alpha1.Volume{}, err
	}
	// get the volume again before returning to have the id populated
	kubeVolume, err := c.kubeWrapper.GetVolume(ctx, name, cr.Namespace)
	if err != nil {
		return v1alpha1.Volume{}, err
	}
	return *kubeVolume, nil
}

func (c *external) ensureNICs(ctx context.Context, cr *v1alpha1.ServerSet, idx, version int) error {
	c.log.Info("Ensuring NIC")
	res := &v1alpha1.ServerList{}
	err := kube.ListResourceFromLabelIndex(ctx, c.kubeWrapper.Kube, "server", idx, res)
	if err != nil {
		return err
	}
	servers := res.Items
	// check if the NIC is attached to the server
	fmt.Printf("we have to check if the NIC is attached to the server %s ", cr.Name)
	if len(servers) > 0 {
		for nicx := range cr.Spec.ForProvider.Template.Spec.NICs {
			if err := c.ensureNIC(ctx, cr, servers[0].Status.AtProvider.ServerID, cr.Spec.ForProvider.Template.Spec.NICs[nicx].Reference, idx, version); err != nil {
				return err
			}
		}
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
func (c *external) areNicsUpToDate(ctx context.Context, cr *v1alpha1.ServerSet) (bool, error) {
	c.log.Info("Ensuring NIC")

	nicList := &v1alpha1.NicList{}
	if err := c.kubeWrapper.Kube.List(ctx, nicList, client.MatchingLabels{
		kube.ServerSetLabel: cr.Name,
	}); err != nil {
		return false, err
	}

	if len(nicList.Items) != cr.Spec.ForProvider.Replicas {
		return false, nil
	}

	return true, nil
}

func (c *external) ensureNIC(ctx context.Context, cr *v1alpha1.ServerSet, serverID, lanName string, replicaIndex, version int) error {
	// get the network
	resourceType := "nic"

	res := &v1alpha1.NicList{}
	err := kube.ListResourceFromLabelIndex(ctx, c.kubeWrapper.Kube, resourceType, replicaIndex, res)
	if err != nil {
		return err
	}
	if len(res.Items) == 0 {
		return c.kubeWrapper.CreateNic(ctx, cr, serverID, lanName, replicaIndex, version)

		// network := v1alpha1.Lan{}
		// if err := c.kubeWrapper.Kube.Get(ctx, types.NamespacedName{
		// 	Namespace: cr.GetNamespace(),
		// 	Name:      lanName,
		// }, &network); err != nil {
		// 	return err
		// }
		// lanID := network.Status.AtProvider.LanID
		// nicName := kubeWrapper.GetNameFromIndex(cr.Name, resourceType, replicaIndex, version)
		// // no NIC found, create one
		// c.log.Info("Creating NIC", "name", nicName)
		// createNic := fromServerSetToNic(cr, nicName, serverID, lanID, replicaIndex, version)
		// createNic.SetProviderConfigReference(cr.Spec.ProviderConfigReference)
		// return c.kubeWrapper.Kube.Create(ctx, &createNic)
	} else {
		observedNic := res.Items[0]
		// NIC found, check if it's attached to the server
		if !strings.EqualFold(observedNic.Status.AtProvider.State, ionoscloud.Available) {
			return fmt.Errorf("observedNic %s got state %s but expected %s", observedNic.GetName(), observedNic.Status.AtProvider.State, ionoscloud.Available)
		}
	}

	return nil
}

func (c *external) getServersFromServerSet(ctx context.Context, name string) ([]v1alpha1.Server, error) {
	serverList := &v1alpha1.ServerList{}
	if err := c.kubeWrapper.Kube.List(ctx, serverList, client.MatchingLabels{
		kube.ServerSetLabel: name,
	}); err != nil {
		return nil, err
	}

	return serverList.Items, nil
}

func (c *external) getVolumesFromServerSet(ctx context.Context, name string) ([]v1alpha1.Volume, error) {
	volumeList := &v1alpha1.VolumeList{}
	if err := c.kubeWrapper.Kube.List(ctx, volumeList, client.MatchingLabels{
		kube.ServerSetLabel: name,
	}); err != nil {
		return nil, err
	}

	return volumeList.Items, nil
}
