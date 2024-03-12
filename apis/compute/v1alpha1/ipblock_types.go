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

// IPBlockParameters are the observable fields of a IPBlock.
// Required values when creating an IPBlock:
// Location,
// Size.
type IPBlockParameters struct {
	// The name of the  resource.
	Name string `json:"name,omitempty"`
	// Location of that IP block. Property cannot be modified after it is created (disallowed in update requests).
	// Location can have the following values: de/fra, us/las, us/ewr, de/txl, gb/lhr, es/vit.
	//
	// +immutable
	// +kubebuilder:validation:Required
	Location string `json:"location"`
	// The size of the IP block.
	//
	// +immutable
	// +kubebuilder:validation:Required
	Size int32 `json:"size"`
}

// IPsConfigs - used by resources that need to link multiple IPs directly or from IPBlock via id or via reference.
type IPsConfigs struct {
	// Use IPs to set specific IPs to the resource. If both IPs and IPsBlockConfigs are set,
	// only `ips` field will be considered.
	IPs []string `json:"ips,omitempty"`
	// Use IpsBlockConfigs to reference existing IPBlocks, and to mention the indexes for the IPs.
	// Indexes start from 0, and multiple indexes can be set. If no index is set, all IPs from the
	// corresponding IPBlock will be assigned to the resource.
	IPBlockCfgs []IPsBlockConfig `json:"ipsBlockConfigs,omitempty"`
}

// IPsBlockConfig - used by resources that need to link IPBlock via id or via reference
// to get multiple IPs.
type IPsBlockConfig struct {
	// IPBlockID is the ID of the IPBlock on which the resource will be created.
	// It needs to be provided via directly or via reference.
	//
	// +immutable
	// +kubebuilder:validation:Format=uuid
	// +crossplane:generate:reference:type=IPBlock
	// +crossplane:generate:reference:extractor=ExtractIPBlockID()
	IPBlockID string `json:"ipBlockId,omitempty"`
	// IPBlockIDRef references to a IPBlock to retrieve its ID.
	//
	// +optional
	// +immutable
	IPBlockIDRef *xpv1.Reference `json:"ipBlockIdRef,omitempty"`
	// IPBlockIDSelector selects reference to a IPBlock to retrieve its IPBlockID.
	//
	// +optional
	IPBlockIDSelector *xpv1.Selector `json:"ipBlockIdSelector,omitempty"`
	// Indexes are referring to the IPs indexes retrieved from the IPBlock.
	// Indexes are starting from 0. If no index is set, all IPs from the
	// corresponding IPBlock will be assigned.
	//
	// +optional
	Indexes []int `json:"indexes,omitempty"`
}

// IPConfig is used by resources that need to link ip directly or from IPBlock via id or via reference.
type IPConfig struct {
	// Use IP to set specific IP to the resource. If both IP and IPBlockConfig are set,
	// only `ip` field will be considered.
	//
	// +kubebuilder:validation:Pattern="^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?).){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$"
	IP string `json:"ip,omitempty"`
	// Use IpBlockConfig to reference existing IPBlock, and to mention the index for the IP.
	// Index starts from 0 and it must be provided.
	IPBlockCfg IPBlockConfig `json:"ipBlockConfig,omitempty"`
}

// IPBlockConfig - used by resources that need to link IPBlock via id or via reference
// to get one single IP.
type IPBlockConfig struct {
	// IPBlockID is the ID of the IPBlock on which the resource will be created.
	// It needs to be provided via directly or via reference.
	//
	// +immutable
	// +kubebuilder:validation:Format=uuid
	// +crossplane:generate:reference:type=IPBlock
	// +crossplane:generate:reference:extractor=ExtractIPBlockID()
	IPBlockID string `json:"ipBlockId,omitempty"`
	// IPBlockIDRef references to a IPBlock to retrieve its ID.
	//
	// +optional
	// +immutable
	IPBlockIDRef *xpv1.Reference `json:"ipBlockIdRef,omitempty"`
	// IPBlockIDSelector selects reference to a IPBlock to retrieve its IPBlockID.
	//
	// +optional
	IPBlockIDSelector *xpv1.Selector `json:"ipBlockIdSelector,omitempty"`
	// Index is referring to the IP index retrieved from the IPBlock.
	// Index is starting from 0.
	//
	// +kubebuilder:validation:Required
	Index int `json:"index"`
}

// IPBlockObservation are the observable fields of a IPBlock.
type IPBlockObservation struct {
	IPBlockID string   `json:"ipBlockId,omitempty"`
	State     string   `json:"state,omitempty"`
	Ips       []string `json:"ips,omitempty"`
}

// A IPBlockSpec defines the desired state of a IPBlock.
type IPBlockSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       IPBlockParameters `json:"forProvider"`
}

// A IPBlockStatus represents the observed state of a IPBlock.
type IPBlockStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          IPBlockObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A IPBlock is an example API type.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="IPBLOCK ID",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="IPS",type="string",JSONPath=".status.atProvider.ips"
// +kubebuilder:printcolumn:name="NAME",priority=1,type="string",JSONPath=".spec.forProvider.name"
// +kubebuilder:printcolumn:name="LOCATION",type="string",JSONPath=".spec.forProvider.location"
// +kubebuilder:printcolumn:name="STATE",type="string",JSONPath=".status.atProvider.state"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,ionoscloud}
type IPBlock struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IPBlockSpec   `json:"spec"`
	Status IPBlockStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// IPBlockList contains a list of IPBlock
type IPBlockList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IPBlock `json:"items"`
}

// IPBlock type metadata.
var (
	IPBlockKind             = reflect.TypeOf(IPBlock{}).Name()
	IPBlockGroupKind        = schema.GroupKind{Group: APIGroup, Kind: IPBlockKind}.String()
	IPBlockKindAPIVersion   = IPBlockKind + "." + SchemeGroupVersion.String()
	IPBlockGroupVersionKind = SchemeGroupVersion.WithKind(IPBlockKind)
)

func init() {
	SchemeBuilder.Register(&IPBlock{}, &IPBlockList{})
}
