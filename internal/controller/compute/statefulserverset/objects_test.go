package statefulserverset

import (
	"fmt"
	"strconv"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/ionos-cloud/sdk-go-bundle/shared"
	ionoscloud "github.com/ionos-cloud/sdk-go/v6"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
)

const (
	bootVolumeName  = "bootvolume"
	bootVolumeSize  = 10
	bootVolumeImage = "ubuntu-20.04"
	bootVolumeType  = "SSD"

	customerLanName         = "customer"
	customerLanIPv6cidrAuto = v1alpha1.LANAuto
	customerLanIPv6cidr1    = "1000:db8::/64"
	customerLanIPv6cidr2    = "2000:db8::/64"
	customerLanPublic       = true

	dataVolume1Name = "storage_disk"
	dataVolume1Size = 10
	dataVolume1Type = "SSD"

	dataVolume2Name = "storage_disk_extend_1"
	dataVolume2Size = 10
	dataVolume2Type = "SSD"

	datacenterName = "example-datacenter"

	lanResourceVersion = 1

	managementLanName   = "management"
	managementLanPublic = false

	nicName = "nic-1"
	nicLAN  = "examplelan"

	serverSetName         = "serverset"
	serverSetNrOfReplicas = 2
	serverSetLabel        = "serverset"

	statefulServerSetName         = "statefulserverset"
	statefulServerSetExternalName = "test"
	stateAvailable                = "AVAILABLE"
	stateBusy                     = "BUSY"

	serverName      = "server"
	serverCPUFamily = "INTEL_XEON"
	serverCores     = 1
	serverRAM       = 1024

	volumeID1 = "volume-id-1"
	volumeID2 = "volume-id-2"
)

var bootVolumeParameters = v1alpha1.VolumeParameters{
	Name:  bootVolumeName,
	Size:  bootVolumeSize,
	Image: bootVolumeImage,
	Type:  bootVolumeType,
}

var serverParameters = v1alpha1.ServerParameters{
	Name:      serverName,
	Cores:     serverCores,
	RAM:       serverRAM,
	CPUFamily: serverCPUFamily,
}

type VolumeFieldUpToDate struct {
	isSizeUpToDate bool
}

type LANFieldsUpToDate struct {
	isPublicUpToDate   bool
	isIpv6CidrUpToDate bool
}

type BootVolumeFieldsUpToDate struct {
	isSizeUpToDate bool
}

type ServeFieldsUpToDate struct {
	areCoresUpToDate bool
}

func createSSSetWithCustomerLanUpdated(params v1alpha1.StatefulServerSetLanSpec) *v1alpha1.StatefulServerSet {
	ssset := createSSSet()
	lanIdx := getCustomerLanIdx(ssset)
	ssset.Spec.ForProvider.Lans[lanIdx].Spec = params
	return ssset
}

