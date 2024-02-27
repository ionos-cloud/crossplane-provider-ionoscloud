package serverset

import (
	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
)

// Conversions from serverset template to server, volume, nic objects

// fromServerSetToServer is a conversion function that converts a ServerSet resource to a Server resource
// attaches a bootvolume to the server based on replicaIndex
func fromServerSetToServer(cr *v1alpha1.ServerSet, replicaIndex int) v1alpha1.Server {
	serverType := "server"
	return v1alpha1.Server{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getNameFromIndex(cr.Name, serverType, replicaIndex),
			Namespace: cr.Namespace,
			Labels: map[string]string{
				serverSetLabel: cr.Name,
			},
		},
		ManagementPolicies: xpv1.ManagementPolicies{"*"},
		Spec: v1alpha1.ServerSpec{
			ForProvider: v1alpha1.ServerParameters{
				DatacenterCfg:    cr.Spec.ForProvider.DatacenterCfg,
				Name:             getNameFromIndex(cr.Name, serverType, replicaIndex),
				Cores:            cr.Spec.ForProvider.Template.Spec.Cores,
				RAM:              cr.Spec.ForProvider.Template.Spec.RAM,
				AvailabilityZone: "AUTO",
				CPUFamily:        cr.Spec.ForProvider.Template.Spec.CPUFamily,
				VolumeCfg: v1alpha1.VolumeConfig{
					VolumeIDRef: &xpv1.Reference{
						Name: getNameFromIndex(cr.Name, "bootvolume", replicaIndex),
					},
				},
			},
		}}
}

func fromServerSetToVolume(cr *v1alpha1.ServerSet, name string) v1alpha1.Volume {
	return v1alpha1.Volume{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cr.Namespace,
			Labels: map[string]string{
				serverSetLabel: cr.Name,
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

func fromServerSetToNic(cr *v1alpha1.ServerSet, name, serverID, lanID string) v1alpha1.Nic {
	return v1alpha1.Nic{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cr.GetNamespace(),
			Labels: map[string]string{
				serverSetLabel: cr.Name,
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
