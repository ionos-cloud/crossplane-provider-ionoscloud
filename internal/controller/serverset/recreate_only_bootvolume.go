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

	// Ensure() is already idempotent by existence: it only creates newVolumeVersion when no
	// boot-volume CR with that name/version exists yet (see kubeBootVolumeController.Ensure in
	// bootvolume_controller.go), so it is always safe to call again here even when a previous
	// update() run already created it - e.g. a run interrupted after creating the replacement
	// volume but before attaching it to the server and deleting the old one. getVolumeVersion /
	// oldestOfAdjacentVolumePair resume such an interrupted run by passing back in the OLD/
	// still-live volumeVersion, so newVolumeVersion here is exactly the volume that run already
	// created.
	//
	// This used to be guarded by first Get()-ing the OLD volume (at volumeVersion) and returning
	// early - skipping the server attach and old-volume delete below entirely - whenever that OLD
	// volume already matched cr.Spec's current target. That check answers the wrong question: it
	// tests whether the OLD volume happens to match the current target, not whether the NEW
	// volume has already been created. During an interrupted-swap resumption, if cr.Spec is
	// reverted back to the OLD volume's image/type while the swap is stuck, the OLD volume then
	// matches and the guard fires, permanently no-op'ing every subsequent reconcile: the orphaned
	// NEW volume is never attached to the server nor deleted, and no error is ever raised. Do not
	// reintroduce a pre-check like that; rely on Ensure's own idempotency instead.
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