func getCustomerLanIdx(ssset *v1alpha1.StatefulServerSet) int {
	for lanIdx, lan := range ssset.Spec.ForProvider.Lans {
		if lan.Metadata.Name == customerLanName {
			return lanIdx
		}
	}
	return -1
}

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
				DatacenterCfg: v1alpha1.DatacenterConfig{
					DatacenterIDRef: &xpv1.Reference{Name: datacenterName},
				},
				Replicas: serverSetNrOfReplicas,
				Template: createSSetTemplate(),
				BootVolumeTemplate: v1alpha1.BootVolumeTemplate{
					Spec: v1alpha1.ServerSetBootVolumeSpec{
						Size:  bootVolumeSize,
						Image: bootVolumeImage,
						Type:  bootVolumeType,
					},
				},
				Lans: []v1alpha1.StatefulServerSetLan{
					{
						Metadata: v1alpha1.StatefulServerSetLanMetadata{
							Name: customerLanName,
						},
						Spec: v1alpha1.StatefulServerSetLanSpec{
							IPv6cidr: customerLanIPv6cidrAuto,
							Public:   customerLanPublic,
						},
					},
					{
						Metadata: v1alpha1.StatefulServerSetLanMetadata{
							Name: managementLanName,
						},
						Spec: v1alpha1.StatefulServerSetLanSpec{
							Public: managementLanPublic,
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

func createSSet() *v1alpha1.ServerSet {
	return &v1alpha1.ServerSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:            computeSSSetOwnerLabel(),
			ResourceVersion: "1",
			Labels: map[string]string{
				statefulServerSetLabel: statefulServerSetName,
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         "",
					Kind:               "",
					Name:               statefulServerSetName,
					UID:                "",
					Controller:         shared.ToPtr(true),
					BlockOwnerDeletion: shared.ToPtr(false),
				},
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
			CPUFamily: serverCPUFamily,
			Cores:     serverCores,
			RAM:       serverRAM,
			NICs: []v1alpha1.ServerSetTemplateNIC{
				{
					Name:         nicName,
					DHCP:         true,
					LanReference: nicLAN,
				},
			},
		},
	}
}

func createLanList() v1alpha1.LanList {
	return v1alpha1.LanList{
		Items: []v1alpha1.Lan{
			*createLAN(v1alpha1.LanParameters{
				Name:     customerLanName,
				Public:   customerLanPublic,
				Ipv6Cidr: customerLanIPv6cidrAuto,
			}),
			*createLAN(v1alpha1.LanParameters{
				Name:   managementLanName,
				Public: managementLanPublic,
			}),
		},
	}
}

func createCustomerLANWithIpv6CidrUpdated() *v1alpha1.Lan {
	lan := createCustomerLANWithIpv6Cidr()
	lan.ResourceVersion = strconv.Itoa(lanResourceVersion + 1)
	lan.Spec.ForProvider.Ipv6Cidr = customerLanIPv6cidr2
	return lan
}

func createCustomerLANWithIpv6Cidr() *v1alpha1.Lan {
	lan := createCustomerLAN()
	lan.Spec.ForProvider.Ipv6Cidr = customerLanIPv6cidr1
	lan.Status = v1alpha1.LanStatus{
		AtProvider: v1alpha1.LanObservation{
			LanID: "lan-id",
			State: ionoscloud.Available,
		},
	}
	return lan
}

func createCustomerLAN() *v1alpha1.Lan {
	return createLAN(v1alpha1.LanParameters{
		Name:     customerLanName,
		Public:   customerLanPublic,
		Ipv6Cidr: customerLanIPv6cidrAuto,
	})
}

func createLAN(parameters v1alpha1.LanParameters) *v1alpha1.Lan {
	return &v1alpha1.Lan{
		ObjectMeta: metav1.ObjectMeta{
			Name:            parameters.Name,
			ResourceVersion: strconv.Itoa(lanResourceVersion),
		},
		Spec: v1alpha1.LanSpec{ForProvider: parameters},
		Status: v1alpha1.LanStatus{
			AtProvider: v1alpha1.LanObservation{
				LanID: "lan-id",
				State: ionoscloud.Available,
			},
		},
	}
}

func createLanListNotUpToDate(l LANFieldsUpToDate) v1alpha1.LanList {
	lans := createLanList()

	for idx := range lans.Items {
		updateFieldIpv6Cidr(l, lans, idx)
		updateFieldPublic(l, lans, idx)
	}
	return lans
}

func updateFieldPublic(l LANFieldsUpToDate, lans v1alpha1.LanList, idx int) {
	if !l.isPublicUpToDate {
		other := findOtherPublic(lans.Items[idx].Spec.ForProvider.Public)
		lans.Items[idx].Spec.ForProvider.Public = other
	}
}

func updateFieldIpv6Cidr(l LANFieldsUpToDate, lans v1alpha1.LanList, idx int) {
	if !l.isIpv6CidrUpToDate {
		other := findOtherIpv6Cidr(lans.Items[idx].Spec.ForProvider.Ipv6Cidr)
		lans.Items[idx].Spec.ForProvider.Ipv6Cidr = other
	}
}

func findOtherIpv6Cidr(actual string) string {
	if actual == v1alpha1.LANAuto {
		return ""
	}
	return v1alpha1.LANAuto
}

func findOtherPublic(actual bool) bool {
	return !actual
}

func createVolumeListNotUpToDate(v VolumeFieldUpToDate) v1alpha1.VolumeList {
	volumes := createVolumeList()

	for idx := range volumes.Items {
		updateFieldSize(v, volumes, idx)
	}

	return volumes
}

func updateFieldSize(v VolumeFieldUpToDate, volumes v1alpha1.VolumeList, idx int) {
	if !v.isSizeUpToDate {
		volumes.Items[idx].Spec.ForProvider.Size *= 10
	}
}

func createBootVolume1() *v1alpha1.Volume {
	bootVolume := createBootVolume(0, bootVolumeParameters)
	return bootVolume
}

func createBootVolume2() *v1alpha1.Volume {
	bootVolume := createBootVolume(1, bootVolumeParameters)
	return bootVolume
}

func createBootVolume1NotUpToDate() *v1alpha1.Volume {
	bootVolume := createBootVolumeNotUpToDate(0, BootVolumeFieldsUpToDate{})
	return bootVolume
}
func createBootVolume2NotUpToDate() *v1alpha1.Volume {
	bootVolume := createBootVolumeNotUpToDate(1, BootVolumeFieldsUpToDate{})
	return bootVolume
}

func createBootVolumeNotUpToDate(replicaIdx int, b BootVolumeFieldsUpToDate) *v1alpha1.Volume {
	bootVolume := createBootVolume(replicaIdx, bootVolumeParameters)

	if !b.isSizeUpToDate {
		bootVolume.Spec.ForProvider.Size *= 10
	}

	return bootVolume
}

func createBootVolume(replicaIdx int, prop v1alpha1.VolumeParameters) *v1alpha1.Volume {
	bootVolume := createVolume(replicaIdx, prop)
	bootVolume.ObjectMeta.Labels = map[string]string{
		serverSetLabel: computeSSSetOwnerLabel(),
	}

	return &bootVolume
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

func createVolumeWithWrongIndexLabel() *v1alpha1.Volume {
	volume := createVolumeWithStatus()
	volume.Labels = map[string]string{
		"wronglabel": "0",
	}
	return volume
}

func create2VolumesWithStatuses() []v1alpha1.Volume {
	volume1 := createVolumeWithStatus()
	volume2 := createVolumeWithStatus()
	volume2.Status.AtProvider = v1alpha1.VolumeObservation{
		VolumeID: volumeID2,
		State:    stateBusy,
		PCISlot:  2,
		Name:     dataVolume2Name,
	}

	return []v1alpha1.Volume{*volume1, *volume2}
}

func createVolumeWithStatus() *v1alpha1.Volume {
	volume := createVolumeDefault()
	volume.Labels = map[string]string{
		fmt.Sprintf("%s-dv-ri", serverName): "0",
	}
	volume.Status = v1alpha1.VolumeStatus{
		ResourceStatus: xpv1.ResourceStatus{},
		AtProvider: v1alpha1.VolumeObservation{
			VolumeID: volumeID1,
			State:    stateAvailable,
			PCISlot:  1,
			Name:     dataVolume1Name,
		},
	}
	return volume
}

func createVolumeDefault() *v1alpha1.Volume {
	volume := createVolume(0, v1alpha1.VolumeParameters{
		Name: dataVolume2Name,
		Size: dataVolume2Size,
		Type: dataVolume2Type,
	})
	return &volume
}

func createVolumeWithState(replicaIdx int, parameters v1alpha1.VolumeParameters, state string) v1alpha1.Volume {
	volume := createVolume(replicaIdx, parameters)
	volume.Status.AtProvider.State = state
	return volume
}

func createVolume(replicaIdx int, parameters v1alpha1.VolumeParameters) v1alpha1.Volume {
	withNameUpdated := parameters
	withNameUpdated.Name = nameWithIdx(replicaIdx, parameters.Name)

	return v1alpha1.Volume{
		ObjectMeta: metav1.ObjectMeta{
			Name: withNameUpdated.Name,
		},

		Spec: v1alpha1.VolumeSpec{ForProvider: withNameUpdated},
	}
}

func createServer1() *v1alpha1.Server {
	return createServer(0, serverParameters)
}
func createServer2() *v1alpha1.Server {
	return createServer(1, serverParameters)
}

func createServer1NotUpToDate() *v1alpha1.Server {
	server := createServerNotUpToDate(0, ServeFieldsUpToDate{})
	return server
}
func createServer2NotUpToDate() *v1alpha1.Server {
	server := createServerNotUpToDate(1, ServeFieldsUpToDate{})
	return server
}

func createServerNotUpToDate(replicaIdx int, b ServeFieldsUpToDate) *v1alpha1.Server {
	server := createServer(replicaIdx, serverParameters)

	if !b.areCoresUpToDate {
		server.Spec.ForProvider.Cores *= 2
	}

	return server
}

func createServer(replicaIdx int, parameters v1alpha1.ServerParameters) *v1alpha1.Server {
	return &v1alpha1.Server{
		ObjectMeta: metav1.ObjectMeta{
			Name: nameWithIdx(replicaIdx, parameters.Name),
			Labels: map[string]string{
				serverSetLabel: computeSSSetOwnerLabel(),
			},
		},
		Spec: v1alpha1.ServerSpec{ForProvider: parameters},
	}
}

func createNIC1() *v1alpha1.Nic {
	return createNIC(0, v1alpha1.NicParameters{})
}

func createNIC2() *v1alpha1.Nic {
	return createNIC(1, v1alpha1.NicParameters{})
}

func createNIC(replicaIdx int, parameters v1alpha1.NicParameters) *v1alpha1.Nic {
	return &v1alpha1.Nic{
		ObjectMeta: metav1.ObjectMeta{
			Name: nameWithIdx(replicaIdx, parameters.Name),
			Labels: map[string]string{
				serverSetLabel: computeSSSetOwnerLabel(),
			},
		},
		Spec: v1alpha1.NicSpec{},
	}
}

func computeSSSetOwnerLabel() string {
	return serverSetName
}

func nameWithIdx(replicaIdx int, name string) string {
	return fmt.Sprintf("%s-%d", name, replicaIdx)
}
