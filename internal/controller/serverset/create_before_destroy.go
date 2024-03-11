package serverset

import (
	"context"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
)

type updater interface {
	update(ctx context.Context, cr *v1alpha1.ServerSet, replicaIndex, volumeVersion, serverVersion int) error
}

type createBeforeDestroy struct {
	bootVolumeController kubeBootVolumeControlManager
	serverController     kubeServerControlManager
	nicController        kubeNicControlManager
}

func newCreateBeforeDestroy(bootVolumeController kubeBootVolumeControlManager, serverController kubeServerControlManager, nicController kubeNicControlManager) *createBeforeDestroy {
	return &createBeforeDestroy{
		bootVolumeController: bootVolumeController,
		serverController:     serverController,
		nicController:        nicController,
	}
}

func (c *createBeforeDestroy) update(ctx context.Context, cr *v1alpha1.ServerSet, replicaIndex, volumeVersion, serverVersion int) error {
	// creates bootvolume, server, nic
	if err := c.createResources(ctx, cr, replicaIndex, volumeVersion+1, serverVersion+1); err != nil {
		return err
	}
	// cleanup - bootvolume, server, nic
	return c.cleanupCondemned(ctx, cr, replicaIndex, volumeVersion, serverVersion)
}

func (c *createBeforeDestroy) createResources(ctx context.Context, cr *v1alpha1.ServerSet, index, volumeVersion, serverVersion int) error {
	if err := c.bootVolumeController.EnsureBootVolume(ctx, cr, index, volumeVersion); err != nil {
		return err
	}
	if err := c.serverController.EnsureServer(ctx, cr, index, serverVersion, volumeVersion); err != nil {
		return err
	}
	return c.nicController.EnsureNICs(ctx, cr, index, serverVersion)
}

func (c *createBeforeDestroy) cleanupCondemned(ctx context.Context, cr *v1alpha1.ServerSet, index, volumeVersion, serverVersion int) error {
	err := c.bootVolumeController.Delete(ctx, getNameFromIndex(cr.Name, resourceBootVolume, index, volumeVersion), cr.Namespace)
	if err != nil {
		return err
	}
	err = c.serverController.Delete(ctx, getNameFromIndex(cr.Name, resourceServer, index, serverVersion), cr.Namespace)
	if err != nil {
		return err
	}
	return c.nicController.Delete(ctx, getNameFromIndex(cr.Name, resourceNIC, index, serverVersion), cr.Namespace)
}
