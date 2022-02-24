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
type CubeServerProperties struct {
	// DatacenterConfig contains information about the datacenter resource
	// on which the server will be created
	//
	// +kubebuilder:validation:Required
	DatacenterCfg DatacenterConfig `json:"datacenterConfig"`
	// The ID or the name of the template for creating a CUBE server.
	//
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
}

// Template refers to the template used for cube servers
type Template struct {
	// The name of the  resource.
	Name string `json:"name,omitempty"`
	// The ID of the  template.
	ID string `json:"id,omitempty"`
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
// +kubebuilder:printcolumn:name="ID",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="STATE",type="string",JSONPath=".status.atProvider.state"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,template}
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
	CubeServerGroupKind        = schema.GroupKind{Group: Group, Kind: CubeServerKind}.String()
	CubeServerKindAPIVersion   = CubeServerKind + "." + SchemeGroupVersion.String()
	CubeServerGroupVersionKind = SchemeGroupVersion.WithKind(CubeServerKind)
)

func init() {
	SchemeBuilder.Register(&CubeServer{}, &CubeServerList{})
}
