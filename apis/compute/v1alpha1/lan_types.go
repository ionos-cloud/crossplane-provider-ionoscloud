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

const (
	// LANAuto is used on a LAN's Ipv6Cidr to indicate that an IPV6 address block will be assigned
	// automatically by Cloud API. See LanParameters.Ipv6Cidr
	LANAuto = "AUTO"
)

// LanParameters are the observable fields of a Lan.
// Required values when creating a Lan:
// DatacenterConfig,
// Public.
type LanParameters struct {
	// DatacenterConfig contains information about the datacenter resource
	// on which the lan will be created.
	//
	// +kubebuilder:validation:Required
	DatacenterCfg DatacenterConfig `json:"datacenterConfig"`
	// The name of the  resource.
	//
	// +kubebuilder:validation:Optional
	Name string `json:"name,omitempty"`
	// The unique identifier of the private Cross-Connect the LAN is connected to, if any.
	//
	// +kubebuilder:validation:Optional
	Pcc PccConfig `json:"pcc,omitempty"`
	// This LAN faces the public Internet.
	//
	// +kubebuilder:validation:Required
	Public bool `json:"public"`
	// For a GET request, this value is either 'null' or contains the LAN's /64 IPv6 CIDR block if this LAN is IPv6-enabled. For POST/PUT/PATCH requests, 'AUTO' will result in enabling this
	// LAN for IPv6 and automatically assign a /64 IPv6 CIDR block to this LAN. If you choose the IPv6 CIDR block on your own, then you must provide a /64 block,
	// which is inside the IPv6 CIDR block of the virtual datacenter and unique inside all LANs from this virtual datacenter. If you enable IPv6 on a LAN with NICs,
	// those NICs will get a /80 IPv6 CIDR block and one IPv6 address assigned to each automatically, unless you specify them explicitly on the NICs.
	// A virtual data center is limited to a maximum of 256 IPv6-enabled LANs.
	//
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=""
	Ipv6Cidr string `json:"ipv6Cidr,omitempty"`
}

// LanConfig is used by resources that need to link lans via id or via reference.
type LanConfig struct {
	// LanID is the ID of the Lan on which the resource will be created.
	// It needs to be provided via directly or via reference.
	//
	// +immutable
	// +crossplane:generate:reference:type=Lan
	// +crossplane:generate:reference:extractor=ExtractLanID()
	LanID string `json:"lanId,omitempty"`
	// LanIDRef references to a Lan to retrieve its ID.
	//
	// +optional
	// +immutable
	LanIDRef *xpv1.Reference `json:"lanIdRef,omitempty"`
	// LanIDSelector selects reference to a Lan to retrieve its LanID.
	//
	// +optional
	LanIDSelector *xpv1.Selector `json:"lanIdSelector,omitempty"`
}

// LanObservation are the observable fields of a Lan.
type LanObservation struct {
	// IPFailovers will be set using IPFailover CR
	IPFailovers []string `json:"ipFailovers,omitempty"`
	LanID       string   `json:"lanId,omitempty"`
	State       string   `json:"state,omitempty"`
	Name        string   `json:"name,omitempty"`
	Ipv4Cidr    string   `json:"ipv4Cidr,omitempty"`
}

// A LanSpec defines the desired state of a Lan.
type LanSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       LanParameters `json:"forProvider"`
}

// A LanStatus represents the observed state of a Lan.
type LanStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          LanObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A Lan is an example API type.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="DATACENTER ID",type="string",JSONPath=".spec.forProvider.datacenterConfig.datacenterId"
// +kubebuilder:printcolumn:name="LAN ID",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="LAN NAME",type="string",JSONPath=".spec.forProvider.name"
// +kubebuilder:printcolumn:name="PUBLIC",priority=1,type="string",JSONPath=".spec.forProvider.public"
// +kubebuilder:printcolumn:name="IPFAILOVERS",priority=1,type="string",JSONPath=".status.atProvider.ipFailovers"
// +kubebuilder:printcolumn:name="IPV4CIDR",priority=1,type="string",JSONPath=".status.atProvider.ipv4Cidr"
// +kubebuilder:printcolumn:name="STATE",type="string",JSONPath=".status.atProvider.state"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,ionoscloud}
type Lan struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LanSpec   `json:"spec"`
	Status LanStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// LanList contains a list of Lan
type LanList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Lan `json:"items"`
}

// Lan type metadata.
var (
	LanKind             = reflect.TypeOf(Lan{}).Name()
	LanGroupKind        = schema.GroupKind{Group: APIGroup, Kind: LanKind}.String()
	LanKindAPIVersion   = LanKind + "." + SchemeGroupVersion.String()
	LanGroupVersionKind = SchemeGroupVersion.WithKind(LanKind)
)

func init() {
	SchemeBuilder.Register(&Lan{}, &LanList{})
}
