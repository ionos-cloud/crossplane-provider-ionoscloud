package ipblock

import (
	"context"
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
	GetIPBlock(ctx context.Context, ipBlockID string) (sdkgo.IpBlock, *sdkgo.APIResponse, error)
	CreateIPBlock(ctx context.Context, ipBlock sdkgo.IpBlock) (sdkgo.IpBlock, *sdkgo.APIResponse, error)
	UpdateIPBlock(ctx context.Context, ipBlockID string, ipBlock sdkgo.IpBlockProperties) (sdkgo.IpBlock, *sdkgo.APIResponse, error)
	DeleteIPBlock(ctx context.Context, ipBlockID string) (*sdkgo.APIResponse, error)
	GetIPs(ctx context.Context, ipBlockID string, indexes ...int) ([]string, error)
	GetAPIClient() *sdkgo.APIClient
}

// GetIPBlock based on ipBlockID
func (cp *APIClient) GetIPBlock(ctx context.Context, ipBlockID string) (sdkgo.IpBlock, *sdkgo.APIResponse, error) {
	return cp.ComputeClient.IPBlocksApi.IpblocksFindById(ctx, ipBlockID).Depth(utils.DepthQueryParam).Execute()
}

// GetIPs based on ipBlockID and indexes. Indexes is an optional arg, if it is not provided, all IPs will be returned.
func (cp *APIClient) GetIPs(ctx context.Context, ipBlockID string, indexes ...int) ([]string, error) {
	ipblockIds := make([]string, 0)
	ipblock, _, err := cp.ComputeClient.IPBlocksApi.IpblocksFindById(ctx, ipBlockID).Depth(utils.DepthQueryParam).Execute()
	if err != nil {
		return nil, err
	}
	if properties, ok := ipblock.GetPropertiesOk(); ok && properties != nil {
		if ipsOk, ok := properties.GetIpsOk(); ok && ipsOk != nil {
			ips := *ipsOk
			if len(indexes) == 0 {
				ipblockIds = append(ipblockIds, ips...)
			}
			for _, index := range indexes {
				if index < len(ips) {
					ipblockIds = append(ipblockIds, ips[index])
				}
			}
		}
	}
	return ipblockIds, nil
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
