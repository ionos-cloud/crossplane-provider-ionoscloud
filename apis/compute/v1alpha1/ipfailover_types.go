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

// IPFailoverParameters are the observable fields of a IPFailover.
// Required values when creating a IPFailover:
// DatacenterConfig,
// LanConfig,
// NicConfig,
// IP.
type IPFailoverParameters struct {
	// DatacenterConfig contains information about the datacenter resource
	// on which the resource will be created
	//
	// +kubebuilder:validation:Required
	DatacenterCfg DatacenterConfig `json:"datacenterConfig"`
	// LanConfig contains information about the lan resource
	// on which the resource will be created
	//
	// +kubebuilder:validation:Required
	LanCfg LanConfig `json:"lanConfig"`
	// NicConfig contains information about the nic resource
	// on which the resource will be created
	//
	// +kubebuilder:validation:Required
	NicCfg NicConfig `json:"nicConfig"`
	// IP must be the public IP for which the group is responsible for
	//
	// +kubebuilder:validation:Required
	IP string `json:"ip"`
}

// IPFailoverObservation are the observable fields of a IPFailover.
type IPFailoverObservation struct {
	IPFailovers []string `json:"ipFailovers,omitempty"`
	State       string   `json:"state,omitempty"`
}

// A IPFailoverSpec defines the desired state of a IPFailover.
type IPFailoverSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       IPFailoverParameters `json:"forProvider"`
}

// A IPFailoverStatus represents the observed state of a IPFailover.
type IPFailoverStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          IPFailoverObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A IPFailover is an example API type.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="DATACENTER ID",type="string",JSONPath=".spec.forProvider.datacenterConfig.datacenterId"
// +kubebuilder:printcolumn:name="LAN ID",type="string",JSONPath=".spec.forProvider.lanConfig.lanId"
// +kubebuilder:printcolumn:name="NIC ID",type="string",JSONPath=".spec.forProvider.nicConfig.nicId"
// +kubebuilder:printcolumn:name="IP",type="string",JSONPath=".spec.forProvider.ip"
// +kubebuilder:printcolumn:name="LAN IPFAILOVERS",priority=1,type="string",JSONPath=".status.atProvider.ipFailovers"
// +kubebuilder:printcolumn:name="STATE",type="string",JSONPath=".status.atProvider.state"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,ionoscloud}
type IPFailover struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IPFailoverSpec   `json:"spec"`
	Status IPFailoverStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// IPFailoverList contains a list of IPFailover
type IPFailoverList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IPFailover `json:"items"`
}

// IPFailover type metadata.
var (
	IPFailoverKind             = reflect.TypeOf(IPFailover{}).Name()
	IPFailoverGroupKind        = schema.GroupKind{Group: Group, Kind: IPFailoverKind}.String()
	IPFailoverKindAPIVersion   = IPFailoverKind + "." + SchemeGroupVersion.String()
	IPFailoverGroupVersionKind = SchemeGroupVersion.WithKind(IPFailoverKind)
)

func init() {
	SchemeBuilder.Register(&IPFailover{}, &IPFailoverList{})
}
