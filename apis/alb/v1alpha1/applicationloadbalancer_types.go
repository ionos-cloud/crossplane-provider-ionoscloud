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
	// A Datacenter, to which the user has access.
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
	//
	// +optional
	// +kubebuilder:validation:Optional
	IpsCfg IpsConfig `json:"ipsConfig,omitempty"`
	// Collection of private IP addresses with the subnet mask of the Application Load Balancer.
	// IPs must contain valid a subnet mask.
	// If no IP is provided, the system will generate an IP with /24 subnet.
	//
	// +optional
	// +kubebuilder:validation:Optional
	LbPrivateIps []string `json:"lbPrivateIps,omitempty"`
}

// DatacenterConfig is used by resources that need to link datacenters via id or via reference.
type DatacenterConfig struct {
	// DatacenterID is the ID of the Datacenter on which the resource should have access.
	// It needs to be provided via directly or via reference.
	//
	// +immutable
	// +kubebuilder:validation:Format=uuid
	// +crossplane:generate:reference:type=github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1.Datacenter
	// +crossplane:generate:reference:extractor=github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1.ExtractDatacenterID()
	DatacenterID string `json:"datacenterId,omitempty"`
	// DatacenterIDRef references to a Datacenter to retrieve its ID
	//
	// +optional
	// +immutable
	DatacenterIDRef *xpv1.Reference `json:"datacenterIdRef,omitempty"`
	// DatacenterIDSelector selects reference to a Datacenter to retrieve its datacenterId
	//
	// +optional
	DatacenterIDSelector *xpv1.Selector `json:"datacenterIdSelector,omitempty"`
}

// LanConfig is used by resources that need to link lans via id or via reference.
type LanConfig struct {
	// LanID is the ID of the Lan on which the resource will be created.
	// It needs to be provided via directly or via reference.
	//
	// +immutable
	// +crossplane:generate:reference:type=github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1.Lan
	// +crossplane:generate:reference:extractor=github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1.ExtractLanID()
	LanID string `json:"lanId,omitempty"`
	// LanIDRef references to a Lan to retrieve its ID
	//
	// +optional
	// +immutable
	LanIDRef *xpv1.Reference `json:"lanIdRef,omitempty"`
	// LanIDSelector selects reference to a Lan to retrieve its lanId
	//
	// +optional
	LanIDSelector *xpv1.Selector `json:"lanIdSelector,omitempty"`
}

// IpsConfig is used by resources that need to link ips from ipblock via id or via reference
// and using index. If no index is set, all IPs from the corresponding IPBlock will be assigned.
// If both ips and ipblockConfigs fields will be set, the IPs assigned will be a sum of the two.
type IpsConfig struct {
	Ips         []string        `json:"ips,omitempty"`
	IPBlockCfgs []IPBlockConfig `json:"ipblockConfigs,omitempty"`
}

// IPBlockConfig is used by resources that need to link ipblock via id or via reference.
type IPBlockConfig struct {
	// NicID is the ID of the IPBlock on which the resource will be created.
	// It needs to be provided via directly or via reference.
	//
	// +immutable
	// +kubebuilder:validation:Format=uuid
	// +crossplane:generate:reference:type=github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1.IPBlock
	// +crossplane:generate:reference:extractor=github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1.ExtractIPBlockID()
	IPBlockID string `json:"ipblockId,omitempty"`
	// IPBlockIDRef references to a IPBlock to retrieve its ID
	//
	// +optional
	// +immutable
	IPBlockIDRef *xpv1.Reference `json:"ipblockIdRef,omitempty"`
	// IPBlockIDSelector selects reference to a IPBlock to retrieve its nicId
	//
	// +optional
	IPBlockIDSelector *xpv1.Selector `json:"ipblockIdSelector,omitempty"`
	// Indexes are referring to the IPs indexes retrieved from the IPBlock.
	//
	// +optional
	Indexes []int `json:"indexes,omitempty"`
}

// ApplicationLoadBalancerObservation are the observable fields of an ApplicationLoadBalancer.
type ApplicationLoadBalancerObservation struct {
	ApplicationLoadBalancerID string   `json:"applicationLoadBalancerId,omitempty"`
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
// +kubebuilder:printcolumn:name="IPS",priority=1,type="string",JSONPath=".spec.forProvider.ipsConfig.ips"
// +kubebuilder:printcolumn:name="LB PRIVATE IPS",priority=1,type="string",JSONPath=".spec.forProvider.lbPrivateIps"
// +kubebuilder:printcolumn:name="STATE",type="string",JSONPath=".status.atProvider.state"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,ionoscloud}
type ApplicationLoadBalancer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ApplicationLoadBalancerSpec   `json:"spec"`
	Status ApplicationLoadBalancerStatus `json:"status,omitempty"`
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
