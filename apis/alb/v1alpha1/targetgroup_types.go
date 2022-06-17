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

// TargetGroupParameters are the observable fields of an TargetGroup.
// Required fields in order to create an TargetGroup:
// Name,
// Algorithm,
// Protocol.
type TargetGroupParameters struct {
	// The name of the target group.
	//
	// +kubebuilder:validation:Required
	Name string `json:"name"`
	// Balancing algorithm
	//
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=ROUND_ROBIN;LEAST_CONNECTION;RANDOM;SOURCE_IP
	Algorithm string `json:"algorithm"`
	// Balancing protocol
	//
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=HTTP
	Protocol string `json:"protocol"`
	// Array of items in the collection.
	//
	// +kubebuilder:validation:Optional
	Targets []TargetGroupTarget `json:"targets,omitempty"`
	// Health check properties for target group
	//
	// +kubebuilder:validation:Optional
	HealthCheck TargetGroupHealthCheck `json:"healthCheck,omitempty"`
	// HTTP health check properties for target group
	//
	// +kubebuilder:validation:Optional
	HTTPHealthCheck TargetGroupHTTPHealthCheck `json:"httpHealthCheck,omitempty"`
}

// TargetGroupTarget struct for TargetGroupTarget
// Required fields in order to create an TargetGroupTarget:
// IPConfig,
// Port,
// Weight.
type TargetGroupTarget struct {
	// The IP of the balanced target VM.
	//
	// +kubebuilder:validation:Required
	IP string `json:"ip"`
	// The port of the balanced target service; valid range is 1 to 65535.
	//
	// +kubebuilder:validation:Required
	Port int32 `json:"port"`
	// Traffic is distributed in proportion to target weight, relative to the combined weight of all targets.
	// A target with higher weight receives a greater share of traffic. Valid range is 0 to 256 and default is 1;
	// targets with weight of 0 do not participate in load balancing but still accept persistent connections.
	// It is best use values in the middle of the range to leave room for later adjustments.
	//
	// +kubebuilder:validation:Required
	Weight int32 `json:"weight"`
	// Makes the target available only if it accepts periodic health check TCP connection attempts;
	// when turned off, the target is considered always available.
	// The health check only consists of a connection attempt to the address and port of the target.
	//
	// +kubebuilder:validation:Optional
	HealthCheckEnabled bool `json:"healthCheckEnabled,omitempty"`
	// Maintenance mode prevents the target from receiving balanced traffic.
	//
	// +kubebuilder:validation:Optional
	MaintenanceEnabled bool `json:"maintenanceEnabled,omitempty"`
}

// TargetGroupHealthCheck struct for TargetGroupHealthCheck
type TargetGroupHealthCheck struct {
	// The maximum time in milliseconds to wait for a target to respond to a check.
	// For target VMs with 'Check Interval' set, the lesser of the two  values
	// is used once the TCP connection is established.
	//
	// +kubebuilder:validation:Optional
	CheckTimeout int32 `json:"checkTimeout,omitempty"`
	// The interval in milliseconds between consecutive health checks; default is 2000.
	//
	// +kubebuilder:validation:Optional
	CheckInterval int32 `json:"checkInterval,omitempty"`
	// The maximum number of attempts to reconnect to a target after a connection failure.
	// Valid range is 0 to 65535, and default is three reconnection attempts.
	//
	// +kubebuilder:validation:Optional
	Retries int32 `json:"retries,omitempty"`
}

// TargetGroupHTTPHealthCheck struct for TargetGroupHttpHealthCheck
// Required fields in order to create an TargetGroupHttpHealthCheck:
// Response,
// MatchType.
type TargetGroupHTTPHealthCheck struct {
	// The path (destination URL) for the HTTP health check request; the default is /.
	//
	// +kubebuilder:validation:Optional
	Path string `json:"path,omitempty"`
	// The method for the HTTP health check.
	//
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=HEAD;PUT;POST;GET;TRACE;PATCH;OPTIONS
	Method string `json:"method,omitempty"`
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum="";STATUS_CODE;RESPONSE_BODY
	MatchType string `json:"matchType"`
	// The response returned by the request, depending on the match type.
	//
	// +kubebuilder:validation:Required
	Response string `json:"response"`
	// +kubebuilder:validation:Optional
	Regex bool `json:"regex,omitempty"`
	// +kubebuilder:validation:Optional
	Negate bool `json:"negate,omitempty"`
}

// TargetGroupConfig is used by resources that need to link application load balancers via id or via reference.
type TargetGroupConfig struct {
	// TargetGroupID is the ID of the TargetGroup on which the resource should have access.
	// It needs to be provided via directly or via reference.
	//
	// +immutable
	// +kubebuilder:validation:Format=uuid
	// +crossplane:generate:reference:type=TargetGroup
	// +crossplane:generate:reference:extractor=ExtractTargetGroupID()
	TargetGroupID string `json:"targetGroupId,omitempty"`
	// TargetGroupIDRef references to a Datacenter to retrieve its ID
	//
	// +optional
	// +immutable
	TargetGroupIDRef *xpv1.Reference `json:"targetGroupIdRef,omitempty"`
	// TargetGroupIDSelector selects reference to a Datacenter to retrieve its datacenterId
	//
	// +optional
	TargetGroupIDSelector *xpv1.Selector `json:"targetGroupIdSelector,omitempty"`
}

// TargetGroupObservation are the observable fields of an TargetGroup.
type TargetGroupObservation struct {
	TargetGroupID string   `json:"targetGroupId,omitempty"`
	TargetIPs     []string `json:"targetIps,omitempty"`
	State         string   `json:"state,omitempty"`
}

// TargetGroupSpec defines the desired state of an TargetGroup.
type TargetGroupSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       TargetGroupParameters `json:"forProvider"`
}

// TargetGroupStatus represents the observed state of an TargetGroup.
type TargetGroupStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          TargetGroupObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// An TargetGroup is an example API type.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="TARGETGROUP ID",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="TARGETGROUP NAME",type="string",JSONPath=".spec.forProvider.name"
// +kubebuilder:printcolumn:name="PROTOCOL",priority=1,type="string",JSONPath=".spec.forProvider.protocol"
// +kubebuilder:printcolumn:name="ALGORITHM",priority=1,type="string",JSONPath=".spec.forProvider.algorithm"
// +kubebuilder:printcolumn:name="TARGET IPS",priority=1,type="string",JSONPath=".status.atProvider.targetIps"
// +kubebuilder:printcolumn:name="STATE",type="string",JSONPath=".status.atProvider.state"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,ionoscloud}
type TargetGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TargetGroupSpec   `json:"spec"`
	Status TargetGroupStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TargetGroupList contains a list of TargetGroup
type TargetGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TargetGroup `json:"items"`
}

// TargetGroup type metadata.
var (
	TargetGroupKind             = reflect.TypeOf(TargetGroup{}).Name()
	TargetGroupGroupKind        = schema.GroupKind{Group: Group, Kind: TargetGroupKind}.String()
	TargetGroupKindAPIVersion   = TargetGroupKind + "." + SchemeGroupVersion.String()
	TargetGroupGroupVersionKind = SchemeGroupVersion.WithKind(TargetGroupKind)
)

func init() {
	SchemeBuilder.Register(&TargetGroup{}, &TargetGroupList{})
}
