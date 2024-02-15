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

type DeploymentStrategy struct {
	// +kubebuilder:validation:Enum=ZONES
	Type string `json:"type"`
}

// StatefulServerSetMetadata are the configurable fields of a StatefulServerSetMetadata.
type StatefulServerSetMetadata struct {
	// +kubebuilder:validation:Required
	Name string `json:"name"`
	// +optional
	Labels map[string]string `json:"labels,omitempty"`
}

type StatefulServerSetTemplateSpecNic struct {
	// This references the LAN from the client.
	//
	// +kubebuilder:validation:Optional
	VNetId string `json:"vnetId"`
	// +kubebuilder:validation:Required
	LANReferenceName string `json:"lanReferenceName"`
}

// StatefulServerSetTemplateSpecVolumeMounts are the configurable fields of a StatefulServerSetTemplateSpecVolumeMounts.
// It is used to mount a volume to a server.
type StatefulServerSetTemplateSpecVolumeMounts struct {
	// +kubebuilder:validation:Required
	ReferenceName string `json:"referenceName"`
}

// StatefulServerSetTemplateSpec are the configurable fields of a StatefulServerSetTemplateSpec.
type StatefulServerSetTemplateSpec struct {
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
	//
	// +kubebuilder:validation:MultipleOf=256
	// +kubebuilder:validation:Required
	RAM int32 `json:"ram"`
	// NICs are the network interfaces of the server.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	NICs []StatefulServerSetTemplateSpecNic `json:"nics,omitempty"`
	// The reference to the boot volume.
	// It must exist in the same data center as the server.
	// +kubebuilder:validation:Required
	BootStorageVolume string `json:"bootStorageVolume"`
	// The reference to the other volumes.
	// They must exist in the same data center as the server.
	// +kubebuilder:validation:Required
	VolumeMounts []StatefulServerSetTemplateSpecVolumeMounts `json:"volumeMounts,omitempty"`
}

// StatefulServerSetTemplate are the configurable fields of a StatefulServerSetTemplate.
type StatefulServerSetTemplate struct {
	// +kubebuilder:validation:Required
	Metadata StatefulServerSetMetadata `json:"metadata"`
	// +kubebuilder:validation:Required
	Spec StatefulServerSetTemplateSpec `json:"spec"`
}

// StatefulServerSetLanMetadata are the configurable fields of a StatefulServerSetLanMetadata.
type StatefulServerSetLanMetadata struct {
	// +kubebuilder:validation:Required
	Name string `json:"name"`
	// +kubebuilder:validation:Optional
	Labels map[string]string `json:"labels,omitempty"`
}

type StatefulServerSetLanSpec struct {
	// +kubebuilder:validation:Optional
	IPv6 bool `json:"ipv6"`
	// +kubebuilder:validation:Optional
	DHCP bool `json:"dhcp"`
}

type StatefulServerSetLan struct {
	Metadata StatefulServerSetLanMetadata `json:"metadata"`
	Spec     StatefulServerSetLanSpec     `json:"spec"`
}

type StatefulServerSetVolumeMetadata struct {
	// +kubebuilder:validation:Required
	Name string `json:"name"`
	// +kubebuilder:validation:Optional
	Labels map[string]string `json:"labels,omitempty"`
}

type StatefulServerSetVolumeSpec struct {
	// The public image UUID or a public image alias.
	//
	// +kubebuilder:validation:Optional
	Image string `json:"image,omitempty"`
	// The size of the volume in GB.
	//
	// +kubebuilder:validation:Required
	Size float32 `json:"size"`
	// Hardware type of the volume.
	//
	// +immutable
	// +kubebuilder:validation:Enum=HDD;SSD;SSD Standard;SSD Premium
	// +kubebuilder:validation:Required
	Type string `json:"type"`
	// The cloud init configuration in base64 encoding.
	UserData string `json:"userData,omitempty"`
}

type StatefulServerSetVolume struct {
	Metadata StatefulServerSetVolumeMetadata `json:"metadata"`
	Spec     StatefulServerSetVolumeSpec     `json:"spec"`
}

// StatefulServerSetParameters are the configurable fields of a StatefulServerSet.
type StatefulServerSetParameters struct {
	// The number of servers that will be created.
	//
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=1
	Replicas           int                `json:"replicas"`
	DeploymentStrategy DeploymentStrategy `json:"deploymentStrategy"`
	// DatacenterConfig contains information about the datacenter resource
	// on which the server will be created.
	//
	// +kubebuilder:validation:Required
	DatacenterCfg DatacenterConfig          `json:"datacenterConfig"`
	Template      StatefulServerSetTemplate `json:"template"`
	Lans          []StatefulServerSetLan    `json:"lans"`
	VolumeSpec    []StatefulServerSetVolume `json:"volumes"`
}

// A StatefulServerSetSpec defines the desired state of a StatefulServerSet.
type StatefulServerSetSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       StatefulServerSetParameters `json:"forProvider"`
}

type StatefulServerSetReplicaStatus struct {
	// Server assigned role
	//
	// +kubebuilder:validation:Enum=ACTIVE;PASSIVE;REPLICA
	Role string `json:"role"`
	Name string `json:"name"`
	// +kubebuilder:validation:Enum=UNKNOWN;READY;ERROR
	Status string `json:"status"`
	// ErrorMessage relayed from the backend.
	ErrorMessage string      `json:"errorMessage,omitempty"`
	LastModified metav1.Time `json:"lastModified,omitempty"`
}

// StatefulServerSetObservation are the observable fields of a StatefulServerSet.
type StatefulServerSetObservation struct {
	// Replicas is the count of ready replicas.
	Replicas      int                              `json:"replicas,omitempty"`
	ReplicaStatus []StatefulServerSetReplicaStatus `json:"replicaStatus,omitempty"`
}

// A StatefulServerSetStatus represents the observed state of a StatefulServerSet.
type StatefulServerSetStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          StatefulServerSetObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A StatefulServerSet is an example API type.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,ionoscloud}
type StatefulServerSet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec StatefulServerSetSpec `json:"spec"`
	//TODO: Not sure if the StatefulServerSetStatus should look like this
	Status StatefulServerSetStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// StatefulServerSetList contains a list of StatefulServerSet
type StatefulServerSetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []StatefulServerSet `json:"items"`
}

// StatefulServerSet type metadata.
var (
	StatefulServerSetKind             = reflect.TypeOf(StatefulServerSet{}).Name()
	StatefulServerSetGroupKind        = schema.GroupKind{Group: Group, Kind: StatefulServerSetKind}.String()
	StatefulServerSetKindAPIVersion   = StatefulServerSetKind + "." + SchemeGroupVersion.String()
	StatefulServerSetGroupVersionKind = SchemeGroupVersion.WithKind(StatefulServerSetKind)
)

func init() {
	SchemeBuilder.Register(&StatefulServerSet{}, &StatefulServerSetList{})
}
