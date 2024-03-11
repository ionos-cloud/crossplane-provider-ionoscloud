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
	newVolumeVersion := volumeVersion + 1
	if err := c.bootVolumeController.Ensure(ctx, cr, replicaIndex, newVolumeVersion); err != nil {
		return err
	}

	server, err := c.serverController.Get(ctx, getNameFromIndex(cr.Name, resourceServer, replicaIndex, serverVersion), cr.Namespace)
	if err != nil {
		return err
	}
	createdVolume, err := c.bootVolumeController.Get(ctx, getNameFromIndex(cr.Name, resourceBootVolume, replicaIndex, newVolumeVersion), cr.Namespace)
	if err != nil {
		return err
	}
	// server.Status.AtProvider.VolumeID = createdVolume.Status.AtProvider.VolumeID
	server.Spec.ForProvider.VolumeCfg.VolumeID = createdVolume.Status.AtProvider.VolumeID
	if err := c.serverController.Update(ctx, server); err != nil {
		return err
	}
	if err = c.bootVolumeController.Delete(ctx, getNameFromIndex(cr.Name, resourceBootVolume, replicaIndex, volumeVersion), cr.Namespace); err != nil {
		return err
	}
	return err
}
