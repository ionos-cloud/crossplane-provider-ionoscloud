package v1alpha1

import xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"

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
	// LanID is the ID of the Lan on which the resource will be created.
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
	// Index starts from 0.
	//
	// +kubebuilder:validation:Required
	Index int `json:"index"`
}
