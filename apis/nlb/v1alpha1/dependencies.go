package v1alpha1

import xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"

// LanConfig is used by resources that need to link lans via id or via reference.
type LanConfig struct {
	// LanID is the ID of the Lan on which the resource will be created.
	// It needs to be provided directly or via reference.
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

// IPBlockConfig used by resources that need to link IPBlocks via id or via reference
type IPBlockConfig struct {
	// IPBlockID is the ID of the IPBlock on which the resource will be created.
	// It needs to be provided directly or via reference.
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
}

// DatacenterConfig is used by resources that need to link datacenters via id or via reference.
type DatacenterConfig struct {
	// DatacenterID is the ID of the Datacenter on which the resource should have access.
	// It needs to be provided directly or via reference.
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
