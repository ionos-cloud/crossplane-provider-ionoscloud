package ipblock

import (
	"context"
	"fmt"
	"reflect"

	sdkgo "github.com/ionos-cloud/sdk-go/v6"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"
)

// APIClient is a wrapper around IONOS Service
type APIClient struct {
	*clients.IonosServices
}

// Client is a wrapper around IONOS Service IPBlock methods
type Client interface {
	CheckDuplicateIPBlock(ctx context.Context, ipBlockName, location string) (*sdkgo.IpBlock, error)
	GetIPBlockID(ipBlock *sdkgo.IpBlock) (string, error)
	GetIPBlock(ctx context.Context, ipBlockID string) (sdkgo.IpBlock, *sdkgo.APIResponse, error)
	CreateIPBlock(ctx context.Context, ipBlock sdkgo.IpBlock) (sdkgo.IpBlock, *sdkgo.APIResponse, error)
	UpdateIPBlock(ctx context.Context, ipBlockID string, ipBlock sdkgo.IpBlockProperties) (sdkgo.IpBlock, *sdkgo.APIResponse, error)
	DeleteIPBlock(ctx context.Context, ipBlockID string) (*sdkgo.APIResponse, error)
	GetIPs(ctx context.Context, ipBlockID string, indexes ...int) ([]string, error)
	GetAPIClient() *sdkgo.APIClient
}

// CheckDuplicateIPBlock based on ipBlockName, and the immutable property location
func (cp *APIClient) CheckDuplicateIPBlock(ctx context.Context, ipBlockName, location string) (*sdkgo.IpBlock, error) { // nolint: gocyclo
	ipBlocks, _, err := cp.ComputeClient.IPBlocksApi.IpblocksGet(ctx).Depth(utils.DepthQueryParam).Execute()
	if err != nil {
		return nil, err
	}
	matchedItems := make([]sdkgo.IpBlock, 0)
	if itemsOk, ok := ipBlocks.GetItemsOk(); ok && itemsOk != nil {
		for _, item := range *itemsOk {
			if propertiesOk, ok := item.GetPropertiesOk(); ok && propertiesOk != nil {
				if nameOk, ok := propertiesOk.GetNameOk(); ok && nameOk != nil {
					if *nameOk == ipBlockName {
						// After checking the name, check the immutable properties
						if locationOk, ok := propertiesOk.GetLocationOk(); ok && locationOk != nil {
							if *locationOk == location {
								matchedItems = append(matchedItems, item)
							} else {
								return nil, fmt.Errorf("error: found ipblock with the name %v, but immutable property location different. expected: %v actual: %v", ipBlockName, location, *locationOk)
							}
						}
					}
				}
			}
		}
	}
	if len(matchedItems) == 0 {
		return nil, nil
	}
	if len(matchedItems) > 1 {
		return nil, fmt.Errorf("error: found multiple ipblocks with the name %v", ipBlockName)
	}
	return &matchedItems[0], nil
}

// GetIPBlockID based on ipBlock
func (cp *APIClient) GetIPBlockID(ipBlock *sdkgo.IpBlock) (string, error) {
	if ipBlock != nil {
		if idOk, ok := ipBlock.GetIdOk(); ok && idOk != nil {
			return *idOk, nil
		}
		return "", fmt.Errorf("error: getting ipblock id")
	}
	return "", nil
}

// GetIPBlock based on ipBlockID
func (cp *APIClient) GetIPBlock(ctx context.Context, ipBlockID string) (sdkgo.IpBlock, *sdkgo.APIResponse, error) {
	return cp.ComputeClient.IPBlocksApi.IpblocksFindById(ctx, ipBlockID).Depth(utils.DepthQueryParam).Execute()
}

// CreateIPBlock based on IPBlock properties
func (cp *APIClient) CreateIPBlock(ctx context.Context, ipBlock sdkgo.IpBlock) (sdkgo.IpBlock, *sdkgo.APIResponse, error) {
	return cp.ComputeClient.IPBlocksApi.IpblocksPost(ctx).Ipblock(ipBlock).Execute()
}

// UpdateIPBlock based on ipBlockID and IPBlock properties
func (cp *APIClient) UpdateIPBlock(ctx context.Context, ipBlockID string, ipBlock sdkgo.IpBlockProperties) (sdkgo.IpBlock, *sdkgo.APIResponse, error) {
	return cp.ComputeClient.IPBlocksApi.IpblocksPatch(ctx, ipBlockID).Ipblock(ipBlock).Execute()
}

