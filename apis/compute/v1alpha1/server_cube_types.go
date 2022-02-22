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
	// +immutable
	DatacenterID string `json:"datacenterID,omitempty"`
	// DatacenterRef references to a Datacenter to retrieve its name
	// +optional
	DatacenterIDRef *xpv1.Reference `json:"datacenterIDRef,omitempty"`
	// The ID of the template for creating a CUBE server; the available templates for CUBE servers can be found on the templates' resource.
	TemplateID string `json:"templateID"`
	// The name of the  resource.
	Name string `json:"name,omitempty"`
	// The availability zone in which the server should be provisioned.
	AvailabilityZone string `json:"availabilityZone,omitempty"`
	// CPU architecture on which server gets provisioned; not all CPU architectures are available in all datacenter regions;
	// available CPU architectures can be retrieved from the datacenter resource.
	CPUFamily string `json:"cpuFamily,omitempty"`
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
	CubeServerKindAPIVersion   = ServerKind + "." + SchemeGroupVersion.String()
	CubeServerGroupVersionKind = SchemeGroupVersion.WithKind(ServerKind)
)

func init() {
	SchemeBuilder.Register(&CubeServer{}, &CubeServerList{})
}
