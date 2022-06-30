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

// FirewallRuleParameters are the observable fields of a FirewallRule.
// Required values when creating a FirewallRule:
// DatacenterConfig,
// ServerConfig,
// NicConfig,
// Protocol.
type FirewallRuleParameters struct {
	// DatacenterConfig contains information about the datacenter resource
	// on which the resource will be created.
	//
	// +kubebuilder:validation:Required
	DatacenterCfg DatacenterConfig `json:"datacenterConfig"`
	// ServerConfig contains information about the server resource
	// on which the resource will be created.
	//
	// +kubebuilder:validation:Required
	ServerCfg ServerConfig `json:"serverConfig"`
	// NicConfig contains information about the nic resource
	// on which the resource will be created.
	//
	// +kubebuilder:validation:Required
	NicCfg NicConfig `json:"nicConfig"`
	// The name of the  resource.
	Name string `json:"name,omitempty"`
	// The protocol for the rule. Property cannot be modified after it is created (disallowed in update requests).
	//
	// +immutable
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=TCP;UDP;ICMP;ANY
	Protocol string `json:"protocol"`
	// Only traffic originating from the respective MAC address is allowed.
	// Valid format: aa:bb:cc:dd:ee:ff. Value null allows traffic from any MAC address.
	//
	// +kubebuilder:validation:Pattern="^([0-9A-Fa-f]{2}[:-]){5}([0-9A-Fa-f]{2})$"
	SourceMac string `json:"sourceMac,omitempty"`
	// Only traffic originating from the respective IPv4 address is allowed.
	// Value null allows traffic from any IP address.
	// SourceIP can be set directly or via reference to an IP Block and index.
	//
	// +kubebuilder:validation:Optional
	SourceIPCfg IPConfig `json:"sourceIpConfig,omitempty"`
	// If the target NIC has multiple IP addresses, only the traffic directed to the respective IP address of the NIC is allowed.
	// Value null allows traffic to any target IP address.
	// TargetIP can be set directly or via reference to an IP Block and index.
	//
	// +kubebuilder:validation:Optional
	TargetIPCfg IPConfig `json:"targetIpConfig,omitempty"`
	// Defines the allowed code (from 0 to 254) if protocol ICMP is chosen. Value null allows all codes.
	//
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=254
	IcmpCode int32 `json:"icmpCode,omitempty"`
	// Defines the allowed type (from 0 to 254) if the protocol ICMP is chosen. Value null allows all types.
	//
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=254
	IcmpType int32 `json:"icmpType,omitempty"`
	// Defines the start range of the allowed port (from 1 to 65534) if protocol TCP or UDP is chosen.
	// Leave portRangeStart and portRangeEnd value null to allow all ports.
	//
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65534
	PortRangeStart int32 `json:"portRangeStart,omitempty"`
	// Defines the end range of the allowed port (from 1 to 65534) if the protocol TCP or UDP is chosen.
	// Leave portRangeStart and portRangeEnd null to allow all ports.
	//
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65534
	PortRangeEnd int32 `json:"portRangeEnd,omitempty"`
	// The type of the firewall rule. If not specified, the default INGRESS value is used.
	//
	// +kubebuilder:validation:Enum=INGRESS;EGRESS
	// +kubebuilder:validation:default=INGRESS
	Type string `json:"type,omitempty"`
}

// FirewallRuleConfig is used by resources that need to link firewallRules via id or via reference.
type FirewallRuleConfig struct {
	// FirewallRuleID is the ID of the FirewallRule on which the resource will be created.
	// It needs to be provided via directly or via reference.
	//
	// +immutable
	// +kubebuilder:validation:Format=uuid
	// +crossplane:generate:reference:type=FirewallRule
	// +crossplane:generate:reference:extractor=ExtractFirewallRuleID()
	FirewallRuleID string `json:"firewallRuleId,omitempty"`
	// FirewallRuleIDRef references to a FirewallRule to retrieve its ID.
	//
	// +optional
	// +immutable
	FirewallRuleIDRef *xpv1.Reference `json:"firewallRuleIdRef,omitempty"`
	// FirewallRuleIDSelector selects reference to a FirewallRule to retrieve its FirewallRuleID.
	//
	// +optional
	FirewallRuleIDSelector *xpv1.Selector `json:"firewallRuleIdSelector,omitempty"`
}

// FirewallRuleObservation are the observable fields of a FirewallRule.
type FirewallRuleObservation struct {
	FirewallRuleID string `json:"firewallRuleId,omitempty"`
	SourceIP       string `json:"sourceIp,omitempty"`
	TargetIP       string `json:"targetIp,omitempty"`
	State          string `json:"state,omitempty"`
}

// A FirewallRuleSpec defines the desired state of a FirewallRule.
type FirewallRuleSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       FirewallRuleParameters `json:"forProvider"`
}

// A FirewallRuleStatus represents the observed state of a FirewallRule.
type FirewallRuleStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          FirewallRuleObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A FirewallRule is an example API type.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="DATACENTER ID",priority=1,type="string",JSONPath=".spec.forProvider.datacenterConfig.datacenterId"
// +kubebuilder:printcolumn:name="SERVER ID",priority=1,type="string",JSONPath=".spec.forProvider.serverConfig.serverId"
// +kubebuilder:printcolumn:name="NIC ID",type="string",JSONPath=".spec.forProvider.nicConfig.nicId"
// +kubebuilder:printcolumn:name="FIREWALLRULE ID",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="STATE",type="string",JSONPath=".status.atProvider.state"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,ionoscloud}
type FirewallRule struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FirewallRuleSpec   `json:"spec"`
	Status FirewallRuleStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// FirewallRuleList contains a list of FirewallRule
type FirewallRuleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FirewallRule `json:"items"`
}

// FirewallRule type metadata.
var (
	FirewallRuleKind             = reflect.TypeOf(FirewallRule{}).Name()
	FirewallRuleGroupKind        = schema.GroupKind{Group: Group, Kind: FirewallRuleKind}.String()
	FirewallRuleKindAPIVersion   = FirewallRuleKind + "." + SchemeGroupVersion.String()
	FirewallRuleGroupVersionKind = SchemeGroupVersion.WithKind(FirewallRuleKind)
)

func init() {
	SchemeBuilder.Register(&FirewallRule{}, &FirewallRuleList{})
}
