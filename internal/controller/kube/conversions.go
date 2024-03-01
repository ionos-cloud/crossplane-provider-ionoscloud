package kube

import (
	"fmt"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
)

// Conversions from serverset template to server, volume, nic objects

// FromServerSetToServer is a conversion function that converts a ServerSet resource to a Server resource
// attaches a bootvolume to the server based on replicaIndex
func FromServerSetToServer(cr *v1alpha1.ServerSet, replicaIndex, version, volumeVersion int) v1alpha1.Server {
	serverType := "server"
	return v1alpha1.Server{
		ObjectMeta: metav1.ObjectMeta{
			Name:      GetNameFromIndex(cr.Name, serverType, replicaIndex, version),
			Namespace: cr.Namespace,
			Labels: map[string]string{
				ServerSetLabel: cr.Name,
				fmt.Sprintf(ServersetIndexLabel, serverType):   fmt.Sprintf("%d", replicaIndex),
				fmt.Sprintf(ServersetVersionLabel, serverType): fmt.Sprintf("%d", version),
			},
		},
		ManagementPolicies: xpv1.ManagementPolicies{"*"},
		Spec: v1alpha1.ServerSpec{
			ForProvider: v1alpha1.ServerParameters{
				DatacenterCfg:    cr.Spec.ForProvider.DatacenterCfg,
				Name:             GetNameFromIndex(cr.Name, serverType, replicaIndex, version),
				Cores:            cr.Spec.ForProvider.Template.Spec.Cores,
				RAM:              cr.Spec.ForProvider.Template.Spec.RAM,
				AvailabilityZone: "AUTO",
				CPUFamily:        cr.Spec.ForProvider.Template.Spec.CPUFamily,
				VolumeCfg: v1alpha1.VolumeConfig{
					VolumeIDRef: &xpv1.Reference{
						Name: GetNameFromIndex(cr.Name, "bootvolume", replicaIndex, volumeVersion),
					},
				},
			},
		}}
}

func FromServerSetToVolume(cr *v1alpha1.ServerSet, name string, replicaIndex, version int) v1alpha1.Volume {
	return v1alpha1.Volume{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cr.Namespace,
			Labels: map[string]string{
				ServerSetLabel: cr.Name,
				fmt.Sprintf(ServersetIndexLabel, "bootvolume"):   fmt.Sprintf("%d", replicaIndex),
				fmt.Sprintf(ServersetVersionLabel, "bootvolume"): fmt.Sprintf("%d", version),
			},
		},
		ManagementPolicies: xpv1.ManagementPolicies{"*"},
		Spec: v1alpha1.VolumeSpec{
			ForProvider: v1alpha1.VolumeParameters{
				DatacenterCfg:    cr.Spec.ForProvider.DatacenterCfg,
				Name:             name,
				AvailabilityZone: "AUTO",
				Size:             cr.Spec.ForProvider.BootVolumeTemplate.Spec.Size,
				Type:             cr.Spec.ForProvider.BootVolumeTemplate.Spec.Type,
				Image:            cr.Spec.ForProvider.BootVolumeTemplate.Spec.Image,
				// todo add to template(?)
				ImagePassword: "imagePassword776",
			},
		}}
}

func FromServerSetToNic(cr *v1alpha1.ServerSet, name, serverID, lanID string, replicaIndex, version int) v1alpha1.Nic {
	return v1alpha1.Nic{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cr.GetNamespace(),
			Labels: map[string]string{
				ServerSetLabel:                            cr.Name,
				fmt.Sprintf(ServersetIndexLabel, "nic"):   fmt.Sprintf("%d", replicaIndex),
				fmt.Sprintf(ServersetVersionLabel, "nic"): fmt.Sprintf("%d", version),
			},
		},
		ManagementPolicies: xpv1.ManagementPolicies{"*"},
		Spec: v1alpha1.NicSpec{
			ForProvider: v1alpha1.NicParameters{
				Name:          name,
				DatacenterCfg: cr.Spec.ForProvider.DatacenterCfg,
				ServerCfg: v1alpha1.ServerConfig{
					ServerID: serverID,
				},
				LanCfg: v1alpha1.LanConfig{
					LanID: lanID,
				},
			},
		},
	}
}
