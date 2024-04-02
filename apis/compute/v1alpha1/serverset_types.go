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

// ServerSetParameters are the configurable fields of a ServerSet.
type ServerSetParameters struct {
	// The number of servers that will be created.
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
	// The reference to the boot volume.
	// It must exist in the same data center as the server.
	// +kubebuilder:validation:Required
	BootStorageVolumeRef string `json:"bootStorageVolumeRef"`
	// The reference to the boot volume.
	// It must exist in the same data center as the server.
	// +kubebuilder:validation:Required
	VolumeMounts []ServerSetTemplateVolumeMount `json:"volumeMounts,omitempty"`
	// NICs are the network interfaces of the server.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	NICs []ServerSetTemplateNIC `json:"nics,omitempty"`
}

// ServerSetTemplateNIC are the configurable fields of a ServerSetTemplateNIC.
type ServerSetTemplateNIC struct {
	// todo add descriptions
	//
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern="[a-z0-9]([-a-z0-9]*[a-z0-9])?"
	// +kubebuilder:validation:MaxLength=63
	Name string `json:"name"`
	// +kubebuilder:validation:Required
	IPv4 string `json:"ipv4"`
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
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern="[a-z0-9]([-a-z0-9]*[a-z0-9])?"
	// +kubebuilder:validation:MaxLength=63
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

// ServerSetReplicaStatus are the observable fields of a ServerSetReplicaStatus.
type ServerSetReplicaStatus struct {
	Name string `json:"name"`
	// +kubebuilder:validation:Enum=UNKNOWN;READY;ERROR
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
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern="[a-z0-9]([-a-z0-9]*[a-z0-9])?"
	// +kubebuilder:validation:MaxLength=63
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
