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

// NodePoolParameters are the observable fields of a NodePool.
// Required fields in order to create a K8s NodePool:
// ClusterConfig,
// Name,
// DatacenterConfig,
// NodeCount,
// CoresCount,
// RAMSize,
// AvailabilityZone,
// StorageType,
// StorageSize.
type NodePoolParameters struct {
	// The K8s Cluster on which the NodePool will be created.
	//
	// +immutable
	// +kubebuilder:validation:Required
	ClusterCfg ClusterConfig `json:"clusterConfig"`
	// A Kubernetes node pool name. Valid Kubernetes node pool name must be 63 characters or less
	// and must be empty or begin and end with an alphanumeric character ([a-z0-9A-Z]) with
	// dashes (-), underscores (_), dots (.), and alphanumerics between.
	//
	// +immutable
	// +kubebuilder:validation:Required
	Name string `json:"name"`
	// A Datacenter, to which the user has access.
	//
	// +immutable
	// +kubebuilder:validation:Required
	DatacenterCfg DatacenterConfig `json:"datacenterConfig"`
	// The number of nodes that make up the node pool.
	//
	// +kubebuilder:validation:Required
	NodeCount int32 `json:"nodeCount"`
	// A valid CPU family name.
	// If no CPUFamily is provided, the API will set a CPUFamily supported by the location.
	//
	// +immutable
	// +kubebuilder:validation:Optional
	CPUFamily string `json:"cpuFamily,omitempty"`
	// The number of cores for the node.
	//
	// +kubebuilder:validation:Required
	CoresCount int32 `json:"coresCount"`
	// The RAM size for the node. Must be set in multiples of 1024 MB, with minimum size is of 2048 MB.
	//
	// +immutable
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MultipleOf=1024
	// +kubebuilder:validation:Minimum=2048
	RAMSize int32 `json:"ramSize"`
	// The availability zone in which the target VM should be provisioned.
	//
	// +immutable
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=AUTO;ZONE_1;ZONE_2
	AvailabilityZone string `json:"availabilityZone"`
	// The type of hardware for the volume.
	//
	// +immutable
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=HDD;SSD
	StorageType string `json:"storageType"`
	// The size of the volume in GB. The size should be greater than 10GB.
	//
	// +immutable
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=10
	StorageSize int32 `json:"storageSize"`
	// The Kubernetes version the NodePool is running. This imposes restrictions on what Kubernetes
	// versions can be run in a cluster's NodePools. Additionally, not all Kubernetes versions are
	// viable upgrade targets for all prior versions.
	//
	// +kubebuilder:validation:Optional
	K8sVersion string `json:"k8sVersion,omitempty"`
	// The maintenance window is used for updating the software on the NodePool's nodes and for upgrading the NodePool's K8s version.
	// If no value is given, one is chosen dynamically, so there is no fixed default.
	//
	// +kubebuilder:validation:Optional
	MaintenanceWindow MaintenanceWindow `json:"maintenanceWindow,omitempty"`
	// property to be set when auto-scaling needs to be enabled for the NodePool.
	// By default, auto-scaling is not enabled.
	//
	// +kubebuilder:validation:Optional
	AutoScaling KubernetesAutoScaling `json:"autoScaling,omitempty"`
	// Array of additional private LANs attached to worker nodes.
	//
	// +kubebuilder:validation:Optional
	Lans []KubernetesNodePoolLan `json:"lans,omitempty"`
	// Map of labels attached to NodePool.
	//
	// +kubebuilder:validation:Optional
	Labels map[string]string `json:"labels,omitempty"`
	// Map of annotations attached to NodePool.
	//
	// +kubebuilder:validation:Optional
	Annotations map[string]string `json:"annotations,omitempty"`
	// Optional array of reserved public IP addresses to be used by the nodes.
	// IPs must be from same location as the Datacenter used for the NodePool.
	// The array must contain one more IP than the maximum possible number of nodes
	// (nodeCount+1 for fixed number of nodes or maxNodeCount+1 when auto-scaling is used).
	// The extra IP is used when the nodes are rebuilt.
	// IPs can be set directly or via reference and indexes.
	//
	// +kubebuilder:validation:Optional
	PublicIPsCfg IPsConfigs `json:"publicIpsConfigs,omitempty"`
	// The type of server for the Kubernetes node pool can be either
	// 'DedicatedCore' (nodes with dedicated CPU cores) or 'VCPU' (nodes with shared CPU cores).
	// This selection corresponds to the server type for the compute engine.
	//
	// +kubebuilder:validation:Optional
	ServerType string `json:"serverType,omitempty"`
}

// KubernetesAutoScaling struct for KubernetesAutoScaling.
type KubernetesAutoScaling struct {
	// The minimum number of worker nodes that the managed node group can scale in.
	// Should be set together with 'maxNodeCount'.
	// Value for this attribute must be greater than equal to 1 and less than equal to maxNodeCount.
	//
	// +kubebuilder:validation:Minimum=1
	MinNodeCount int32 `json:"minNodeCount,omitempty"`
	// The maximum number of worker nodes that the managed node pool can scale-out.
	// Should be set together with 'minNodeCount'.
	// Value for this attribute must be greater than equal to 1 and minNodeCount.
	//
	// +kubebuilder:validation:Minimum=1
	MaxNodeCount int32 `json:"maxNodeCount,omitempty"`
}

