package serverset

import (
	"context"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
)

type createBeforeDestroyOnlyBootVolume struct {
	bootVolumeController kubeBootVolumeControlManager
	serverController     kubeServerControlManager
}

func newCreateBeforeDestroyOnlyBootVolume(bootVolumeController kubeBootVolumeControlManager, serverController kubeServerControlManager) *createBeforeDestroyOnlyBootVolume {
	return &createBeforeDestroyOnlyBootVolume{
		bootVolumeController: bootVolumeController,
		serverController:     serverController,
	}
}

func (c *createBeforeDestroyOnlyBootVolume) update(ctx context.Context, cr *v1alpha1.ServerSet, replicaIndex, volumeVersion, serverVersion int) error {
	// todo we need the same IP in case of an update that deletes the bootvolume
	newVolumeVersion := volumeVersion + 1

	existingVolume, err := c.bootVolumeController.Get(ctx, getNameFrom(cr.Spec.ForProvider.BootVolumeTemplate.Metadata.Name, replicaIndex, volumeVersion), cr.Namespace)
	if err != nil {
		return err
	}
	// no need to recreate bootvolume, we created one already. we need to do this because we update the status between creations of replicas, so a new observe runs so it tries
	// to create a volume while the other one is creating
	// if there are other fields that force re-creation, they should be added here
	if existingVolume != nil && existingVolume.Spec.ForProvider.Type == cr.Spec.ForProvider.BootVolumeTemplate.Spec.Type &&
		existingVolume.Spec.ForProvider.Image == cr.Spec.ForProvider.BootVolumeTemplate.Spec.Image &&
		existingVolume.Spec.ForProvider.SetHotPlugsFromImage == cr.Spec.ForProvider.BootVolumeTemplate.Spec.SetHotPlugsFromImage {
		return nil
	}
	if err := c.bootVolumeController.Ensure(ctx, cr, replicaIndex, newVolumeVersion); err != nil {
		return err
	}
	server, err := c.serverController.Get(ctx, getNameFrom(cr.Spec.ForProvider.Template.Metadata.Name, replicaIndex, serverVersion), cr.Namespace)
	if err != nil {
		return err
	}
	createdVolume, err := c.bootVolumeController.Get(ctx, getNameFrom(cr.Spec.ForProvider.BootVolumeTemplate.Metadata.Name, replicaIndex, newVolumeVersion), cr.Namespace)
	if err != nil {
		return err
	}
	server.Spec.ForProvider.VolumeCfg.VolumeID = createdVolume.Status.AtProvider.VolumeID
	if err := c.serverController.Update(ctx, server); err != nil {
		return err
	}
	if err = c.bootVolumeController.Delete(ctx, getNameFrom(cr.Spec.ForProvider.BootVolumeTemplate.Metadata.Name, replicaIndex, volumeVersion), cr.Namespace); err != nil {
		return err
	}
	return err
}
