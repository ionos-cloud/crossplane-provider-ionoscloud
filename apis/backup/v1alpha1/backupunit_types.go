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

// BackupUnitParameters are the observable fields of an BackupUnit.
// Required fields in order to create an BackupUnit:
// Name,
// Password,
// Email.
type BackupUnitParameters struct {
	// The name of the  resource (alphanumeric characters only).
	//
	// +immutable
	// +kubebuilder:validation:Required
	Name string `json:"name"`
	// The password associated with that resource.
	//
	// +kubebuilder:validation:Required
	Password string `json:"password"`
	// The email associated with the backup unit. Bear in mind that this email does not be the same email as of the user.
	//
	// +kubebuilder:validation:Required
	Email string `json:"email"`
}

// BackupUnitObservation are the observable fields of an BackupUnit.
type BackupUnitObservation struct {
	BackupUnitID string `json:"backupUnitId,omitempty"`
	State        string `json:"state,omitempty"`
}

// BackupUnitSpec defines the desired state of an BackupUnit.
type BackupUnitSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       BackupUnitParameters `json:"forProvider"`
}

// BackupUnitStatus represents the observed state of an BackupUnit.
type BackupUnitStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          BackupUnitObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// An BackupUnit is an example API type.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="BACKUPUNIT ID",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="NAME",type="string",JSONPath=".spec.forProvider.name"
// +kubebuilder:printcolumn:name="EMAIL",type="string",JSONPath=".spec.forProvider.email"
// +kubebuilder:printcolumn:name="STATE",type="string",JSONPath=".status.atProvider.state"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,ionoscloud}
type BackupUnit struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec     BackupUnitSpec   `json:"spec"`
	Status   BackupUnitStatus `json:"status,omitempty"`
	Policies xpv1.ManagementPolicies
}

func (mg *BackupUnit) SetManagementPolicies(p xpv1.ManagementPolicies) {
	mg.Policies = p
}

func (mg *BackupUnit) GetManagementPolicies() xpv1.ManagementPolicies {
	return mg.Policies
}

// +kubebuilder:object:root=true

// BackupUnitList contains a list of BackupUnit
type BackupUnitList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BackupUnit `json:"items"`
}

// BackupUnit type metadata.
var (
	BackupUnitKind             = reflect.TypeOf(BackupUnit{}).Name()
	BackupUnitGroupKind        = schema.GroupKind{Group: Group, Kind: BackupUnitKind}.String()
	BackupUnitKindAPIVersion   = BackupUnitKind + "." + SchemeGroupVersion.String()
	BackupUnitGroupVersionKind = SchemeGroupVersion.WithKind(BackupUnitKind)
)

func init() {
	SchemeBuilder.Register(&BackupUnit{}, &BackupUnitList{})
}
