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

// ForwardingRuleParameters are the observable fields of a Network Load Balancer ForwardingRule.
// Required fields in order to create a Network Load Balancer ForwardingRule:
// DatacenterCfg (via ID or via reference),
// NLBCfg (via ID or via reference),
// Name,
// Protocol,
// ListenerIP (via ID or via reference),
// ListenerPort.
type ForwardingRuleParameters struct {
	// Datacenter in which the Network Load Balancer that this Forwarding Rule applies to is provisioned in.
	//
	// +immutable
	// +kubebuilder:validation:Required
	DatacenterCfg DatacenterConfig `json:"datacenterConfig"`
	// NetworkLoadBalancer to which this Forwarding Rule will apply.
	//
	// +immutable
	// +kubebuilder:validation:Required
	NLBCfg NetworkLoadBalancerConfig `json:"networkLoadBalancerConfig"`
	// The name of the Network Load Balancer Forwarding Rule.
	//
	// +immutable
	// +kubebuilder:validation:Required
	Name string `json:"name"`
	// Balancing protocol
	//
	// +kubebuilder:validation:Enum=TCP;HTTP
	// +kubebuilder:validation:Required
	Protocol string `json:"protocol"`
	// Listening (inbound) IP. IP must be assigned to the listener NIC of the Network Load Balancer.
	//
	// +kubebuilder:validation:Required
	ListenerIP IPConfig `json:"listenerIpConfig"`
	// Listening (inbound) port number; valid range is 1 to 65535.
	//
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// +kubebuilder:validation:Required
	ListenerPort int32 `json:"listenerPort"`
	// Algorithm used in load balancing
	//
	// +kubebuilder:validation:Enum=ROUND_ROBIN;LEAST_CONNECTION;RANDOM;SOURCE_IP
	// +kubebuilder:validation:Required
	Algorithm string `json:"algorithm"`
	// HealthCheck options for the forwarding rule health check
	//
	// +kubebuilder:validation:Optional
	HealthCheck ForwardingRuleHealthCheck `json:"healthCheck,omitempty"`
	// Targets is the list of load balanced targets
	//
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems:1
	Targets []ForwardingRuleTarget `json:"targets"`
}

// ForwardingRuleHealthCheck structure for the forwarding rule health check
type ForwardingRuleHealthCheck struct {
	// ClientTimeout the maximum time in milliseconds to wait for the client to acknowledge or send data; default is 50,000 (50 seconds).
	//
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=50
	ClientTimeout int32 `json:"clientTimeout,omitempty"`
	// ConnectTimeout the maximum time in milliseconds to wait for a connection attempt to a target to succeed; default is 5000 (five seconds).
	//
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=5000
	ConnectTimeout int32 `json:"connectTimeout,omitempty"`
	// TargetTimeout the maximum time in milliseconds that a target can remain inactive; default is 50,000 (50 seconds).
	//
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=50
	TargetTimeout int32 `json:"targetTimeout,omitempty"`
	// Retries the maximum number of attempts to reconnect to a target after a connection failure. Valid range is 0 to 65535 and default is three reconnection attempts.
	//
	// +kubebuilder:validation:Optional
	Retries int32 `json:"retries,omitempty"`
}

// ForwardingRuleTarget structure for the forwarding rule target
// Required fields for the ForwardingRuleTarget:
// IP
// Port
// Weight
type ForwardingRuleTarget struct {
	// IP of the balanced target
	//
	// +kubebuilder:validation:Required
	IPCfg IPConfig `json:"ip"`
	// Port of the balanced target
	//
	// +kubebuilder:validation:Required
	Port int32 `json:"port"`
	// Weight of the balanced target Traffic is distributed in proportion to target weight, relative to the combined weight of all targets.
	// A target with higher weight receives a greater share of traffic. Valid range is 0 to 256 and default is 1.
	// Targets with weight of 0 do not participate in load balancing but still accept persistent connections.
	// It is best to assign weights in the middle of the range to leave room for later adjustments.
	//
	// +kubebuilder:validation:Minimum:0
	// +kubebuilder:validation:Maximum:256
	// +kubebuilder:validation:Required
	Weight int32 `json:"weight"`
	// ProxyProtocol version of the proxy protocol
	//
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=none;v1;v2;v2ssl
	// +kubebuilder:default=none
	ProxyProtocol string `json:"proxyProtocol,omitempty"`
	// HealthCheck options of the balanced target health check
	//
	// +kubebuilder:validation:Optional
	HealthCheck ForwardingRuleTargetHealthCheck `json:"healthCheck,omitempty"`
}

