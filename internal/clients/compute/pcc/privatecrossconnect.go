package pcc

import (
	"context"
	"fmt"

	sdkgo "github.com/ionos-cloud/sdk-go/v6"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"
)

// APIClient is a wrapper around IONOS Service
type APIClient struct {
	*clients.IonosServices
}

// Client is a wrapper around IONOS Service pcc methods
type Client interface {
	CheckDuplicatePrivateCrossConnect(ctx context.Context, privateCrossConnectName string) (*sdkgo.PrivateCrossConnect, error)
	GetPrivateCrossConnectID(privateCrossConnect *sdkgo.PrivateCrossConnect) (string, error)
	GetPrivateCrossConnect(ctx context.Context, privateCrossConnectID string) (sdkgo.PrivateCrossConnect, *sdkgo.APIResponse, error)
	CreatePrivateCrossConnect(ctx context.Context, privateCrossConnect sdkgo.PrivateCrossConnect) (sdkgo.PrivateCrossConnect, *sdkgo.APIResponse, error)
	UpdatePrivateCrossConnect(ctx context.Context, privateCrossConnectID string, privateCrossConnect sdkgo.PrivateCrossConnectProperties) (sdkgo.PrivateCrossConnect, *sdkgo.APIResponse, error)
	DeletePrivateCrossConnect(ctx context.Context, privateCrossConnectID string) (*sdkgo.APIResponse, error)
	GetAPIClient() *sdkgo.APIClient
}

// CheckDuplicatePrivateCrossConnect based on privateCrossConnectName, and the immutable property location
func (cp *APIClient) CheckDuplicatePrivateCrossConnect(ctx context.Context, privateCrossConnectName string) (*sdkgo.PrivateCrossConnect, error) { // nolint: gocyclo
	privateCrossConnects, _, err := cp.ComputeClient.PrivateCrossConnectsApi.PccsGet(ctx).Depth(utils.DepthQueryParam).Execute()
	if err != nil {
		return nil, err
	}
	matchedItems := make([]sdkgo.PrivateCrossConnect, 0)
	if itemsOk, ok := privateCrossConnects.GetItemsOk(); ok && itemsOk != nil {
		for _, item := range *itemsOk {
			if propertiesOk, ok := item.GetPropertiesOk(); ok && propertiesOk != nil {
				if nameOk, ok := propertiesOk.GetNameOk(); ok && nameOk != nil {
					if *nameOk == privateCrossConnectName {
						// After checking the name, check the immutable properties
						matchedItems = append(matchedItems, item)
					}
				}
			}
		}
	}
	if len(matchedItems) == 0 {
		return nil, nil
	}
	if len(matchedItems) > 1 {
		return nil, fmt.Errorf("error: found multiple privateCrossConnects with the name %v", privateCrossConnectName)
	}
	return &matchedItems[0], nil
}

// GetPrivateCrossConnectID based on privateCrossConnect
func (cp *APIClient) GetPrivateCrossConnectID(privateCrossConnect *sdkgo.PrivateCrossConnect) (string, error) {
	if privateCrossConnect != nil {
		if idOk, ok := privateCrossConnect.GetIdOk(); ok && idOk != nil {
			return *idOk, nil
		}
		return "", fmt.Errorf("error: getting privateCrossConnect id")
	}
	return "", nil
}

// GetPrivateCrossConnect based on privateCrossConnectID
func (cp *APIClient) GetPrivateCrossConnect(ctx context.Context, privateCrossConnectID string) (sdkgo.PrivateCrossConnect, *sdkgo.APIResponse, error) {
	return cp.ComputeClient.PrivateCrossConnectsApi.PccsFindById(ctx, privateCrossConnectID).Depth(utils.DepthQueryParam).Execute()
}

// CreatePrivateCrossConnect based on pcc properties
func (cp *APIClient) CreatePrivateCrossConnect(ctx context.Context, privateCrossConnect sdkgo.PrivateCrossConnect) (sdkgo.PrivateCrossConnect, *sdkgo.APIResponse, error) {
	return cp.ComputeClient.PrivateCrossConnectsApi.PccsPost(ctx).Pcc(privateCrossConnect).Execute()
}

// UpdatePrivateCrossConnect based on privateCrossConnectID and pcc properties
func (cp *APIClient) UpdatePrivateCrossConnect(ctx context.Context, privateCrossConnectID string, privateCrossConnect sdkgo.PrivateCrossConnectProperties) (sdkgo.PrivateCrossConnect, *sdkgo.APIResponse, error) {
	return cp.ComputeClient.PrivateCrossConnectsApi.PccsPatch(ctx, privateCrossConnectID).Pcc(privateCrossConnect).Execute()
}

// DeletePrivateCrossConnect based on privateCrossConnectID
func (cp *APIClient) DeletePrivateCrossConnect(ctx context.Context, privateCrossConnectID string) (*sdkgo.APIResponse, error) {
	resp, err := cp.ComputeClient.PrivateCrossConnectsApi.PccsDelete(ctx, privateCrossConnectID).Execute()
	return resp, err
}

// GetAPIClient gets the APIClient
func (cp *APIClient) GetAPIClient() *sdkgo.APIClient {
	return cp.ComputeClient
}

// GenerateCreatePrivateCrossConnectInput returns sdkgo.pcc based on the CR spec
func GenerateCreatePrivateCrossConnectInput(cr *v1alpha1.Pcc) (*sdkgo.PrivateCrossConnect, error) {
	instanceCreateInput := sdkgo.PrivateCrossConnect{
		Properties: &sdkgo.PrivateCrossConnectProperties{
			Name:        &cr.Spec.ForProvider.Name,
			Description: &cr.Spec.ForProvider.Description,
		},
	}
	return &instanceCreateInput, nil
}

// GenerateUpdatePrivateCrossConnectInput returns sdkgo.PrivateCrossConnectProperties based on the CR spec modifications
func GenerateUpdatePrivateCrossConnectInput(cr *v1alpha1.Pcc) (*sdkgo.PrivateCrossConnectProperties, error) {
	instanceUpdateInput := sdkgo.PrivateCrossConnectProperties{
		Name:        &cr.Spec.ForProvider.Name,
		Description: &cr.Spec.ForProvider.Description,
	}
	return &instanceUpdateInput, nil
}

// IsPrivateCrossConnectUpToDate returns true if the pcc is up-to-date or false if it does not
func IsPrivateCrossConnectUpToDate(cr *v1alpha1.Pcc, privateCrossConnect sdkgo.PrivateCrossConnect) bool { // nolint:gocyclo
	switch {
	case cr == nil && privateCrossConnect.Properties == nil:
		return true
	case cr == nil && privateCrossConnect.Properties != nil:
		return false
	case cr != nil && privateCrossConnect.Properties == nil:
		return false
	case privateCrossConnect.Metadata.State != nil && *privateCrossConnect.Metadata.State == "BUSY":
		return true
	case privateCrossConnect.Properties.Name != nil && *privateCrossConnect.Properties.Name != cr.Spec.ForProvider.Name:
		return false
	case privateCrossConnect.Properties.Name == nil && cr.Spec.ForProvider.Name != "":
		return false
	case privateCrossConnect.Properties.Description != nil && *privateCrossConnect.Properties.Description != cr.Spec.ForProvider.Description:
		return false
	default:
		return true
	}
}
