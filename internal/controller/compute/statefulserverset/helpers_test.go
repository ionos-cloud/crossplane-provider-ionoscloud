package statefulserverset

import (
	"fmt"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
)

const (
	customerLanName     = "customer"
	customerLanIPv6cidr = "AUTO"
	customerLanDHCP     = true

	managementLanName = "management"
	managementLanDHCP = false

	dataVolume1Name = "storage_disk"
	dataVolume1Size = 10
	dataVolume1Type = "SSD"

	dataVolume2Name = "storage_disk_extend_1"
	dataVolume2Size = 10
	dataVolume2Type = "SSD"

	statefulServerSetName         = "statefulserverset"
	statefulServerSetExternalName = "test"

	datacenterName = "example-datacenter"

	serverSetName = "serverset"

	serverSetNrOfReplicas = 2
)

func createSSSet() *v1alpha1.StatefulServerSet {
	return &v1alpha1.StatefulServerSet{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: statefulServerSetName,
			Annotations: map[string]string{
				"crossplane.io/external-name": statefulServerSetExternalName,
			},
		},
		Spec: v1alpha1.StatefulServerSetSpec{
			ForProvider: v1alpha1.StatefulServerSetParameters{
				Replicas: 2,
				Template: createSSetTemplate(),
				Lans: []v1alpha1.StatefulServerSetLan{
					{
						Metadata: v1alpha1.StatefulServerSetLanMetadata{
							Name: customerLanName,
						},
						Spec: v1alpha1.StatefulServerSetLanSpec{
							IPv6cidr: customerLanIPv6cidr,
							DHCP:     customerLanDHCP,
						},
					},
					{
						Metadata: v1alpha1.StatefulServerSetLanMetadata{
							Name: managementLanName,
						},
						Spec: v1alpha1.StatefulServerSetLanSpec{
							DHCP: managementLanDHCP,
						},
					},
				},
				Volumes: []v1alpha1.StatefulServerSetVolume{
					{
						Metadata: v1alpha1.StatefulServerSetVolumeMetadata{
							Name: dataVolume1Name,
						},
						Spec: v1alpha1.StatefulServerSetVolumeSpec{
							Size: dataVolume1Size,
							Type: dataVolume1Type,
						},
					},
					{
						Metadata: v1alpha1.StatefulServerSetVolumeMetadata{
							Name: dataVolume2Name,
						},
						Spec: v1alpha1.StatefulServerSetVolumeSpec{
							Size: dataVolume2Size,
							Type: dataVolume2Type,
						},
					},
				},
			},
		},
	}
}

func createLanList() v1alpha1.LanList {
	return v1alpha1.LanList{
		Items: []v1alpha1.Lan{
			{
				Spec: v1alpha1.LanSpec{
					ForProvider: v1alpha1.LanParameters{
						Name:     customerLanName,
						Public:   customerLanDHCP,
						Ipv6Cidr: customerLanIPv6cidr,
					},
				},
			},
			{
				Spec: v1alpha1.LanSpec{
					ForProvider: v1alpha1.LanParameters{
						Name:   managementLanName,
						Public: managementLanDHCP,
					},
				},
			},
		},
	}
}

func createLanListNotUpToDate() v1alpha1.LanList {
	lans := createLanList()
	lans.Items[0].Spec.ForProvider.Public = false
	return lans
}

func createVolumeList() v1alpha1.VolumeList {
	return v1alpha1.VolumeList{
		Items: []v1alpha1.Volume{
			createVolume(0, v1alpha1.VolumeParameters{
				Name: dataVolume1Name,
				Size: dataVolume1Size,
				Type: dataVolume1Type,
			}),
			createVolume(0, v1alpha1.VolumeParameters{
				Name: dataVolume2Name,
				Size: dataVolume2Size,
				Type: dataVolume2Type,
			}),
			createVolume(1, v1alpha1.VolumeParameters{
				Name: dataVolume1Name,
				Size: dataVolume1Size,
				Type: dataVolume1Type,
			}),
			createVolume(1, v1alpha1.VolumeParameters{
				Name: dataVolume2Name,
				Size: dataVolume2Size,
				Type: dataVolume2Type,
			}),
		},
	}
}

func createVolume(replicaIdx int, prop v1alpha1.VolumeParameters) v1alpha1.Volume {
	return v1alpha1.Volume{
		Spec: v1alpha1.VolumeSpec{
			ForProvider: v1alpha1.VolumeParameters{
				Name: fmt.Sprintf("%s-%d", prop.Name, replicaIdx),
				Size: prop.Size,
				Type: prop.Type,
			},
		},
	}
}

func createSSet() *v1alpha1.ServerSet {
	return &v1alpha1.ServerSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:            statefulServerSetName + "-" + serverSetName,
			ResourceVersion: "1",
			Labels: map[string]string{
				statefulServerSetLabel: statefulServerSetName,
			},
		},
		Spec: v1alpha1.ServerSetSpec{
			ResourceSpec: xpv1.ResourceSpec{
				ManagementPolicies:      []xpv1.ManagementAction{"*"},
				ProviderConfigReference: &xpv1.Reference{Name: "example"},
			},
			ForProvider: v1alpha1.ServerSetParameters{
				Replicas: serverSetNrOfReplicas,
				DatacenterCfg: v1alpha1.DatacenterConfig{
					DatacenterIDRef: &xpv1.Reference{Name: datacenterName},
				},
				Template: createSSetTemplate(),
			},
		},
	}
}

func createSSetTemplate() v1alpha1.ServerSetTemplate {
	return v1alpha1.ServerSetTemplate{
		Metadata: v1alpha1.ServerSetMetadata{
			Name: serverSetName,
			Labels: map[string]string{
				"aKey": "aValue",
			},
		},
		Spec: v1alpha1.ServerSetTemplateSpec{
			CPUFamily: "INTEL_XEON",
			Cores:     1,
			RAM:       1024,
			NICs: []v1alpha1.ServerSetTemplateNIC{
				{
					Name:      "nic-1",
					IPv4:      "10.0.0.1/24",
					Reference: "examplelan",
				},
			},
			VolumeMounts: []v1alpha1.ServerSetTemplateVolumeMount{
				{
					Reference: "volume-mount-id",
				},
			},
			BootStorageVolumeRef: "volume-id",
		},
	}
}
