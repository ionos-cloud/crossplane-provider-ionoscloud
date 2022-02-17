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
type DatacenterProperties struct {
	// The name of the  resource.
	Name string `json:"name,omitempty"`
	// A description for the datacenter, such as staging, production.
	Description string `json:"description,omitempty"`
	// The physical location where the datacenter will be created. This will be where all of your servers live. Property cannot be modified after datacenter creation (disallowed in update requests).
	Location string `json:"location"`
	// The version of the data center; incremented with every change.
	Version int32 `json:"version,omitempty"`
	// Boolean value representing if the data center requires extra protection, such as two-step verification.
	SecAuthProtection bool `json:"secAuthProtection,omitempty"`
}

// DatacenterObservation are the observable fields of a Datacenter.
type DatacenterObservation struct {
	DatacenterID string `json:"datacenterID,omitempty"`
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
// +kubebuilder:printcolumn:name="ID",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
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
