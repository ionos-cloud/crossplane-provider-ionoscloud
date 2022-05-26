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

// ForwardingRuleParameters are the observable fields of an ApplicationLoadBalancerForwardingRule.
// Required fields in order to create an ApplicationLoadBalancerForwardingRule:
// DatacenterConfig (via ID or via reference),
// ApplicationLoadBalancerConfig (via ID or via reference),
// Name,
// Protocol,
// ListenerIPConfig (via ID or via reference),
// ListenerPort.
type ForwardingRuleParameters struct {
	// A Datacenter, to which the user has access, to provision
	// the ApplicationLoadBalancer in.
	//
	// +immutable
	// +kubebuilder:validation:Required
	DatacenterCfg DatacenterConfig `json:"datacenterConfig"`
	// An ApplicationLoadBalancer, to which the user has access, to provision
	// the Forwarding Rule in.
	//
	// +immutable
	// +kubebuilder:validation:Required
	ALBCfg ApplicationLoadBalancerConfig `json:"applicationLoadBalancerConfig"`
	// The name of the Application Load Balancer Forwarding Rule.
	//
	// +kubebuilder:validation:Required
	Name string `json:"name"`
	// Balancing protocol
	//
	// +kubebuilder:validation:Enum=HTTP
	// +kubebuilder:validation:Required
	Protocol string `json:"protocol"`
	// Listening (inbound) IP
	//
	// +kubebuilder:validation:Required
	ListenerIP IPConfig `json:"listenerIpConfig"`
	// Listening (inbound) port number; valid range is 1 to 65535.
	//
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// +kubebuilder:validation:Required
	ListenerPort int32 `json:"listenerPort"`
	// The maximum time in milliseconds to wait for the client to acknowledge or send data;
	// default is 50,000 (50 seconds).
	//
	// +kubebuilder:validation:Optional
	ClientTimeout int32 `json:"clientTimeout,omitempty"`
	// Array of items in the collection.
	//
	// +kubebuilder:validation:Optional
	ServerCertificatesIDs []string `json:"serverCertificatesIds,omitempty"`
	// An array of items in the collection. The original order of rules is preserved during processing,
	// except for Forward-type rules are processed after the rules with other action defined.
	// The relative order of Forward-type rules is also preserved during the processing.
	//
	// +kubebuilder:validation:Optional
	HTTPRules []ApplicationLoadBalancerHTTPRule `json:"httpRules,omitempty"`
}

// ApplicationLoadBalancerHTTPRule struct for Application Load Balancer HTTP Rule
// Required fields in order to create an ApplicationLoadBalancerHTTPRule:
// Name,
// Type,
// TargetGroup (via ID or via reference) - required only for FORWARD actions.
// Location - required only for REDIRECT actions.
// ResponseMessage - required only for STATIC actions.
type ApplicationLoadBalancerHTTPRule struct {
	// The unique name of the Application Load Balancer HTTP rule.
	//
	// +kubebuilder:validation:Required
	Name string `json:"name"`
	// Type of the HTTP rule.
	//
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=FORWARD;STATIC;REDIRECT
	Type string `json:"type"`
	// The ID of the target group; mandatory and only valid for FORWARD actions.
	// The ID can be set directly or via reference.
	//
	// +kubebuilder:validation:Optional
	TargetGroupCfg TargetGroupConfig `json:"targetGroupConfig,omitempty"`
	// Default is false; valid only for REDIRECT actions.
	//
	// +kubebuilder:validation:Optional
	DropQuery bool `json:"dropQuery,omitempty"`
	// The location for redirecting; mandatory and valid only for REDIRECT actions.
	// Example: www.ionos.com
	//
	// +kubebuilder:validation:Optional
	Location string `json:"location,omitempty"`
	// Valid only for REDIRECT and STATIC actions.
	// For REDIRECT actions, default is 301 and possible values are 301, 302, 303, 307, and 308.
	// For STATIC actions, default is 503 and valid range is 200 to 599.
	//
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=301;302;303;307;308;200;503;599
	StatusCode int32 `json:"statusCode,omitempty"`
	// The response message of the request; mandatory for STATIC actions.
	//
	// +kubebuilder:validation:Optional
	ResponseMessage string `json:"responseMessage,omitempty"`
	// Valid only for STATIC actions. Example: text/html
	//
	// +kubebuilder:validation:Optional
	ContentType string `json:"contentType,omitempty"`
	// An array of items in the collection.
	// The action is only performed if each and every condition is met; if no conditions are set, the rule will always be performed.
	//
	// +kubebuilder:validation:Optional
	Conditions []ApplicationLoadBalancerHTTPRuleCondition `json:"conditions,omitempty"`
}

