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
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
)

// CubeServerProperties are the observable fields of a Cube Server.
// Required values when creating a CubeServer:
// Datacenter ID or Reference,
// Template ID or Name,
// Volume Properties (Name, Bus, Licence Type or Image/Image Alias).
type CubeServerProperties struct {
	// DatacenterConfig contains information about the datacenter resource
	// on which the server will be created.
	//
	// +immutable
	// +kubebuilder:validation:Required
	DatacenterCfg DatacenterConfig `json:"datacenterConfig"`
	// The ID or the name of the template for creating a CUBE server.
	//
	// +immutable
	// +kubebuilder:validation:Required
	Template Template `json:"template"`
	// The name of the  resource.
	Name string `json:"name,omitempty"`
	// The availability zone in which the server should be provisioned.
	//
	// +kubebuilder:validation:Enum=AUTO;ZONE_1;ZONE_2
	// +kubebuilder:default=AUTO
	AvailabilityZone string `json:"availabilityZone,omitempty"`
	// CPU architecture on which server gets provisioned; not all CPU architectures are available in all datacenter regions;
	// available CPU architectures can be retrieved from the datacenter resource.
	//
	// +kubebuilder:validation:Enum=AMD_OPTERON;INTEL_SKYLAKE;INTEL_XEON
	CPUFamily string `json:"cpuFamily,omitempty"`
	// DasVolumeProperties contains properties for the DAS volume attached to the Cube Server.
	//
	// +kubebuilder:validation:Required
	DasVolumeProperties DasVolumeProperties `json:"volume"`
}

// DasVolumeProperties are the observable fields of a Cube Server's DAS Volume.
// Required values when creating a DAS Volume:
// Name,
// Bus,
// Licence Type or Image/Image Alias.
// If using Image or Image Alias, you may want to provide also
// an Image Password or SSH Keys.
type DasVolumeProperties struct {
	// The name of the DAS Volume.
	//
	// +kubebuilder:validation:Required
	Name string `json:"name"`
	// The bus type of the volume.
	//
	// +kubebuilder:validation:Enum=VIRTIO;IDE;UNKNOWN
	// +kubebuilder:validation:Required
	Bus string `json:"bus"`
	// Image or snapshot ID to be used as template for this volume.
	// Make sure the image selected is compatible with the datacenter's location.
	// Note: when creating a volume - set image, image alias, or licence type.
	//
	// +immutable
	Image string `json:"image,omitempty"`
	// Initial password to be set for installed OS. Works with public images only. Not modifiable, forbidden in update requests.
	// Password rules allows all characters from a-z, A-Z, 0-9.
	//
	// +immutable
	ImagePassword string `json:"imagePassword,omitempty"`
	// Image Alias to be used for this volume.
	// Note: when creating a volume - set image, image alias, or licence type.
	//
	// +immutable
	ImageAlias string `json:"imageAlias,omitempty"`
	// Public SSH keys are set on the image as authorized keys for appropriate SSH login to the instance using the corresponding private key.
	// This field may only be set in creation requests. When reading, it always returns null.
	// SSH keys are only supported if a public Linux image is used for the volume creation.
	//
	// +immutable
	SSHKeys []string `json:"sshKeys,omitempty"`
	// OS type for this volume.
	// Note: when creating a volume - set image, image alias, or licence type.
	//
	// +immutable
	// +kubebuilder:validation:Enum=UNKNOWN;WINDOWS;WINDOWS2016;WINDOWS2022;LINUX;OTHER
	LicenceType string `json:"licenceType,omitempty"`
	// Hot-plug capable CPU (no reboot required).
	CPUHotPlug bool `json:"cpuHotPlug,omitempty"`
	// Hot-plug capable RAM (no reboot required).
	RAMHotPlug bool `json:"ramHotPlug,omitempty"`
	// Hot-plug capable NIC (no reboot required).
	NicHotPlug bool `json:"nicHotPlug,omitempty"`
	// Hot-unplug capable NIC (no reboot required).
	NicHotUnplug bool `json:"nicHotUnplug,omitempty"`
	// Hot-plug capable Virt-IO drive (no reboot required).
	DiscVirtioHotPlug bool `json:"discVirtioHotPlug,omitempty"`
	// Hot-unplug capable Virt-IO drive (no reboot required). Not supported with Windows VMs.
	DiscVirtioHotUnplug bool `json:"discVirtioHotUnplug,omitempty"`
	// BackupUnitCfg contains information about the backup unit resource
	// that the user has access to.
	// The property is immutable and is only allowed to be set on creation of a new a volume.
	// It is mandatory to provide either 'public image' or 'imageAlias' in conjunction with this property.
	//
	// +immutable
	BackupUnitCfg BackupUnitConfig `json:"backupUnitConfig,omitempty"`
	// The cloud-init configuration for the volume as base64 encoded string.
	// The property is immutable and is only allowed to be set on creation of a new a volume.
	// It is mandatory to provide either 'public image' or 'imageAlias' that has cloud-init compatibility in conjunction with this property.
	//
	// +immutable
	UserData string `json:"userData,omitempty"`
}

// Template refers to the Template used for Cube Servers.
type Template struct {
	// The name of the Template from IONOS Cloud.
	Name string `json:"name,omitempty"`
	// The ID of the Template from IONOS Cloud.
	//
	// +kubebuilder:validation:Format=uuid
	TemplateID string `json:"templateId,omitempty"`
}

// A CubeServerSpec defines the desired state of a Cube Server.
type CubeServerSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       CubeServerProperties `json:"forProvider"`
}

// +kubebuilder:object:root=true

// A CubeServer is an example API type.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="DATACENTER ID",type="string",JSONPath=".spec.forProvider.datacenterConfig.datacenterId"
// +kubebuilder:printcolumn:name="SERVER ID",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="VOLUME ID",priority=1,type="string",JSONPath=".status.atProvider.volumeId"
// +kubebuilder:printcolumn:name="STATE",type="string",JSONPath=".status.atProvider.state"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,ionoscloud}
type CubeServer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CubeServerSpec `json:"spec"`
	Status ServerStatus   `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CubeServerList contains a list of Server
type CubeServerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CubeServer `json:"items"`
}

// CubeServer type metadata.
var (
	CubeServerKind             = reflect.TypeOf(CubeServer{}).Name()
	CubeServerGroupKind        = schema.GroupKind{Group: APIGroup, Kind: CubeServerKind}.String()
	CubeServerKindAPIVersion   = CubeServerKind + "." + SchemeGroupVersion.String()
	CubeServerGroupVersionKind = SchemeGroupVersion.WithKind(CubeServerKind)
)

func init() {
	SchemeBuilder.Register(&CubeServer{}, &CubeServerList{})
}
