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

// DatacenterProperties are the observable fields of a Datacenter.
// Required values when creating a Volume:
// Location.
type DatacenterProperties struct {
	// The name of the  resource.
	Name string `json:"name,omitempty"`
	// A description for the datacenter, such as staging, production.
	Description string `json:"description,omitempty"`
	// The physical location where the datacenter will be created. This will be where all of your servers live.
	// Property cannot be modified after datacenter creation (disallowed in update requests).
	//
	// +immutable
	// +kubebuilder:validation:Enum=de/fra;us/las;us/ewr;de/txl;gb/lhr;es/vit
	// +kubebuilder:validation:Required
	Location string `json:"location"`
	// Boolean value representing if the data center requires extra protection, such as two-step verification.
	SecAuthProtection bool `json:"secAuthProtection,omitempty"`
}

// DatacenterConfig is used by resources that need to link datacenters via id or via reference.
type DatacenterConfig struct {
	// DatacenterID is the ID of the Datacenter on which the resource will be created.
	// It needs to be provided via directly or via reference.
	//
	// +immutable
	// +crossplane:generate:reference:type=Datacenter
	// +crossplane:generate:reference:extractor=ExtractDatacenterID()
	DatacenterID string `json:"datacenterId,omitempty"`
	// DatacenterIDRef references to a Datacenter to retrieve its ID
	//
	// +optional
	// +immutable
	DatacenterIDRef *xpv1.Reference `json:"datacenterIdRef,omitempty"`
	// DatacenterIDSelector selects reference to a Datacenter to retrieve its datacenterId
	//
	// +optional
	DatacenterIDSelector *xpv1.Selector `json:"datacenterIdSelector,omitempty"`
}

// DatacenterObservation are the observable fields of a Datacenter.
type DatacenterObservation struct {
	DatacenterID string `json:"datacenterId,omitempty"`
	State        string `json:"state,omitempty"`
}

// A DatacenterSpec defines the desired state of a Datacenter.
type DatacenterSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       DatacenterProperties `json:"forProvider"`
}

// A DatacenterStatus represents the observed state of a Datacenter.
type DatacenterStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          DatacenterObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A Datacenter is an example API type.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="DATACENTER ID",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="LOCATION",priority=1,type="string",JSONPath=".spec.forProvider.location"
// +kubebuilder:printcolumn:name="NAME",priority=1,type="string",JSONPath=".spec.forProvider.name"
// +kubebuilder:printcolumn:name="STATE",type="string",JSONPath=".status.atProvider.state"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,template}
type Datacenter struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DatacenterSpec   `json:"spec"`
	Status DatacenterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DatacenterList contains a list of Datacenter
type DatacenterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Datacenter `json:"items"`
}

// Datacenter type metadata.
var (
	DatacenterKind             = reflect.TypeOf(Datacenter{}).Name()
	DatacenterGroupKind        = schema.GroupKind{Group: Group, Kind: DatacenterKind}.String()
	DatacenterKindAPIVersion   = DatacenterKind + "." + SchemeGroupVersion.String()
	DatacenterGroupVersionKind = SchemeGroupVersion.WithKind(DatacenterKind)
)

func init() {
	SchemeBuilder.Register(&Datacenter{}, &DatacenterList{})
}