// ForwardingRuleTargetHealthCheck structure for the forwarding rule target health check
type ForwardingRuleTargetHealthCheck struct {
	// Check makes the target available only if it accepts periodic health check TCP connection attempts.
	// When turned off, the target is considered always available.
	// The health check only consists of a connection attempt to the address and port of the target.
	//
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=true
	Check bool `json:"check,omitempty"`
	// CheckInterval the interval in milliseconds between consecutive health checks; default is 2000.
	//
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=2000
	CheckInterval int32 `json:"checkInterval,omitempty"`
	// Maintenance mode prevents the target from receiving balanced traffic.
	//
	// +kubebuilder:validation:Optional
	Maintenance bool `json:"maintenance,omitempty"`
}

// IPConfig used by resources that need to link a single IP directly or by indexing a referenced IPBlock resource
type IPConfig struct {
	// IP can be used to directly specify a single ip to the resource
	//
	// +kubebuilder:validation:Pattern="^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?).){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$"
	IP string `json:"ip,omitempty"`
	// IPBlockConfig can be used to reference an existing IPBlock and assign an ip by indexing
	// For Network Load Balancer Forwarding Rules, only a single index can be specified
	IPBlockConfig `json:"ipBlock,omitempty"`
	// Index can be used to retrieve an ip from the referenced IPBlock
	// Starting index is 0.
	Index uint `json:"index,omitempty"`
}

// ForwardingRuleObservation are the observable fields of a Network Load Balancer ForwardingRule.
type ForwardingRuleObservation struct {
	ForwardingRuleID string `json:"forwardingRuleId,omitempty"`
	State            string `json:"state,omitempty"`
}

// ForwardingRuleSpec defines the desired state of a Network Load Balancer ForwardingRule.
type ForwardingRuleSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       ForwardingRuleParameters `json:"forProvider"`
}

// ForwardingRuleStatus represents the observed state of a Network Load Balancer ForwardingRule.
type ForwardingRuleStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          ForwardingRuleObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// An ForwardingRule is an example API type.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="DATACENTER ID",type="string",JSONPath=".spec.forProvider.datacenterConfig.datacenterId"
// +kubebuilder:printcolumn:name="NETWORKLOADBALANCER ID",type="string",JSONPath=".spec.forProvider.networkLoadBalancerConfig.networkLoadBalancerId"
// +kubebuilder:printcolumn:name="FORWARDINGRULE ID",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="FORWARDINGRULE NAME",type="string",JSONPath=".spec.forProvider.name"
// +kubebuilder:printcolumn:name="PROTOCOL",priority=1,type="string",JSONPath=".spec.forProvider.protocol"
// +kubebuilder:printcolumn:name="LISTENER IP",priority=1,type="string",JSONPath=".spec.forProvider.listenerIp"
// +kubebuilder:printcolumn:name="LISTENER PORT",priority=1,type="string",JSONPath=".spec.forProvider.listenerPort"
// +kubebuilder:printcolumn:name="STATE",type="string",JSONPath=".status.atProvider.state"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,ionoscloud}
type ForwardingRule struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ForwardingRuleSpec   `json:"spec"`
	Status ForwardingRuleStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ForwardingRuleList contains a list of NetworkLoadBalancer
type ForwardingRuleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ForwardingRule `json:"items"`
}

// NetworkLoadBalancer type metadata.
var (
	ForwardingRuleKind             = reflect.TypeOf(ForwardingRule{}).Name()
	ForwardingRuleGroupKind        = schema.GroupKind{Group: Group, Kind: ForwardingRuleKind}.String()
	ForwardingRuleKindAPIVersion   = ForwardingRuleKind + "." + SchemeGroupVersion.String()
	ForwardingRuleGroupVersionKind = SchemeGroupVersion.WithKind(ForwardingRuleKind)
)

func init() {
	SchemeBuilder.Register(&ForwardingRule{}, &ForwardingRuleList{})
}
