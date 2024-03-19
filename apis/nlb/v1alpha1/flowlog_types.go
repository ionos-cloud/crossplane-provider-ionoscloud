package v1alpha1

import (
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
)

// FlowLogParameters are the observable fields of a Network Load Balancer FlowLog.
// Required fields in order to create a Network Load Balancer FlowLog:
// DatacenterCfg (via ID or via reference),
// NLBCfg (via ID or via reference),
// Name,
// Name,
// Action
// Direction.
// Bucket.
type FlowLogParameters struct {
	// Datacenter in which the Network Load Balancer that this Flow Log applies to is provisioned in.
	//
	// +immutable
	// +kubebuilder:validation:Required
	DatacenterCfg DatacenterConfig `json:"datacenterConfig"`
	// NetworkLoadBalancer to which this Flow Log will apply. There can only be one flow log per Network Load Balancer.
	//
	// +immutable
	// +kubebuilder:validation:Required
	NLBCfg NetworkLoadBalancerConfig `json:"networkLoadBalancerConfig"`
	// Name of the Flow Log.
	//
	// +kubebuilder:validation:Required
	Name string `json:"name"`
	// Action specifies the traffic action pattern, accepted values: ACCEPTED, REJECTED, ALL
	//
	// +kubebuilder:validation:Enum=ACCEPTED;REJECTED;ALL
	// +kubebuilder:validation:Required
	Action string `json:"action"`
	// Direction specifies the traffic action pattern, accepted values: INGRESS, EGRESS, BIDIRECTIONAL
	//
	// +kubebuilder:validation:Enum=INGRESS;EGRESS;BIDIRECTIONAL
	// +kubebuilder:validation:Required
	Direction string `json:"direction"`
	// Bucket name of an existing IONOS Cloud S3 bucket
	//
	// +kubebuilder:validation:Required
	Bucket string `json:"bucket"`
}

// FlowLogObservation are the observable fields of a Network Load Balancer FlowLog.
type FlowLogObservation struct {
	FlowLogID string `json:"flowLogId,omitempty"`
	State     string `json:"state,omitempty"`
}

// FlowLogSpec defines the desired state of a Network Load Balancer FlowLog.
type FlowLogSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       FlowLogParameters `json:"forProvider"`
}

// FlowLogStatus represents the observed state of a Network Load Balancer FlowLog.
type FlowLogStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          FlowLogObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// An FlowLog is an example API type.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="DATACENTER ID",type="string",JSONPath=".spec.forProvider.datacenterConfig.datacenterId"
// +kubebuilder:printcolumn:name="NETWORKLOADBALANCER ID",type="string",JSONPath=".spec.forProvider.networkLoadBalancerConfig.networkLoadBalancerId"
// +kubebuilder:printcolumn:name="FLOWLOG ID",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="FLOWLOG NAME",type="string",JSONPath=".spec.forProvider.name"
// +kubebuilder:printcolumn:name="ACTION",priority=1,type="string",JSONPath=".spec.forProvider.action"
// +kubebuilder:printcolumn:name="DIRECTION",priority=1,type="string",JSONPath=".status.atProvider.direction"
// +kubebuilder:printcolumn:name="STATE",type="string",JSONPath=".status.atProvider.state"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,ionoscloud}
type FlowLog struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FlowLogSpec   `json:"spec"`
	Status FlowLogStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// FlowLogList contains a list of NetworkLoadBalancer
type FlowLogList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FlowLog `json:"items"`
}

// NetworkLoadBalancer type metadata.
var (
	FlowLogKind             = reflect.TypeOf(FlowLog{}).Name()
	FlowLogGroupKind        = schema.GroupKind{Group: Group, Kind: FlowLogKind}.String()
	FlowLogKindAPIVersion   = FlowLogKind + "." + SchemeGroupVersion.String()
	FlowLogGroupVersionKind = SchemeGroupVersion.WithKind(FlowLogKind)
)

func init() {
	SchemeBuilder.Register(&FlowLog{}, &FlowLogList{})
}
