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

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// VolumeSelectorParameters are the configurable fields of a Volume.
type VolumeSelectorParameters struct {
	// The number of servers that will be created.
	//
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=1
	Replicas int `json:"replicas"`
	// Name of the serverset on which the volume and server will be
	//
	// +kubebuilder:validation:Required
	ServersetName string `json:"serversetName"`
}

// A VolumeselectorSpec defines the desired state of a Volumeselector.
type VolumeselectorSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       VolumeSelectorParameters `json:"forProvider"`
}

// A VolumeselectorStatus represents the observed state of a Server.
type VolumeselectorStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          VolumeselectorObservation `json:"atProvider,omitempty"`
}

// VolumeselectorObservation are the observable fields of a Server.
type VolumeselectorObservation struct {
	State string `json:"state,omitempty"`
}

// +kubebuilder:object:root=true

// Volumeselector is a managed resource that represents a Volumeselector
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,ionoscloud},shortName=vs;volsel
type Volumeselector struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VolumeselectorSpec   `json:"spec"`
	Status VolumeselectorStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// VolumeselectorList contains a list of Volumeselector
type VolumeselectorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Volumeselector `json:"items"`
}

// Volumeselector type metadata.
var (
	VolumeselectorKind             = reflect.TypeOf(Volumeselector{}).Name()
	VolumeselectorGroupKind        = schema.GroupKind{Group: Group, Kind: VolumeselectorKind}.String()
	VolumeSelectorKindAPIVersion   = VolumeselectorKind + "." + SchemeGroupVersion.String()
	VolumeselectorGroupVersionKind = SchemeGroupVersion.WithKind(VolumeselectorKind)
)

func init() {
	SchemeBuilder.Register(&Volumeselector{}, &VolumeselectorList{})
}