// ApplicationLoadBalancerHTTPRuleCondition struct for Application Load Balancer HTTP Rule Condition
// Required fields in order to create an ApplicationLoadBalancerHTTPRuleCondition:
// Type,
// Condition.
type ApplicationLoadBalancerHTTPRuleCondition struct {
	// Type of the HTTP rule condition.
	//
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=HEADER;PATH;QUERY;METHOD;HOST;COOKIE;SOURCE_IP
	Type string `json:"type"`
	// Matching rule for the HTTP rule condition attribute;
	// Mandatory for HEADER, PATH, QUERY, METHOD, HOST, and COOKIE types; Must be null when type is SOURCE_IP.
	//
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=EXISTS;CONTAINS;EQUALS;MATCHES;STARTS_WITH;ENDS_WITH
	Condition string `json:"condition"`
	// Specifies whether the condition is negated or not; the default is False.
	//
	// +kubebuilder:validation:Optional
	Negate bool `json:"negate,omitempty"`
	// Must be null when type is PATH, METHOD, HOST, or SOURCE_IP. Key can only be set when type is COOKIES, HEADER, or QUERY.
	//
	// +kubebuilder:validation:Optional
	Key string `json:"key,omitempty"`
	// Mandatory for conditions CONTAINS, EQUALS, MATCHES, STARTS_WITH, ENDS_WITH;
	// Must be null when condition is EXISTS; should be a valid CIDR if provided and if type is SOURCE_IP.
	//
	// +kubebuilder:validation:Optional
	Value string `json:"value,omitempty"`
}

// ForwardingRuleObservation are the observable fields of an ApplicationLoadBalancerForwardingRule.
type ForwardingRuleObservation struct {
	ForwardingRuleID string `json:"forwardingRuleId,omitempty"`
	ListenerIP       string `json:"listenerIp,omitempty"`
	State            string `json:"state,omitempty"`
}

// ForwardingRuleSpec defines the desired state of an ApplicationLoadBalancer.
type ForwardingRuleSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       ForwardingRuleParameters `json:"forProvider"`
}

// ForwardingRuleStatus represents the observed state of an ApplicationLoadBalancer.
type ForwardingRuleStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          ForwardingRuleObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// An ForwardingRule is an example API type.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="DATACENTER ID",type="string",JSONPath=".spec.forProvider.datacenterConfig.datacenterId"
// +kubebuilder:printcolumn:name="APPLICATIONLOADBALANCER ID",type="string",JSONPath=".spec.forProvider.applicationLoadBalancerConfig.applicationLoadBalancerId"
// +kubebuilder:printcolumn:name="FORWARDINGRULE ID",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="FORWARDINGRULE NAME",type="string",JSONPath=".spec.forProvider.name"
// +kubebuilder:printcolumn:name="PROTOCOL",priority=1,type="string",JSONPath=".spec.forProvider.protocol"
// +kubebuilder:printcolumn:name="LISTENER IP",priority=1,type="string",JSONPath=".status.atProvider.listenerIp"
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

// ForwardingRuleList contains a list of ApplicationLoadBalancer
type ForwardingRuleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ForwardingRule `json:"items"`
}

// ApplicationLoadBalancer type metadata.
var (
	ForwardingRuleKind             = reflect.TypeOf(ForwardingRule{}).Name()
	ForwardingRuleGroupKind        = schema.GroupKind{Group: Group, Kind: ForwardingRuleKind}.String()
	ForwardingRuleKindAPIVersion   = ForwardingRuleKind + "." + SchemeGroupVersion.String()
	ForwardingRuleGroupVersionKind = SchemeGroupVersion.WithKind(ForwardingRuleKind)
)

func init() {
	SchemeBuilder.Register(&ForwardingRule{}, &ForwardingRuleList{})
}
