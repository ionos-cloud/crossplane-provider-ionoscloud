package serverset

import (
	"context"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type updater interface {
	update(ctx context.Context, cr *v1alpha1.ServerSet, replicaIndex, volumeVersion, serverVersion int) error
}

type createBeforeDestroy struct {
	bootVolumeController   kubeBootVolumeControlManager
	serverController       kubeServerControlManager
	nicController          kubeNicControlManager
	firewallRuleController kubeFirewallRuleControlManager
	kube client.Client
}

func newCreateBeforeDestroy(
	bootVolumeController kubeBootVolumeControlManager, serverController kubeServerControlManager,
	nicController kubeNicControlManager, firewallRuleController kubeFirewallRuleControlManager,
) *createBeforeDestroy {
	return &createBeforeDestroy{
		bootVolumeController:   bootVolumeController,
		serverController:       serverController,
		nicController:          nicController,
		firewallRuleController: firewallRuleController,
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
	if err := c.serverController.Ensure(ctx, cr, index, serverVersion, volumeVersion); err != nil {
		return err
	}
	if err := c.bootVolumeController.Ensure(ctx, cr, index, volumeVersion); err != nil {
		return err
	}

	bootVolume, err := c.bootVolumeController.Get(ctx, getNameFrom(cr.Spec.ForProvider.BootVolumeTemplate.Metadata.Name, index, volumeVersion), cr.Namespace)
	if err != nil {
		return err
	}

	server, err := c.serverController.Get(ctx, getNameFrom(cr.Spec.ForProvider.Template.Metadata.Name, index, serverVersion), cr.Namespace)
	if err != nil {
		return err
	}
	server.Spec.ForProvider.VolumeCfg.VolumeID = bootVolume.Status.AtProvider.VolumeID
	if err := c.serverController.Update(ctx, server); err != nil {
		return err
	}

	serverID := server.Status.AtProvider.ServerID

	if err := c.nicController.EnsureNICs(ctx, cr, index, serverVersion, serverID); err != nil {
		return err
	}

	return c.firewallRuleController.EnsureFirewallRules(ctx, cr, index, serverVersion, serverID)
}

func (c *createBeforeDestroy) cleanupCondemned(ctx context.Context, cr *v1alpha1.ServerSet, replicaIndex, volumeVersion, serverVersion int) error {
	if err := c.bootVolumeController.Delete(ctx, getNameFrom(cr.Spec.ForProvider.BootVolumeTemplate.Metadata.Name, replicaIndex, volumeVersion), cr.Namespace); err != nil {
		return err
	}
	if err := c.serverController.Delete(ctx, getNameFrom(cr.Spec.ForProvider.Template.Metadata.Name, replicaIndex, serverVersion), cr.Namespace); err != nil {
		return err
	}
	for nicIndex := range cr.Spec.ForProvider.Template.Spec.NICs {
		for fwRuleIdx := range cr.Spec.ForProvider.Template.Spec.NICs[nicIndex].FirewallRules {
			if err := c.firewallRuleController.Delete(
				ctx,
				getFirewallRuleName(
					cr.Spec.ForProvider.Template.Spec.NICs[nicIndex].FirewallRules[fwRuleIdx].Name,
					replicaIndex, nicIndex, fwRuleIdx, serverVersion,
				),
				cr.Namespace,
			); err != nil {
				return err
			}
		}

		if err := c.nicController.Delete(ctx, getNicName(cr.Spec.ForProvider.Template.Spec.NICs[nicIndex].Name, replicaIndex, nicIndex, serverVersion), cr.Namespace); err != nil {
			return err
		}
	}
	return nil
}
