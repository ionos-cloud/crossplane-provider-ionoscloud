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
	if err := c.cleanupCondemned(ctx, cr, replicaIndex, volumeVersion, serverVersion); err != nil {
		return err
	}
	return nil
}

func (c *createBeforeDestroy) createResources(ctx context.Context, cr *v1alpha1.ServerSet, index, volumeVersion, serverVersion int) error {
	if err := c.bootVolumeController.Ensure(ctx, cr, index, volumeVersion); err != nil {
		return err
	}
	if err := c.serverController.Ensure(ctx, cr, index, serverVersion, volumeVersion); err != nil {
		return err
	}
	return c.nicController.EnsureNICs(ctx, cr, index, serverVersion)
}

func (c *createBeforeDestroy) cleanupCondemned(ctx context.Context, cr *v1alpha1.ServerSet, replicaIndex, volumeVersion, serverVersion int) error {
	if err := c.bootVolumeController.Delete(ctx, getNameFromIndex(cr.Spec.ForProvider.BootVolumeTemplate.Metadata.Name, replicaIndex, volumeVersion), cr.Namespace); err != nil {
		return err
	}
	if err := c.serverController.Delete(ctx, getNameFromIndex(cr.Spec.ForProvider.Template.Metadata.Name, replicaIndex, serverVersion), cr.Namespace); err != nil {
		return err
	}
	for nicIndex := range cr.Spec.ForProvider.Template.Spec.NICs {
		if err := c.nicController.Delete(ctx, getNicName(cr.Spec.ForProvider.Template.Spec.NICs[replicaIndex].Name, replicaIndex, nicIndex, serverVersion), cr.Namespace); err != nil {
			return err
		}
	}
	return nil
}
