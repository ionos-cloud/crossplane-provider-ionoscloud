/*
Copyright 2022 The Crossplane Authors.

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

// Role is the role of a ServerSet Replica. It can be ACTIVE or PASSIVE. The default value is PASSIVE.
// When a ServerSet Replica has role ACTIVE, it is the primary server and is used to serve the traffic.
type Role string

const (
	// Active means that the ServerSet Replica is the primary server and is used to serve the traffic.
	Active Role = "ACTIVE"
	// Passive means that the ServerSet Replica is the secondary server and is not used to serve the traffic.
	Passive Role = "PASSIVE"
)

// ServerSetParameters are the configurable fields of a ServerSet.
type ServerSetParameters struct {
	// The number of servers that will be created. Cannot be decreased once set, only increased.
	//
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=1
	Replicas int `json:"replicas"`
	// DatacenterConfig contains information about the datacenter resource
	// on which the server will be created.
	//
	// +kubebuilder:validation:Required
	DatacenterCfg DatacenterConfig `json:"datacenterConfig"`

	Template           ServerSetTemplate  `json:"template"`
	BootVolumeTemplate BootVolumeTemplate `json:"bootVolumeTemplate"`
}

// ServerSetTemplateSpec are the configurable fields of a ServerSetTemplateSpec.
type ServerSetTemplateSpec struct {
	// CPU architecture on which server gets provisioned; not all CPU architectures are available in all datacenter regions;
	// available CPU architectures can be retrieved from the datacenter resource.
	//
	// +immutable
	// +kubebuilder:validation:Enum=AMD_OPTERON;INTEL_SKYLAKE;INTEL_XEON
	CPUFamily string `json:"cpuFamily,omitempty"`
	// The total number of cores for the server.
	//
	// +kubebuilder:validation:Required
	Cores int32 `json:"cores"`
	// The memory size for the server in MB, such as 2048. Size must be specified in multiples of 256 MB with a minimum of 256 MB.
	// however, if you set ramHotPlug to TRUE then you must use a minimum of 1024 MB. If you set the RAM size more than 240GB,
	// then ramHotPlug will be set to FALSE and can not be set to TRUE unless RAM size not set to less than 240GB.
	//
	// +kubebuilder:validation:MultipleOf=1024
	// +kubebuilder:validation:Required
	RAM int32 `json:"ram"`
	// NICs are the network interfaces of the server.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	NICs []ServerSetTemplateNIC `json:"nics,omitempty"`
}

// ServerSetTemplateNIC are the configurable fields of a ServerSetTemplateNIC.
// +kubebuilder:validation:XValidation:rule="!has(self.dhcpv6) || (self.dhcp == false && self.dhcpv6 == false) || (self.dhcp != self.dhcpv6)", message="Only one of 'dhcp' or 'dhcpv6' can be set to true"
type ServerSetTemplateNIC struct {
	// Name of the NIC. Replica index, NIC index, and version are appended to the name. Resulting name will be in format: {name}-{replicaIndex}-{nicIndex}-{version}.
	// Version increases if the NIC is re-created due to an immutable field changing. E.g. if the bootvolume type or image are changed and the strategy is createAllBeforeDestroy, the NIC is re-created and the version is increased.
	//
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern="[a-z0-9]([-a-z0-9]*[a-z0-9])?"
	// +kubebuilder:validation:MaxLength=50
	Name string `json:"name"`
	// +kubebuilder:validation:Required
	DHCP bool `json:"dhcp"`
	// +kubebuilder:validation:Optional
	DHCPv6 *bool `json:"dhcpv6"`
	// +kubebuilder:validation:Optional
	VNetID string `json:"vnetId,omitempty"`
	// +kubebuilder:validation:Required
	Reference string `json:"reference"`
}

// ServerSetTemplateVolumeMount are the configurable fields of a ServerSetTemplateVolumeMount.
// It is used to mount a volume to a server.
type ServerSetTemplateVolumeMount struct {
	// +kubebuilder:validation:Required
	Reference string `json:"reference"`
}

// ServerSetTemplate are the configurable fields of a ServerSetTemplate.
type ServerSetTemplate struct {
	// +kubebuilder:validation:Required
	Metadata ServerSetMetadata `json:"metadata"`
	// +kubebuilder:validation:Required
	Spec ServerSetTemplateSpec `json:"spec"`
}

// ServerSetMetadata are the configurable fields of a ServerSetMetadata.
type ServerSetMetadata struct {
	// Name of the Server. Replica index and version are appended to the name. Resulting name will be in format: {name}-{replicaIndex}-{version}
	// Version increases if the Server is re-created due to an immutable field changing. E.g. if the bootvolume type or image are changed and the strategy is createAllBeforeDestroy, the Server is re-created and the version is increased.
	//
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern="[a-z0-9]([-a-z0-9]*[a-z0-9])?"
	// +kubebuilder:validation:MaxLength=55
	Name string `json:"name"`
	// +optional
	Labels map[string]string `json:"labels,omitempty"`
}

// ServerSetObservation are the observable fields of a ServerSet.
type ServerSetObservation struct {
	// Replicas is the count of ready replicas.
	Replicas        int                      `json:"replicas,omitempty"`
	ReplicaStatuses []ServerSetReplicaStatus `json:"replicaStatus,omitempty"`
}

// ServerSetReplicaStatus contains the status of a Server Replica.
type ServerSetReplicaStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	// +kubebuilder:validation:Enum=ACTIVE;PASSIVE
	Role         Role        `json:"role"`
	Name         string      `json:"name"`
	ReplicaIndex int         `json:"replicaIndex"`
	NICStatuses  []NicStatus `json:"nicStatus,omitempty"`
	// +kubebuilder:validation:Enum=UNKNOWN;READY;ERROR;BUSY
	Status string `json:"status"`
	// ErrorMessage relayed from the backend.
	ErrorMessage string      `json:"errorMessage,omitempty"`
	LastModified metav1.Time `json:"lastModified,omitempty"`
}

// A ServerSetSpec defines the desired state of a ServerSet.
type ServerSetSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       ServerSetParameters `json:"forProvider"`
}

// A ServerSetStatus represents the observed state of a ServerSet.
type ServerSetStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          ServerSetObservation `json:"atProvider,omitempty"`
}

// BootVolumeTemplate are the configurable fields of a BootVolumeTemplate.
type BootVolumeTemplate struct {
	// +kubebuilder:validation:Optional
	Metadata ServerSetBootVolumeMetadata `json:"metadata"`
	// +kubebuilder:validation:Required
	Spec ServerSetBootVolumeSpec `json:"spec"`
}

// ServerSetBootVolumeMetadata are the configurable fields of a ServerSetBootVolumeMetadata.
type ServerSetBootVolumeMetadata struct {
	// Name of the BootVolume. Replica index, volume index, and version are appended to the name. Resulting name will be in format: {name}-{replicaIndex}-{version}. Version increases if the bootvolume is
	// re-created due to an immutable field changing. E.g. if the image or the disk type are changed, the bootvolume is re-created and the version is increased.
	//
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern="[a-z0-9]([-a-z0-9]*[a-z0-9])?"
	// +kubebuilder:validation:MaxLength=55
	Name string `json:"name"`
	// +optional
	Labels map[string]string `json:"labels,omitempty"`
}

// ServerSetBootVolumeSpec are the configurable fields of a ServerSetBootVolumeSpec.
// +kubebuilder:validation:XValidation:rule="has(self.imagePassword) || has(self.sshKeys)",message="either imagePassword or sshKeys must be set"
type ServerSetBootVolumeSpec struct {
	// Image or snapshot ID to be used as template for this volume.
	// Make sure the image selected is compatible with the datacenter's location.
	// Note: when creating a volume and setting image, set imagePassword or SSKeys as well.
	//
	// +immutable
	// +kubebuilder:validation:Required
	Image string `json:"image,omitempty"`
	// The size of the volume in GB.
	//
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:XValidation:rule="self >= oldSelf", message="Size cannot be decreased once set, only increased"
	Size float32 `json:"size"`
	// Changing type re-creates either the bootvolume, or the bootvolume, server and nic depending on the UpdateStrategy chosen`
	//
	// +immutable
	// +kubebuilder:validation:Enum=HDD;SSD;SSD Standard;SSD Premium;DAS;ISO
	// +kubebuilder:validation:Required
	Type string `json:"type"`
	// The cloud-init configuration for the volume as base64 encoded string.
	// The property is immutable and is only allowed to be set on creation of a new a volume.
	// It is mandatory to provide either 'public image' or 'imageAlias' that has cloud-init compatibility in conjunction with this property.
	//
	// +immutable
	UserData string `json:"userData,omitempty"`
	// Initial password to be set for installed OS. Works with public images only. Not modifiable, forbidden in update requests.
	// Password rules allows all characters from a-z, A-Z, 0-9.
	//
	// +immutable
	// +kubebuilder:validation:MinLength=8
	// +kubebuilder:validation:MaxLength=50
	// +kubebuilder:validation:Pattern="^[A-Za-z0-9]+$"
	ImagePassword string `json:"imagePassword,omitempty"`
	// Public SSH keys are set on the image as authorized keys for appropriate SSH login to the instance using the corresponding private key.
	// This field may only be set in creation requests. When reading, it always returns null.
	// SSH keys are only supported if a public Linux image is used for the volume creation.
	//
	// +immutable
	SSHKeys  []string             `json:"sshKeys,omitempty"`
	Selector metav1.LabelSelector `json:"selector,omitempty"`
	// UpdateStrategy is the update strategy when changing immutable fields on boot volume. The default value is createBeforeDestroyBootVolume which creates a new bootvolume before deleting the old one

	// +kubebuilder:validation:Required
	UpdateStrategy UpdateStrategy `json:"updateStrategy,omitempty"`
}

// UpdateStrategy is the update strategy for the boot volume.
type UpdateStrategy struct {
	// +kubebuilder:validation:Enum=createAllBeforeDestroy;createBeforeDestroyBootVolume
	// +kubebuilder:default=createBeforeDestroyBootVolume
	Stype UpdateStrategyType `json:"type"`
}

// UpdateStrategyType is the type of the update strategy for the boot volume.
type UpdateStrategyType string

const (
	// CreateAllBeforeDestroy creates server, boot volume, and NIC before destroying the old ones.
	CreateAllBeforeDestroy UpdateStrategyType = "createAllBeforeDestroy"
	// CreateBeforeDestroyBootVolume creates boot volume before destroying the old one.
	CreateBeforeDestroyBootVolume = "createBeforeDestroyBootVolume"
)

// +kubebuilder:object:root=true

// A ServerSet is an example API type.
// +kubebuilder:resource:scope=Cluster,categories=crossplane,shortName=sset;ss
// +kubebuilder:printcolumn:name="Datacenter ID",type="string",JSONPath=".spec.forProvider.datacenterConfig.datacenterId"
// +kubebuilder:printcolumn:name="REPLICAS",type="integer",JSONPath=".status.atProvider.replicas"
// +kubebuilder:printcolumn:name="servers",priority=1,type="string",JSONPath=".status.atProvider.replicaStatus"
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,ionoscloud},shortName=ss;sset
// +kubebuilder:subresource:scale:specpath=.spec.forProvider.replicas,statuspath=.status.atProvider.replicas,selectorpath=.status.selector
type ServerSet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ServerSetSpec   `json:"spec"`
	Status ServerSetStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ServerSetList contains a list of ServerSet
type ServerSetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ServerSet `json:"items"`
}

// ServerSet type metadata.
var (
	ServerSetKind             = reflect.TypeOf(ServerSet{}).Name()
	ServerSetGroupKind        = schema.GroupKind{Group: Group, Kind: ServerSetKind}.String()
	ServerSetKindAPIVersion   = ServerSetKind + "." + SchemeGroupVersion.String()
	ServerSetGroupVersionKind = SchemeGroupVersion.WithKind(ServerSetKind)
)

func init() {
	SchemeBuilder.Register(&ServerSet{}, &ServerSetList{})
}
