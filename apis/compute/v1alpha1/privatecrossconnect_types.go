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

// PrivateCrossConnectParameters To connect two PrivateCrossconnects we need 2 lans defined, one in each PrivateCrossConnect.
// After, we reference the pcc through which we want the connection to be established.
type PrivateCrossConnectParameters struct {
	// The name of the private cross-connection.
	Name string `json:"name,omitempty"`
	// A short description for the private cross-connection.
	Description string `json:"description,omitempty"`
}

// PrivateCrossConnectConfig is used by resources that need to link a Private Cross Connect via id or via reference.
type PrivateCrossConnectConfig struct {
	// PrivateCrossConnectID is the ID of the PrivateCrossConnect on which the resource will be created.
	// It needs to be provided via directly or via reference.
	//
	// +immutable
	// +kubebuilder:validation:Format=uuid
	// +crossplane:generate:reference:type=PrivateCrossConnect
	// +crossplane:generate:reference:extractor=ExtractPrivateCrossConnectID()
	PrivateCrossConnectID string `json:"PrivateCrossConnectId,omitempty"`
	// PrivateCrossConnectIDRef references to a PrivateCrossConnect to retrieve its ID.
	//
	// +optional
	// +immutable
	PrivateCrossConnectIDRef *xpv1.Reference `json:"PrivateCrossConnectIdRef,omitempty"`
	// PrivateCrossConnectIDSelector selects reference to a PrivateCrossConnect to retrieve its PrivateCrossConnectID.
	//
	// +optional
	PrivateCrossConnectIDSelector *xpv1.Selector `json:"PrivateCrossConnectIdSelector,omitempty"`
}

// PrivateCrossConnectObservation are the observable fields of a PrivateCrossConnect.
type PrivateCrossConnectObservation struct {
	PrivateCrossConnectID string `json:"PrivateCrossConnectId,omitempty"`
	State                 string `json:"state,omitempty"`
}

// A PrivateCrossConnectSpec defines the desired state of a PrivateCrossConnect.
type PrivateCrossConnectSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       PrivateCrossConnectParameters `json:"forProvider"`
}

// A PrivateCrossConnectStatus represents the observed state of a PrivateCrossConnect.
type PrivateCrossConnectStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          PrivateCrossConnectObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A PrivateCrossConnect is an example API type.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="PrivateCrossConnect ID",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="PrivateCrossConnect NAME",type="string",JSONPath=".spec.forProvider.name"
// +kubebuilder:printcolumn:name="STATE",type="string",JSONPath=".status.atProvider.state"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,ionoscloud}
type PrivateCrossConnect struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec               PrivateCrossConnectSpec   `json:"spec"`
	Status             PrivateCrossConnectStatus `json:"status,omitempty"`
	ManagementPolicies xpv1.ManagementPolicies   `json:"managementPolicies"`
}

// SetManagementPolicies implement managed interface
func (mg *PrivateCrossConnect) SetManagementPolicies(p xpv1.ManagementPolicies) {
	mg.ManagementPolicies = p
}

// GetManagementPolicies implement managed interface
func (mg *PrivateCrossConnect) GetManagementPolicies() xpv1.ManagementPolicies {
	return mg.ManagementPolicies
}

// +kubebuilder:object:root=true

// PrivateCrossConnectList contains a list of PrivateCrossConnect
type PrivateCrossConnectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PrivateCrossConnect `json:"items"`
}

// PrivateCrossConnect type metadata.
var (
	PrivateCrossConnectKind             = reflect.TypeOf(PrivateCrossConnect{}).Name()
	PrivateCrossConnectGroupKind        = schema.GroupKind{Group: Group, Kind: PrivateCrossConnectKind}.String()
	PrivateCrossConnectKindAPIVersion   = PrivateCrossConnectKind + "." + SchemeGroupVersion.String()
	PrivateCrossConnectGroupVersionKind = SchemeGroupVersion.WithKind(PrivateCrossConnectKind)
)

func init() {
	SchemeBuilder.Register(&PrivateCrossConnect{}, &PrivateCrossConnectList{})
}