// KubernetesNodePoolLan struct for KubernetesNodePoolLan.
type KubernetesNodePoolLan struct {
	// The LAN of an existing private LAN at the related datacenter.
	//
	// +kubebuilder:validation:Optional
	LanCfg LanConfig `json:"lanConfig"`
	// Indicates if the Kubernetes NodePool LAN will reserve an IP using DHCP.
	//
	// +kubebuilder:validation:Optional
	Dhcp bool `json:"dhcp,omitempty"`
	// Array of additional LANs Routes attached to worker nodes.
	//
	// +kubebuilder:validation:Optional
	Routes []KubernetesNodePoolLanRoutes `json:"routes,omitempty"`
	// The datacenter ID, requires system privileges, for internal usage only
	//
	// +kubebuilder:validation:Optional
	DatacenterID string `json:"datacenterID,omitempty"`
}

// KubernetesNodePoolLanRoutes struct for KubernetesNodePoolLanRoutes.
type KubernetesNodePoolLanRoutes struct {
	// IPv4 or IPv6 CIDR to be routed via the interface.
	//
	// +kubebuilder:validation:Optional
	Network string `json:"network,omitempty"`
	// IPv4 or IPv6 Gateway IP for the route.
	//
	// +kubebuilder:validation:Optional
	GatewayIP string `json:"gatewayIp,omitempty"`
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
	// DatacenterIDRef references to a Datacenter to retrieve its ID.
	//
	// +optional
	// +immutable
	DatacenterIDRef *xpv1.Reference `json:"datacenterIdRef,omitempty"`
	// DatacenterIDSelector selects reference to a Datacenter to retrieve its DatacenterID.
	//
	// +optional
	DatacenterIDSelector *xpv1.Selector `json:"datacenterIdSelector,omitempty"`
}

// LanConfig is used by resources that need to link lans via id or via reference.
type LanConfig struct {
	// LanID is the ID of the Lan on which the NodePool will connect to.
	// It needs to be provided via directly or via reference.
	//
	// +immutable
	// +crossplane:generate:reference:type=github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1.Lan
	// +crossplane:generate:reference:extractor=github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1.ExtractLanID()
	LanID string `json:"lanId,omitempty"`
	// LanIDRef references to a Lan to retrieve its ID.
	//
	// +optional
	// +immutable
	LanIDRef *xpv1.Reference `json:"lanIdRef,omitempty"`
	// LanIDSelector selects reference to a Lan to retrieve its LanID.
	//
	// +optional
	LanIDSelector *xpv1.Selector `json:"lanIdSelector,omitempty"`
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
	// +crossplane:generate:reference:type=github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1.IPBlock
	// +crossplane:generate:reference:extractor=github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1.ExtractIPBlockID()
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
	// +crossplane:generate:reference:type=github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1.IPBlock
	// +crossplane:generate:reference:extractor=github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1.ExtractIPBlockID()
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

// NodePoolObservation are the observable fields of a NodePool.
type NodePoolObservation struct {
	NodePoolID               string   `json:"nodePoolId,omitempty"`
	State                    string   `json:"state,omitempty"`
	PublicIPs                []string `json:"publicIps,omitempty"`
	AvailableUpgradeVersions []string `json:"availableUpgradeVersions,omitempty"`
	CPUFamily                string   `json:"cpuFamily,omitempty"`
	NodeCount                *int32   `json:"nodeCount,omitempty"`
	K8sVersion               string   `json:"k8sVersion,omitempty"`
	ServerType               string   `json:"serverType,omitempty"`
}

// A NodePoolSpec defines the desired state of a NodePool.
type NodePoolSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       NodePoolParameters `json:"forProvider"`
}

// A NodePoolStatus represents the observed state of a NodePool.
type NodePoolStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          NodePoolObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A NodePool is an example API type.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="CLUSTER ID",type="string",JSONPath=".spec.forProvider.clusterConfig.clusterId"
// +kubebuilder:printcolumn:name="NODEPOOL ID",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="NODEPOOL NAME",type="string",JSONPath=".spec.forProvider.name"
// +kubebuilder:printcolumn:name="DATACENTER ID",priority=1,type="string",JSONPath=".spec.forProvider.datacenterConfig.datacenterId"
// +kubebuilder:printcolumn:name="K8S VERSION",priority=1,type="string",JSONPath=".status.atProvider.k8sVersion"
// +kubebuilder:printcolumn:name="STATE",type="string",JSONPath=".status.atProvider.state"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,ionoscloud},shortName=k8sn;k8snp
type NodePool struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NodePoolSpec   `json:"spec"`
	Status NodePoolStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// NodePoolList contains a list of NodePool
type NodePoolList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NodePool `json:"items"`
}

// NodePool type metadata.
var (
	NodePoolKind             = reflect.TypeOf(NodePool{}).Name()
	NodePoolGroupKind        = schema.GroupKind{Group: Group, Kind: NodePoolKind}.String()
	NodePoolKindAPIVersion   = NodePoolKind + "." + SchemeGroupVersion.String()
	NodePoolGroupVersionKind = SchemeGroupVersion.WithKind(NodePoolKind)
)

func init() {
	SchemeBuilder.Register(&NodePool{}, &NodePoolList{})
}
