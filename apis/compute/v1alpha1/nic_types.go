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

// NicParameters are the observable fields of a Nic.
// Required values when creating a Nic:
// Datacenter ID or Reference,
// Server ID or Reference,
// Lan ID or Reference,
// DHCP.
type NicParameters struct {
	// DatacenterConfig contains information about the datacenter resource
	// on which the nic will be created.
	//
	// +kubebuilder:validation:Required
	DatacenterCfg DatacenterConfig `json:"datacenterConfig"`
	// ServerConfig contains information about the server resource
	// on which the nic will be created.
	//
	// +kubebuilder:validation:Required
	ServerCfg ServerConfig `json:"serverConfig"`
	// LanConfig contains information about the lan resource
	// on which the nic will be on.
	//
	// +kubebuilder:validation:Required
	LanCfg LanConfig `json:"lanConfig"`
	// The name of the  resource.
	//
	// +kubebuilder:validation:Optional
	Name string `json:"name,omitempty"`
	// Collection of IP addresses, assigned to the NIC.
	// Explicitly assigned public IPs need to come from reserved IP blocks.
	// Passing value null or empty array will assign an IP address automatically.
	// The IPs can be set directly or using reference to the existing IPBlocks and indexes.
	// If no indexes are set, all IPs from the corresponding IPBlock will be assigned.
	// All IPs set on the Nic will be displayed on the status's ips field.
	//
	// +kubebuilder:validation:Optional
	IpsCfg IPsConfigs `json:"ipsConfigs,omitempty"`
	// Indicates if the NIC will reserve an IP using DHCP.
	//
	// +kubebuilder:validation:Required
	Dhcp bool `json:"dhcp"`
	// Activate or deactivate the firewall. By default, an active firewall without any defined rules
	// will block all incoming network traffic except for the firewall rules that explicitly allows certain protocols, IP addresses and ports.
	//
	// +kubebuilder:validation:Optional
	FirewallActive bool `json:"firewallActive,omitempty"`
	// The type of firewall rules that will be allowed on the NIC. If not specified, the default INGRESS value is used.
	//
	// +kubebuilder:validation:Enum=BIDIRECTIONAL;EGRESS;INGRESS
	// +kubebuilder:validation:Optional
	FirewallType string `json:"firewallType,omitempty"`

	// The vnet ID that belongs to this NIC. Requires system privileges
	//
	// +kubebuilder:validation:Optional
	Vnet string `json:"vnet,omitempty"`
}

// NicConfig is used by resources that need to link nic via id or via reference.
type NicConfig struct {
	// NicID is the ID of the Nic on which the resource will be created.
	// It needs to be provided via directly or via reference.
	//
	// +immutable
	// +kubebuilder:validation:Format=uuid
	// +crossplane:generate:reference:type=Nic
	// +crossplane:generate:reference:extractor=ExtractNicID()
	NicID string `json:"nicId,omitempty"`
	// NicIDRef references to a Nic to retrieve its ID.
	//
	// +optional
	// +immutable
	NicIDRef *xpv1.Reference `json:"nicIdRef,omitempty"`
	// NicIDSelector selects reference to a Nic to retrieve its NicID.
	//
	// +optional
	NicIDSelector *xpv1.Selector `json:"nicIdSelector,omitempty"`
}

// NicObservation are the observable fields of a Nic.
type NicObservation struct {
	NicID    string   `json:"nicId,omitempty"`
	VolumeID string   `json:"volumeId,omitempty"`
	IPs      []string `json:"ips,omitempty"`
	State    string   `json:"state,omitempty"`
	Mac      string   `json:"mac,omitempty"`
	PCISlot  int32    `json:"pciSlot,omitempty"`
}

// A NicSpec defines the desired state of a Nic.
type NicSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       NicParameters `json:"forProvider"`
}

// A NicStatus represents the observed state of a Nic.
type NicStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          NicObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A Nic is an example API type.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="DATACENTER ID",type="string",JSONPath=".spec.forProvider.datacenterConfig.datacenterId"
// +kubebuilder:printcolumn:name="SERVER ID",type="string",JSONPath=".spec.forProvider.serverConfig.serverId"
// +kubebuilder:printcolumn:name="LAN ID",type="string",JSONPath=".spec.forProvider.lanConfig.lanId"
// +kubebuilder:printcolumn:name="NIC ID",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="IPS",priority=1,type="string",JSONPath=".status.atProvider.ips"
// +kubebuilder:printcolumn:name="STATE",type="string",JSONPath=".status.atProvider.state"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="PCISlot",type="string",JSONPath=".status.atProvider.pciSlot"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,ionoscloud}
type Nic struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec               NicSpec                 `json:"spec"`
	Status             NicStatus               `json:"status,omitempty"`
	ManagementPolicies xpv1.ManagementPolicies `json:"managementPolicies"`
}

// SetManagementPolicies implement managed interface
func (mg *Nic) SetManagementPolicies(p xpv1.ManagementPolicies) {
	mg.ManagementPolicies = p
}

// GetManagementPolicies implement managed interface
func (mg *Nic) GetManagementPolicies() xpv1.ManagementPolicies {
	return mg.ManagementPolicies
}

// +kubebuilder:object:root=true

// NicList contains a list of Nic
type NicList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Nic `json:"items"`
}

// Nic type metadata.
var (
	NicKind             = reflect.TypeOf(Nic{}).Name()
	NicGroupKind        = schema.GroupKind{Group: Group, Kind: NicKind}.String()
	NicKindAPIVersion   = NicKind + "." + SchemeGroupVersion.String()
	NicGroupVersionKind = SchemeGroupVersion.WithKind(NicKind)
)

func init() {
	SchemeBuilder.Register(&Nic{}, &NicList{})
}
