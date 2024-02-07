package managementgroup

import (
	"context"
	"fmt"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"
	sdkgo "github.com/ionos-cloud/sdk-go/v6"
)

// APIClient is a wrapper around IONOS Service
type APIClient struct {
	*clients.IonosServices
}

// Client is a wrapper around IONOS Service Group methods
type Client interface {
	CheckDuplicateGroup(ctx context.Context, groupName string) (*sdkgo.Group, error)
	GetGroupID(group *sdkgo.Group) (string, error)
	GetGroup(ctx context.Context, groupID string) (sdkgo.Group, *sdkgo.APIResponse, error)
	GetGroupMembers(ctx context.Context, groupID string) (sdkgo.GroupMembers, *sdkgo.APIResponse, error)
	GetGroupResourceShares(ctx context.Context, groupID string) (sdkgo.GroupShares, *sdkgo.APIResponse, error)
	CreateGroup(ctx context.Context, group sdkgo.Group) (sdkgo.Group, *sdkgo.APIResponse, error)
	UpdateGroup(ctx context.Context, groupID string, group sdkgo.Group) (sdkgo.Group, *sdkgo.APIResponse, error)
	DeleteGroup(ctx context.Context, groupID string) (*sdkgo.APIResponse, error)
	GetAPIClient() *sdkgo.APIClient
}