// DeleteIPBlock based on ipBlockID
func (cp *APIClient) DeleteIPBlock(ctx context.Context, ipBlockID string) (*sdkgo.APIResponse, error) {
	resp, err := cp.ComputeClient.IPBlocksApi.IpblocksDelete(ctx, ipBlockID).Execute()
	return resp, err
}

// GetIPs based on ipBlockID and indexes (optional argument).
// If indexes (0-indexes) are not set, all IPs will be returned.
func (cp *APIClient) GetIPs(ctx context.Context, ipBlockID string, indexes ...int) ([]string, error) {
	ipBlockIds := make([]string, 0)
	ipBlock, _, err := cp.ComputeClient.IPBlocksApi.IpblocksFindById(ctx, ipBlockID).Depth(utils.DepthQueryParam).Execute()
	if err != nil {
		return nil, err
	}
	if properties, ok := ipBlock.GetPropertiesOk(); ok && properties != nil {
		if ipsOk, ok := properties.GetIpsOk(); ok && ipsOk != nil {
			ips := *ipsOk
			if len(indexes) == 0 {
				ipBlockIds = append(ipBlockIds, ips...)
			}
			for _, index := range indexes {
				if index >= len(ips) {
					return ipBlockIds, fmt.Errorf("error: index out of range. it must be less than %v", len(ips))
				}
				ipBlockIds = append(ipBlockIds, ips[index])
			}
			return ipBlockIds, nil
		}
		return nil, fmt.Errorf("error: getting ips from ipblock properties: %v", ipBlockID)
	}
	return nil, fmt.Errorf("error: getting properties from ipblock: %v", ipBlockID)
}

// GetAPIClient gets the APIClient
func (cp *APIClient) GetAPIClient() *sdkgo.APIClient {
	return cp.ComputeClient
}

// GenerateCreateIPBlockInput returns IpBlock based on the CR spec
func GenerateCreateIPBlockInput(cr *v1alpha1.IPBlock) (*sdkgo.IpBlock, error) {
	instanceCreateInput := sdkgo.IpBlock{
		Properties: &sdkgo.IpBlockProperties{
			Location: &cr.Spec.ForProvider.Location,
			Size:     &cr.Spec.ForProvider.Size,
		},
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.Name)) {
		instanceCreateInput.Properties.SetName(cr.Spec.ForProvider.Name)
	}
	return &instanceCreateInput, nil
}

// GenerateUpdateIPBlockInput returns IpBlockProperties based on the CR spec modifications
func GenerateUpdateIPBlockInput(cr *v1alpha1.IPBlock) (*sdkgo.IpBlockProperties, error) {
	instanceUpdateInput := sdkgo.IpBlockProperties{
		Name: &cr.Spec.ForProvider.Name,
	}
	return &instanceUpdateInput, nil
}

// LateStatusInitializer fills the empty fields in *v1alpha1.IPBlockObservation with
// the values seen in sdkgo.IpBlockProperties.
func LateStatusInitializer(in *v1alpha1.IPBlockObservation, sg *sdkgo.IpBlock) {
	if sg == nil {
		return
	}
	// Add IPs to the Status
	if propertiesOk, ok := sg.GetPropertiesOk(); ok && propertiesOk != nil {
		if ipsOk, ok := propertiesOk.GetIpsOk(); ok && ipsOk != nil {
			in.Ips = *ipsOk
		}
	}
}

// IsIPBlockUpToDate returns true if the IPBlock is up-to-date or false if it does not
func IsIPBlockUpToDate(cr *v1alpha1.IPBlock, ipBlock sdkgo.IpBlock) bool { // nolint:gocyclo
	switch {
	case cr == nil && ipBlock.Properties == nil:
		return true
	case cr == nil && ipBlock.Properties != nil:
		return false
	case cr != nil && ipBlock.Properties == nil:
		return false
	case ipBlock.Metadata.State != nil && *ipBlock.Metadata.State == "BUSY":
		return true
	case ipBlock.Properties.Name != nil && *ipBlock.Properties.Name != cr.Spec.ForProvider.Name:
		return false
	default:
		return true
	}
}
