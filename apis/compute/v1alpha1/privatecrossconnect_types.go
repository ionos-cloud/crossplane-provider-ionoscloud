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

// PccParameters To connect two PrivateCrossconnects we need 2 lans defined, one in each Pcc.
// After, we reference the Pcc through which we want the connection to be established.
type PccParameters struct {
	// The name of the private cross-connection.
	Name string `json:"name,omitempty"`
	// A short description for the private cross-connection.
	Description string `json:"description,omitempty"`
}

// PccConfig is used by resources that need to link a Private Cross Connect via id or via reference.
type PccConfig struct {
	// PrivateCrossConnectID is the ID of the Pcc on which the resource will be created.
	// It needs to be provided via directly or via reference.
	//
	// +immutable
	// +kubebuilder:validation:Format=uuid
	// +crossplane:generate:reference:type=Pcc
	// +crossplane:generate:reference:extractor=ExtractPccID()
	PrivateCrossConnectID string `json:"PrivateCrossConnectId,omitempty"`
	// PrivateCrossConnectIDRef references to a Pcc to retrieve its ID.
	//
	// +optional
	// +immutable
	PrivateCrossConnectIDRef *xpv1.Reference `json:"PrivateCrossConnectIdRef,omitempty"`
	// PrivateCrossConnectIDSelector selects reference to a Pcc to retrieve its PrivateCrossConnectID.
	//
	// +optional
	PrivateCrossConnectIDSelector *xpv1.Selector `json:"PrivateCrossConnectIdSelector,omitempty"`
}

// PccObservation are the observable fields of a Pcc.
type PccObservation struct {
	PrivateCrossConnectID string `json:"PrivateCrossConnectId,omitempty"`
	State                 string `json:"state,omitempty"`
}

// A PccSpec defines the desired state of a Pcc.
type PccSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       PccParameters `json:"forProvider"`
}

// A PccStatus represents the observed state of a Pcc.
type PccStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          PccObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A Pcc is an example API type.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="Pcc ID",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="Pcc NAME",type="string",JSONPath=".spec.forProvider.name"
// +kubebuilder:printcolumn:name="STATE",type="string",JSONPath=".status.atProvider.state"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,ionoscloud}
type Pcc struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec               PccSpec                 `json:"spec"`
	Status             PccStatus               `json:"status,omitempty"`
	ManagementPolicies xpv1.ManagementPolicies `json:"managementPolicies"`
}

// SetManagementPolicies implement managed interface
func (mg *Pcc) SetManagementPolicies(p xpv1.ManagementPolicies) {
	mg.ManagementPolicies = p
}

// GetManagementPolicies implement managed interface
func (mg *Pcc) GetManagementPolicies() xpv1.ManagementPolicies {
	return mg.ManagementPolicies
}

// +kubebuilder:object:root=true

// PccList contains a list of Pcc
type PccList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Pcc `json:"items"`
}

// Pcc type metadata.
var (
	PrivateCrossConnectKind             = reflect.TypeOf(Pcc{}).Name()
	PrivateCrossConnectGroupKind        = schema.GroupKind{Group: Group, Kind: PrivateCrossConnectKind}.String()
	PrivateCrossConnectKindAPIVersion   = PrivateCrossConnectKind + "." + SchemeGroupVersion.String()
	PrivateCrossConnectGroupVersionKind = SchemeGroupVersion.WithKind(PrivateCrossConnectKind)
)

func init() {
	SchemeBuilder.Register(&Pcc{}, &PccList{})
}
