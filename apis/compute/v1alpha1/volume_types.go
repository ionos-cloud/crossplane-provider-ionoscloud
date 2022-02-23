/*
Copyright 2020 The Crossplane Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"reflect"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
)

// VolumeProperties are the observable fields of a Volume.
type VolumeProperties struct {
	// +immutable
	// +crossplane:generate:reference:type=Datacenter
	DatacenterID string `json:"datacenterID,omitempty"`
	// DatacenterRef references to a Datacenter to retrieve its name
	// +optional
	DatacenterIDRef *xpv1.Reference `json:"datacenterIDRef,omitempty"`
	// +optional
	DatacenterIDSelector *xpv1.Selector `json:"datacenterIDSelector,omitempty"`
	// The name of the  resource.
	Name string `json:"name,omitempty"`
	// Hardware type of the volume. DAS (Direct Attached Storage) could be used only in a composite call with a Cube server.
	Type string `json:"type,omitempty"`
	// The size of the volume in GB.
	Size float32 `json:"size"`
	// The availability zone in which the volume should be provisioned.
	// The storage volume will be provisioned on as few physical storage devices as possible, but this cannot be guaranteed upfront.
	// This is unavailable for DAS (Direct Attached Storage), and subject to availability for SSD.
	AvailabilityZone string `json:"availabilityZone,omitempty"`
	// Image or snapshot ID to be used as template for this volume.
	Image string `json:"image,omitempty"`
	// Initial password to be set for installed OS. Works with public images only. Not modifiable, forbidden in update requests.
	// Password rules allows all characters from a-z, A-Z, 0-9.
	ImagePassword string `json:"imagePassword,omitempty"`
	ImageAlias    string `json:"imageAlias,omitempty"`
	// Public SSH keys are set on the image as authorized keys for appropriate SSH login to the instance using the corresponding private key.
	// This field may only be set in creation requests. When reading, it always returns null.
	// SSH keys are only supported if a public Linux image is used for the volume creation.
	SshKeys []string `json:"sshKeys,omitempty"`
	// The bus type of the volume. Default is VIRTIO
	Bus string `json:"bus,omitempty"`
	// OS type for this volume.
	LicenceType string `json:"licenceType,omitempty"`
	// Hot-plug capable CPU (no reboot required).
	CpuHotPlug bool `json:"cpuHotPlug,omitempty"`
	// Hot-plug capable RAM (no reboot required).
	RamHotPlug bool `json:"ramHotPlug,omitempty"`
	// Hot-plug capable NIC (no reboot required).
	NicHotPlug bool `json:"nicHotPlug,omitempty"`
	// Hot-unplug capable NIC (no reboot required).
	NicHotUnplug bool `json:"nicHotUnplug,omitempty"`
	// Hot-plug capable Virt-IO drive (no reboot required).
	DiscVirtioHotPlug bool `json:"discVirtioHotPlug,omitempty"`
	// Hot-unplug capable Virt-IO drive (no reboot required). Not supported with Windows VMs.
	DiscVirtioHotUnplug bool `json:"discVirtioHotUnplug,omitempty"`
	// The Logical Unit Number of the storage volume. Null for volumes, not mounted to a VM.
	DeviceNumber int64 `json:"deviceNumber,omitempty"`
	// The PCI slot number of the storage volume. Null for volumes, not mounted to a VM.
	PciSlot int32 `json:"pciSlot,omitempty"`
	// The ID of the backup unit that the user has access to.
	// The property is immutable and is only allowed to be set on creation of a new a volume.
	// It is mandatory to provide either 'public image' or 'imageAlias' in conjunction with this property.
	BackupunitId string `json:"backupunitId,omitempty"`
	// The cloud-init configuration for the volume as base64 encoded string.
	// The property is immutable and is only allowed to be set on creation of a new a volume.
	// It is mandatory to provide either 'public image' or 'imageAlias' that has cloud-init compatibility in conjunction with this property.
	UserData string `json:"userData,omitempty"`
}

// VolumeObservation are the observable fields of a Volume.
type VolumeObservation struct {
	VolumeID string `json:"volumeID,omitempty"`
	State    string `json:"state,omitempty"`
}

// A VolumeSpec defines the desired state of a Volume.
type VolumeSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       VolumeProperties `json:"forProvider"`
}

// A VolumeStatus represents the observed state of a Volume.
type VolumeStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          VolumeObservation `json:"atProvider,omitempty"`
}

//// ResolveReferences implements the ReferenceResolver interface for a resource.
//// It's called to resolve references in the server to e.g. the datacenter.
//func (in *Volume) ResolveReferences(ctx context.Context, c client.Reader) error {
//	r := reference.NewAPIResolver(c, in)
//	const undefined = ""
//	// Resolve spec.forProvider.datacenterID
//	datacenterID, err := r.Resolve(ctx, reference.ResolutionRequest{
//		CurrentValue: in.Spec.ForProvider.DatacenterID,
//		Reference:    in.Spec.ForProvider.DatacenterIDRef,
//		Selector:     in.Spec.ForProvider.DatacenterIDRefSelector,
//		To:           reference.To{Managed: &Datacenter{}, List: &DatacenterList{}},
//		Extract: func(managed resource.Managed) string {
//			c, ok := managed.(*Datacenter)
//			if !ok {
//				return undefined
//			}
//			if meta.GetExternalName(c) == c.Name {
//				return undefined
//			}
//			return meta.GetExternalName(c)
//		},
//	})
//	if err != nil {
//		return errors.Wrap(err, "spec.forProvider.datacenterID")
//	}
//	if datacenterID.ResolvedValue == undefined {
//		return errors.New("datacenter either not found or not reconciled yet")
//	}
//	in.Spec.ForProvider.DatacenterID = datacenterID.ResolvedValue
//	in.Spec.ForProvider.DatacenterIDRef = datacenterID.ResolvedReference
//	return nil
//}

// +kubebuilder:object:root=true

// A Volume is an example API type.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="DATACENTER ID",type="string",JSONPath=".spec.forProvider.datacenterID"
// +kubebuilder:printcolumn:name="ID",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="STATE",type="string",JSONPath=".status.atProvider.state"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,template}
type Volume struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VolumeSpec   `json:"spec"`
	Status VolumeStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// VolumeList contains a list of Volume
type VolumeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Volume `json:"items"`
}

// Volume type metadata.
var (
	VolumeKind             = reflect.TypeOf(Volume{}).Name()
	VolumeGroupKind        = schema.GroupKind{Group: Group, Kind: VolumeKind}.String()
	VolumeKindAPIVersion   = VolumeKind + "." + SchemeGroupVersion.String()
	VolumeGroupVersionKind = SchemeGroupVersion.WithKind(VolumeKind)
)

func init() {
	SchemeBuilder.Register(&Volume{}, &VolumeList{})
}
