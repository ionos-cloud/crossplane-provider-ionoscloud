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

// ApplicationLoadBalancerParameters are the observable fields of an ApplicationLoadBalancer.
// Required fields in order to create an ApplicationLoadBalancer:
// DatacenterConfig (via ID or via reference),
// Name,
// ListenerLanConfig (via ID or via reference),
// TargetLanConfig (via ID or via reference).
type ApplicationLoadBalancerParameters struct {
	// A Datacenter, to which the user has access, to provision
	// the ApplicationLoadBalancer in.
	//
	// +immutable
	// +kubebuilder:validation:Required
	DatacenterCfg DatacenterConfig `json:"datacenterConfig"`
	// The name of the Application Load Balancer.
	//
	// +kubebuilder:validation:Required
	Name string `json:"name"`
	// ID of the listening (inbound) LAN.
	// Lan ID can be set directly or via reference.
	//
	// +kubebuilder:validation:Required
	ListenerLanCfg LanConfig `json:"listenerLanConfig"`
	// ID of the balanced private target LAN (outbound).
	// Lan ID can be set directly or via reference.
	//
	// +kubebuilder:validation:Required
	TargetLanCfg LanConfig `json:"targetLanConfig"`
	// Collection of the Application Load Balancer IP addresses.
	// (Inbound and outbound) IPs of the listenerLan are customer-reserved public IPs for
	// the public Load Balancers, and private IPs for the private Load Balancers.
	// The IPs can be set directly or using reference to the existing IPBlocks and indexes.
	// If no indexes are set, all IPs from the corresponding IPBlock will be assigned.
	// All IPs set on the Nic will be displayed on the status's ips field.
	//
	// +optional
	// +kubebuilder:validation:Optional
	IpsCfg IPsConfigs `json:"ipsConfig,omitempty"`
	// Collection of private IP addresses with the subnet mask of the Application Load Balancer.
	// IPs must contain valid a subnet mask.
	// If no IP is provided, the system will generate an IP with /24 subnet.
	//
	// +optional
	// +kubebuilder:validation:Optional
	LbPrivateIps []string `json:"lbPrivateIps,omitempty"`
}

// ApplicationLoadBalancerConfig is used by resources that need to link application load balancers via id or via reference.
type ApplicationLoadBalancerConfig struct {
	// ApplicationLoadBalancerID is the ID of the ApplicationLoadBalancer on which the resource should have access.
	// It needs to be provided via directly or via reference.
	//
	// +immutable
	// +kubebuilder:validation:Format=uuid
	// +crossplane:generate:reference:type=ApplicationLoadBalancer
	// +crossplane:generate:reference:extractor=ExtractApplicationLoadBalancerID()
	ApplicationLoadBalancerID string `json:"applicationLoadBalancerId,omitempty"`
	// ApplicationLoadBalancerIDRef references to a Datacenter to retrieve its ID.
	//
	// +optional
	// +immutable
	ApplicationLoadBalancerIDRef *xpv1.Reference `json:"applicationLoadBalancerIdRef,omitempty"`
	// ApplicationLoadBalancerIDSelector selects reference to a Datacenter to retrieve its DatacenterID.
	//
	// +optional
	ApplicationLoadBalancerIDSelector *xpv1.Selector `json:"applicationLoadBalancerIdSelector,omitempty"`
}

// ApplicationLoadBalancerObservation are the observable fields of an ApplicationLoadBalancer.
type ApplicationLoadBalancerObservation struct {
	ApplicationLoadBalancerID string   `json:"applicationLoadBalancerId,omitempty"`
	PublicIPs                 []string `json:"publicIps,omitempty"`
	State                     string   `json:"state,omitempty"`
	AvailableUpgradeVersions  []string `json:"availableUpgradeVersions,omitempty"`
	ViableNodePoolVersions    []string `json:"viableNodePoolVersions,omitempty"`
}

// ApplicationLoadBalancerSpec defines the desired state of an ApplicationLoadBalancer.
type ApplicationLoadBalancerSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       ApplicationLoadBalancerParameters `json:"forProvider"`
}

// ApplicationLoadBalancerStatus represents the observed state of an ApplicationLoadBalancer.
type ApplicationLoadBalancerStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          ApplicationLoadBalancerObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// An ApplicationLoadBalancer is an example API type.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="DATACENTER ID",type="string",JSONPath=".spec.forProvider.datacenterConfig.datacenterId"
// +kubebuilder:printcolumn:name="APPLICATIONLOADBALANCER ID",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="APPLICATIONLOADBALANCER NAME",type="string",JSONPath=".spec.forProvider.name"
// +kubebuilder:printcolumn:name="LISTENER LAN",priority=1,type="string",JSONPath=".spec.forProvider.listenerLanConfig.lanId"
// +kubebuilder:printcolumn:name="TARGET LAN",priority=1,type="string",JSONPath=".spec.forProvider.targetLanConfig.lanId"
// +kubebuilder:printcolumn:name="IPS",priority=1,type="string",JSONPath=".status.atProvider.publicIps"
// +kubebuilder:printcolumn:name="STATE",type="string",JSONPath=".status.atProvider.state"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,ionoscloud}
type ApplicationLoadBalancer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec     ApplicationLoadBalancerSpec   `json:"spec"`
	Status   ApplicationLoadBalancerStatus `json:"status,omitempty"`
	Policies xpv1.ManagementPolicies
}

func (mg *ApplicationLoadBalancer) SetManagementPolicies(p xpv1.ManagementPolicies) {
	mg.Policies = p
}

func (mg *ApplicationLoadBalancer) GetManagementPolicies() xpv1.ManagementPolicies {
	return mg.Policies
}

// +kubebuilder:object:root=true

// ApplicationLoadBalancerList contains a list of ApplicationLoadBalancer
type ApplicationLoadBalancerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ApplicationLoadBalancer `json:"items"`
}

// ApplicationLoadBalancer type metadata.
var (
	ApplicationLoadBalancerKind             = reflect.TypeOf(ApplicationLoadBalancer{}).Name()
	ApplicationLoadBalancerGroupKind        = schema.GroupKind{Group: Group, Kind: ApplicationLoadBalancerKind}.String()
	ApplicationLoadBalancerKindAPIVersion   = ApplicationLoadBalancerKind + "." + SchemeGroupVersion.String()
	ApplicationLoadBalancerGroupVersionKind = SchemeGroupVersion.WithKind(ApplicationLoadBalancerKind)
)

func init() {
	SchemeBuilder.Register(&ApplicationLoadBalancer{}, &ApplicationLoadBalancerList{})
}