// CheckDuplicateGroup based on groupName
func (cp *APIClient) CheckDuplicateGroup(ctx context.Context, groupName string) (*sdkgo.Group, error) {
	groups, _, err := cp.ComputeClient.UserManagementApi.UmGroupsGet(ctx).Depth(utils.DepthQueryParam).Execute()
	if err != nil {
		return nil, err
	}
	matchedItems := make([]sdkgo.Group, 0)
	if itemsOk, ok := groups.GetItemsOk(); ok && itemsOk != nil {
		for _, item := range *itemsOk {
			if propertiesOk, ok := item.GetPropertiesOk(); ok && propertiesOk != nil {
				if nameOk, ok := propertiesOk.GetNameOk(); ok && nameOk != nil {
					if *nameOk == groupName {
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
		return nil, fmt.Errorf("error: found multiple groups with the name %v", groupName)
	}
	return &matchedItems[0], nil
}

// GetGroupID based on group
func (cp *APIClient) GetGroupID(group *sdkgo.Group) (string, error) {
	if group != nil {
		if idPtr, ok := group.GetIdOk(); ok && idPtr != nil {
			return *idPtr, nil
		}
		return "", fmt.Errorf("error: getting group id")
	}
	return "", nil
}

// GetGroup based on groupID
func (cp *APIClient) GetGroup(ctx context.Context, groupID string) (sdkgo.Group, *sdkgo.APIResponse, error) {
	return cp.ComputeClient.UserManagementApi.UmGroupsFindById(ctx, groupID).Depth(utils.DepthQueryParam).Execute()
}

// GetGroupMembers retrieves users that are added to the group
func (cp *APIClient) GetGroupMembers(ctx context.Context, groupID string) (sdkgo.GroupMembers, *sdkgo.APIResponse, error) {
	return cp.ComputeClient.UserManagementApi.UmGroupsUsersGet(ctx, groupID).Execute()
}

// GetGroupResourceShares WIP
func (cp *APIClient) GetGroupResourceShares(ctx context.Context, groupID string) (sdkgo.GroupShares, *sdkgo.APIResponse, error) {
	return cp.ComputeClient.UserManagementApi.UmGroupsSharesGet(ctx, groupID).Execute()
}

// CreateGroup based on Group properties
func (cp *APIClient) CreateGroup(ctx context.Context, group sdkgo.Group) (sdkgo.Group, *sdkgo.APIResponse, error) {
	return cp.ComputeClient.UserManagementApi.UmGroupsPost(ctx).Group(group).Execute()
}

// UpdateGroup based on groupID and Group properties
func (cp *APIClient) UpdateGroup(ctx context.Context, groupID string, group sdkgo.Group) (sdkgo.Group, *sdkgo.APIResponse, error) {
	return cp.ComputeClient.UserManagementApi.UmGroupsPut(ctx, groupID).Group(group).Execute()
}

// DeleteGroup based on groupID
func (cp *APIClient) DeleteGroup(ctx context.Context, groupID string) (*sdkgo.APIResponse, error) {
	resp, err := cp.ComputeClient.UserManagementApi.UmGroupsDelete(ctx, groupID).Execute()
	return resp, err
}

// GetAPIClient gets the APIClient
func (cp *APIClient) GetAPIClient() *sdkgo.APIClient {
	return cp.ComputeClient
}

// GenerateCreateGroupInput returns sdkgo.Group based on the CR spec
func GenerateCreateGroupInput(cr *v1alpha1.ManagementGroup) (*sdkgo.Group, error) {
	instanceCreateInput := sdkgo.Group{
		Properties: &sdkgo.GroupProperties{
			Name:                        &cr.Spec.ForProvider.Name,
			AccessActivityLog:           &cr.Spec.ForProvider.AccessActivityLog,
			AccessAndManageCertificates: &cr.Spec.ForProvider.AccessAndManageCertificates,
			//AccessAndManageDNS:          &cr.Spec.ForProvider.AccessAndManageDNS,
			AccessAndManageMonitoring: &cr.Spec.ForProvider.AccessAndManageMonitoring,
			CreateBackupUnit:          &cr.Spec.ForProvider.CreateBackupUnit,
			CreateDataCenter:          &cr.Spec.ForProvider.CreateDataCenter,
			CreateFlowLog:             &cr.Spec.ForProvider.CreateFlowLog,
			CreateInternetAccess:      &cr.Spec.ForProvider.CreateInternetAccess,
			CreateK8sCluster:          &cr.Spec.ForProvider.CreateK8sCluster,
			CreatePcc:                 &cr.Spec.ForProvider.CreatePcc,
			CreateSnapshot:            &cr.Spec.ForProvider.CreateSnapshot,
			ManageDBaaS:               &cr.Spec.ForProvider.ManageDBaaS,
			//ManageDataPlatform:        &cr.Spec.ForProvider.ManageDataPlatform,
			//ManageRegistry: 			 &cr.Spec.ForProvider.ManageRegistry,
			ReserveIp:   &cr.Spec.ForProvider.ReserveIP,
			S3Privilege: &cr.Spec.ForProvider.S3Privilege,
		},
	}
	return &instanceCreateInput, nil
}

// GenerateUpdateGroupInput returns sdkgo.GroupProperties based on the CR spec modifications
func GenerateUpdateGroupInput(cr *v1alpha1.ManagementGroup) (*sdkgo.Group, error) {
	instanceUpdateInput := sdkgo.Group{Properties: &sdkgo.GroupProperties{
		Name:                        &cr.Spec.ForProvider.Name,
		AccessActivityLog:           &cr.Spec.ForProvider.AccessActivityLog,
		AccessAndManageCertificates: &cr.Spec.ForProvider.AccessAndManageCertificates,
		AccessAndManageDns:          &cr.Spec.ForProvider.AccessAndManageDNS,
		AccessAndManageMonitoring:   &cr.Spec.ForProvider.AccessAndManageMonitoring,
		CreateBackupUnit:            &cr.Spec.ForProvider.CreateBackupUnit,
		CreateDataCenter:            &cr.Spec.ForProvider.CreateDataCenter,
		CreateFlowLog:               &cr.Spec.ForProvider.CreateFlowLog,
		CreateInternetAccess:        &cr.Spec.ForProvider.CreateInternetAccess,
		CreateK8sCluster:            &cr.Spec.ForProvider.CreateK8sCluster,
		CreatePcc:                   &cr.Spec.ForProvider.CreatePcc,
		CreateSnapshot:              &cr.Spec.ForProvider.CreateSnapshot,
		ManageDBaaS:                 &cr.Spec.ForProvider.ManageDBaaS,
		ManageDataplatform:          &cr.Spec.ForProvider.ManageDataPlatform,
		ManageRegistry:              &cr.Spec.ForProvider.ManageRegistry,
		ReserveIp:                   &cr.Spec.ForProvider.ReserveIP,
		S3Privilege:                 &cr.Spec.ForProvider.S3Privilege,
	}}
	return &instanceUpdateInput, nil
}

// IsManagementGroupUpToDate returns true if the Group is up-to-date or false otherwise
func IsManagementGroupUpToDate(cr *v1alpha1.ManagementGroup, observed sdkgo.Group) bool { // nolint:gocyclo
	switch {
	case cr == nil && observed.Properties == nil:
		return true
	case cr == nil && observed.Properties != nil:
		return false
	case cr != nil && observed.Properties == nil:
		return false
	}
	return utils.IsEqSdkPropertiesToCR(cr.Spec.ForProvider, *observed.Properties)
}
