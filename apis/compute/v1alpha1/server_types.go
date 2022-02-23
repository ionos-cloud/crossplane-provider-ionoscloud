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

// ServerProperties are the observable fields of a Server.
type ServerProperties struct {
	// DatacenterID is the ID of the Datacenter on which the server will be created.
	// It needs to be provided via directly or via reference.
	//
	// +immutable
	// +crossplane:generate:reference:type=Datacenter
	DatacenterID string `json:"datacenterId,omitempty"`
	// DatacenterIDRef references to a Datacenter to retrieve its ID
	//
	// +optional
	DatacenterIDRef *xpv1.Reference `json:"datacenterIdRef,omitempty"`
	// DatacenterIDSelector selects reference to a Datacenter to retrieve its datacenterId
	//
	// +optional
	DatacenterIDSelector *xpv1.Selector `json:"datacenterIdSelector,omitempty"`
	// The name of the  resource.
	Name string `json:"name,omitempty"`
	// The total number of cores for the server.
	//
	// +kubebuilder:validation:Required
	Cores int32 `json:"cores"`
	// The memory size for the server in MB, such as 2048. Size must be specified in multiples of 256 MB with a minimum of 256 MB.
	// however, if you set ramHotPlug to TRUE then you must use a minimum of 1024 MB. If you set the RAM size more than 240GB,
	// then ramHotPlug will be set to FALSE and can not be set to TRUE unless RAM size not set to less than 240GB.
	//
	// +kubebuilder:validation:MultipleOf=256
	// +kubebuilder:validation:Required
	RAM int32 `json:"ram"`
	// The availability zone in which the server should be provisioned.
	//
	// +kubebuilder:validation:Enum=AUTO;ZONE_1;ZONE_2
	AvailabilityZone string `json:"availabilityZone,omitempty"`
	// CPU architecture on which server gets provisioned; not all CPU architectures are available in all datacenter regions;
	// available CPU architectures can be retrieved from the datacenter resource.
	//
	// +kubebuilder:validation:Enum=AMD_OPTERON;INTEL_SKYLAKE;INTEL_XEON
	CPUFamily string `json:"cpuFamily,omitempty"`
}

// ServerObservation are the observable fields of a Server.
type ServerObservation struct {
	ServerID string `json:"serverId,omitempty"`
	State    string `json:"state,omitempty"`
}

// A ServerSpec defines the desired state of a Server.
type ServerSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       ServerProperties `json:"forProvider"`
}

// A ServerStatus represents the observed state of a Server.
type ServerStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          ServerObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A Server is an example API type.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="DATACENTER ID",type="string",JSONPath=".spec.forProvider.datacenterId"
// +kubebuilder:printcolumn:name="ID",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="STATE",type="string",JSONPath=".status.atProvider.state"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,template}
type Server struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ServerSpec   `json:"spec"`
	Status ServerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ServerList contains a list of Server
type ServerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Server `json:"items"`
}

// Server type metadata.
var (
	ServerKind             = reflect.TypeOf(Server{}).Name()
	ServerGroupKind        = schema.GroupKind{Group: Group, Kind: ServerKind}.String()
	ServerKindAPIVersion   = ServerKind + "." + SchemeGroupVersion.String()
	ServerGroupVersionKind = SchemeGroupVersion.WithKind(ServerKind)
)

func init() {
	SchemeBuilder.Register(&Server{}, &ServerList{})
}
