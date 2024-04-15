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

// DeploymentStrategy describes what strategy should be used to deploy the servers.
type DeploymentStrategy struct {
	// +kubebuilder:validation:Enum=ZONES
	Type string `json:"type"`
}

// StatefulServerSetLanMetadata are the configurable fields of a StatefulServerSetLanMetadata.
type StatefulServerSetLanMetadata struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern="[a-z0-9]([-a-z0-9]*[a-z0-9])?"
	// +kubebuilder:validation:MaxLength=63
	Name string `json:"name"`
	// +kubebuilder:validation:Optional
	Labels map[string]string `json:"labels,omitempty"`
}

// StatefulServerSetLanSpec are the configurable fields of a StatefulServerSetLanSpec.
type StatefulServerSetLanSpec struct {
	// +kubebuilder:validation:Optional
	IPv6cidr string `json:"ipv6cidr"`
	// +kubebuilder:validation:Optional
	Public bool `json:"public"`
}

// StatefulServerSetLan are the configurable fields of a StatefulServerSetLan.
type StatefulServerSetLan struct {
	Metadata StatefulServerSetLanMetadata `json:"metadata"`
	Spec     StatefulServerSetLanSpec     `json:"spec"`
}

// StatefulServerSetVolumeMetadata are the configurable fields of a StatefulServerSetVolumeMetadata.
type StatefulServerSetVolumeMetadata struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern="[a-z0-9]([-a-z0-9]*[a-z0-9])?"
	// +kubebuilder:validation:MaxLength=63
	Name string `json:"name"`
	// +kubebuilder:validation:Optional
	Labels map[string]string `json:"labels,omitempty"`
}

// StatefulServerSetVolumeSpec are the configurable fields of a StatefulServerSetVolumeSpec.
type StatefulServerSetVolumeSpec struct {
	// The public image UUID or a public image alias.
	//
	// +kubebuilder:validation:Optional
	Image string `json:"image,omitempty"`
	// The size of the volume in GB.
	//
	// +kubebuilder:validation:Required
	Size float32 `json:"size"`
	// Hardware type of the volume. E.g: HDD;SSD;SSD Standard;SSD Premium
	//
	// +immutable
	// +kubebuilder:validation:Enum=HDD;SSD;SSD Standard;SSD Premium
	// +kubebuilder:validation:Required
	// +kubebuilder:example=SSD
	Type string `json:"type"`
	// The cloud init configuration in base64 encoding.
	UserData string `json:"userData,omitempty"`
}

// StatefulServerSetVolume are the configurable fields of a StatefulServerSetVolume.
type StatefulServerSetVolume struct {
	Metadata StatefulServerSetVolumeMetadata `json:"metadata"`
	Spec     StatefulServerSetVolumeSpec     `json:"spec"`
}

// StatefulServerSetParameters are the configurable fields of a StatefulServerSet.
type StatefulServerSetParameters struct {
	// The number of servers that will be created. Cannot be decreased once set, only increased. Has a minimum of 1.
	//
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:XValidation:rule="self >= oldSelf", message="Replicas can only be increased"
	Replicas           int                `json:"replicas"`
	DeploymentStrategy DeploymentStrategy `json:"deploymentStrategy"`
	// DatacenterConfig contains information about the datacenter resource
	// on which the server will be created.
	//
	// +kubebuilder:validation:Required
	DatacenterCfg      DatacenterConfig          `json:"datacenterConfig"`
	Template           ServerSetTemplate         `json:"template"`
	BootVolumeTemplate BootVolumeTemplate        `json:"bootVolumeTemplate"`
	Lans               []StatefulServerSetLan    `json:"lans"`
	Volumes            []StatefulServerSetVolume `json:"volumes"`
}

// A StatefulServerSetSpec defines the desired state of a StatefulServerSet.
type StatefulServerSetSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       StatefulServerSetParameters `json:"forProvider"`
}

// StatefulServerSetReplicaStatus are the configurable fields of a StatefulServerSetReplicaStatus.
type StatefulServerSetReplicaStatus struct {
	// Server assigned role
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
	xpv1.ResourceStatus `json:",inline"`
	// Replicas is the count of ready replicas.
	Replicas           int                      `json:"replicas,omitempty"`
	ReplicaStatus      []ServerSetReplicaStatus `json:"replicaStatus,omitempty"`
	DataVolumeStatuses []VolumeStatus           `json:"dataVolumeStatus,omitempty"`
	LanStatuses        []LanStatus              `json:"lanStatus,omitempty"`
}

// A StatefulServerSetStatus represents the observed state of a StatefulServerSet.
type StatefulServerSetStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          StatefulServerSetObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A StatefulServerSet is an example API type.
// +kubebuilder:printcolumn:name="Datacenter ID",type="string",JSONPath=".spec.forProvider.datacenterConfig.datacenterId"
// +kubebuilder:printcolumn:name="REPLICAS",type="integer",JSONPath=".status.atProvider.replicas"
// +kubebuilder:printcolumn:name="servers",priority=1,type="string",JSONPath=".status.atProvider.replicaStatus"
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,ionoscloud},shortName=sss;ssset
// +kubebuilder:subresource:scale:specpath=.spec.forProvider.replicas,statuspath=.status.atProvider.replicas,selectorpath=.status.selector
type StatefulServerSet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   StatefulServerSetSpec   `json:"spec"`
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
