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

// NetworkLoadBalancerParameters are the observable fields of a NetworkLoadBalancer
// Required fields in order to create a NetworkLoadBalancer:
// DatacenterCfg (via ID or via reference),
// Name,
// ListenerLanCfg (via ID or via reference),
// TargetLanCfg (via ID or via reference).
type NetworkLoadBalancerParameters struct {
	// A Datacenter, to which the user has access, to provision the Network Load Balancer in.
	//
	// +immutable
	// +kubebuilder:validation:Required
	DatacenterCfg DatacenterConfig `json:"datacenterConfig"`
	// The name of the Network Load Balancer.
	//
	// +kubebuilder:validation:Required
	Name string `json:"name"`
	// ID of the listening (inbound) LAN.
	// Lan ID can be set directly or via reference.
	//
	// +kubebuilder:validation:Required
	ListenerLanCfg LanConfig `json:"listenerLanConfig"`
	// ID of the balanced private target (outbound) LAN .
	// Lan ID can be set directly or via reference.
	//
	// +kubebuilder:validation:Required
	TargetLanCfg LanConfig `json:"targetLanConfig"`
	// Collection of the Network Load Balancer IP addresses.
	// (Inbound and outbound) IPs of the listenerLan are customer-reserved public IPs for
	// the public Load Balancers, and private IPs for the private Load Balancers.
	// The IPs can be set directly or using reference to the existing IPBlocks and indexes.
	//
	// +kubebuilder:validation:Optional
	IpsCfg IPsConfig `json:"ipsConfig,omitempty"`
	// Collection of private IP addresses with the subnet mask of the Network Load Balancer.
	// IPs must contain valid a subnet mask.
	// If no IP is provided, the system will generate an IP with /24 subnet.
	//
	// +kubebuilder:validation:Optional
	LbPrivateIps []string `json:"lbPrivateIps,omitempty"`
}

// IPsConfig used by resources that need to link multiple IPs directly or by indexing referenced IPBlock resources
type IPsConfig struct {
	// IPs can be used to directly specify a list of ips to the resource
	IPs []string `json:"ips,omitempty"`
	// IPBlocks can be used to reference existing IPBlocks and assign ips by indexing
	IPsBlocksCfg []IPsBlockConfig `json:"ipsBlocksConfig,omitempty"`
}

// IPsBlockConfig used to specify an IPBlock together with an Indexes string to select multiple IPs
type IPsBlockConfig struct {
	// IPBlock  used to reference an existing IPBlock
	IPBlock IPBlockConfig `json:"ipBlockConfig,omitempty"`
	// Indexes can be used to retrieve multiple ips from an IPBlock
	// Starting index is 0. If no index is set, the entire IP set of the block will be assigned.
	Indexes []int `json:"indexes,omitempty"`
}

// NetworkLoadBalancerConfig is used by resources that need to link Network Load Balancers via id or via reference
type NetworkLoadBalancerConfig struct {
	// NetworkLoadBalancerID is the ID of the NetworkLoadBalancer on which the resource should have access.
	// It needs to be provided directly or via reference.
	//
	// +immutable
	// +kubebuilder:validation:Format=uuid
	// +crossplane:generate:reference:type=NetworkLoadBalancer
	// +crossplane:generate:reference:extractor=ExtractNetworkLoadBalancerID()
	NetworkLoadBalancerID string `json:"networkLoadBalancerId,omitempty"`
	// NetworkLoadBalancerIDRef references to a Datacenter to retrieve its ID.
	//
	// +optional
	// +immutable
	NetworkLoadBalancerIDRef *xpv1.Reference `json:"networkLoadBalancerIdRef,omitempty"`
	// NetworkLoadBalancerIDSelector selects reference to a Datacenter to retrieve its DatacenterID.
	//
	// +optional
	NetworkLoadBalancerIDSelector *xpv1.Selector `json:"networkLoadBalancerIdSelector,omitempty"`
}

// NetworkLoadBalancerObservation are the observable fields of an NetworkLoadBalancer.
type NetworkLoadBalancerObservation struct {
	NetworkLoadBalancerID string `json:"networkLoadBalancerId,omitempty"`
	State                 string `json:"state,omitempty"`
	ListenerIPs           string `json:"listenerIps,omitempty"`
	PrivateIPs            string `json:"privateIps,omitempty"`
}

// NetworkLoadBalancerSpec defines the desired state of an NetworkLoadBalancer.
type NetworkLoadBalancerSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       NetworkLoadBalancerParameters `json:"forProvider"`
}

// NetworkLoadBalancerStatus represents the observed state of an NetworkLoadBalancer.
type NetworkLoadBalancerStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          NetworkLoadBalancerObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A NetworkLoadBalancer is an example API type.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="DATACENTER ID",type="string",JSONPath=".spec.forProvider.datacenterConfig.datacenterId"
// +kubebuilder:printcolumn:name="NETWORKLOADBALANCER ID",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="NETWORKLOADBALANCER NAME",type="string",JSONPath=".spec.forProvider.name"
// +kubebuilder:printcolumn:name="LISTENER LAN",type="string",JSONPath=".spec.forProvider.listenerLanConfig.lanId"
// +kubebuilder:printcolumn:name="TARGET LAN",type="string",JSONPath=".spec.forProvider.targetLanConfig.lanId"
// +kubebuilder:printcolumn:name="LISTENER IPS",type="string",JSONPath=".status.atProvider.listenerIps"
// +kubebuilder:printcolumn:name="PRIVATE IPS",type="string",JSONPath=".status.atProvider.privateIps"
// +kubebuilder:printcolumn:name="STATE",type="string",JSONPath=".status.atProvider.state"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,ionoscloud},shortName=nlb;networklb
type NetworkLoadBalancer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NetworkLoadBalancerSpec   `json:"spec"`
	Status NetworkLoadBalancerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// NetworkLoadBalancerList contains a list of NetworkLoadBalancer
type NetworkLoadBalancerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NetworkLoadBalancer `json:"items"`
}

// NetworkLoadBalancer type metadata
var (
	NetworkLoadBalancerKind             = reflect.TypeOf(NetworkLoadBalancer{}).Name()
	NetworkLoadBalancerGroupKind        = schema.GroupKind{Group: Group, Kind: NetworkLoadBalancerKind}.String()
	NetworkLoadBalancerKindAPIVersion   = NetworkLoadBalancerKind + "." + SchemeGroupVersion.String()
	NetworkLoadBalancerGroupVersionKind = SchemeGroupVersion.WithKind(NetworkLoadBalancerKind)
)

func init() {
	SchemeBuilder.Register(&NetworkLoadBalancer{}, &NetworkLoadBalancerList{})
}
