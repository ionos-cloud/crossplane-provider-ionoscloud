package managementgroup

import (
	"context"
	"errors"
	"fmt"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"
	sdkgo "github.com/ionos-cloud/sdk-go/v6"
)

type Set[T comparable] map[T]struct{}

func (S Set[T]) Add(elements ...T) {
	for _, e := range elements {
		S[e] = struct{}{}
	}
}
func (S Set[T]) Contains(e T) bool {
	_, ok := S[e]
	return ok
}
func (S Set[T]) Remove(elements ...T) {
	for _, e := range elements {
		delete(S, e)
	}
}
func (S Set[T]) EqualTo(s Set[T]) bool {
	if len(S) != len(s) {
		return false
	}
	for e := range s {
		if _, ok := S[e]; !ok {
			return false
		}
	}
	return true
}
func NewSetFromSlice[T comparable](s []T) Set[T] {
	S := make(Set[T])
	for _, v := range s {
		S[v] = struct{}{}
	}
	return S
}

// APIClient is a wrapper around IONOS Service
type APIClient struct {
	*clients.IonosServices
}

// Client is a wrapper around IONOS Service Group methods
type Client interface {
	CheckDuplicateGroup(ctx context.Context, groupName string) (*sdkgo.Group, error)
	GetGroupID(group *sdkgo.Group) (string, error)
	GetGroup(ctx context.Context, groupID string) (sdkgo.Group, *sdkgo.APIResponse, error)
	GetGroupMembers(ctx context.Context, groupID string) (Set[string], *sdkgo.APIResponse, error)
	GetGroupResourceShares(ctx context.Context, groupID string) (sdkgo.GroupShares, *sdkgo.APIResponse, error)
	CreateGroup(ctx context.Context, group sdkgo.Group) (sdkgo.Group, *sdkgo.APIResponse, error)
	UpdateGroup(ctx context.Context, groupID string, group sdkgo.Group) (sdkgo.Group, *sdkgo.APIResponse, error)
	AddGroupMember(ctx context.Context, groupID, userID string) (sdkgo.User, *sdkgo.APIResponse, error)
	AddGroupMembers(ctx context.Context, groupID string, userIDs Set[string]) error
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
func (cp *APIClient) GetGroupMembers(ctx context.Context, groupID string) (Set[string], *sdkgo.APIResponse, error) {
	members, apiResponse, err := cp.ComputeClient.UserManagementApi.UmGroupsUsersGet(ctx, groupID).Execute()
	if err != nil {
		return nil, apiResponse, err
	}
	memberIDs := Set[string]{}
	if !members.HasItems() {
		return memberIDs, apiResponse, nil
	}
	for _, item := range *members.Items {
		if item.Id != nil {
			memberIDs.Add(*item.Id)
		}
	}
	return memberIDs, apiResponse, nil
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

// AddGroupMember adds the User referenced by userID to the Group with groupID
func (cp *APIClient) AddGroupMember(ctx context.Context, groupID, userID string) (sdkgo.User, *sdkgo.APIResponse, error) {
	return cp.ComputeClient.UserManagementApi.UmGroupsUsersPost(ctx, groupID).User(sdkgo.User{Id: &userID}).Execute()
}

// AddGroupMembers adds each user referenced by the IDs in userIDs list as a member the Group with groupID
func (cp *APIClient) AddGroupMembers(ctx context.Context, groupID string, userIDs Set[string]) error {

	addErrs := make([]error, 0, len(userIDs))
	waitErrs := make([]error, 0, len(userIDs))
	for userID := range userIDs {
		// go for loop semantics
		_userID := userID
		_, apiResponse, err := cp.AddGroupMember(ctx, groupID, _userID)
		if err != nil {
			addErrs = append(addErrs, compute.AddAPIResponseInfo(apiResponse, err))
		}
		if err = compute.WaitForRequest(ctx, cp.GetAPIClient(), apiResponse); err != nil {
			waitErrs = append(waitErrs, err)
		}
	}
	return errors.Join(append(addErrs, waitErrs...)...)
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
func GenerateCreateGroupInput(cr *v1alpha1.ManagementGroup) (*sdkgo.Group, Set[string], error) {
	return GenerateUpdateGroupInput(cr)
}

// GenerateUpdateGroupInput returns sdkgo.Group based on the CR spec modifications
func GenerateUpdateGroupInput(cr *v1alpha1.ManagementGroup) (*sdkgo.Group, Set[string], error) {
	instanceCreateInput := sdkgo.Group{
		Properties: &sdkgo.GroupProperties{
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
		},
	}
	memberIDsSet(cr)

	return &instanceCreateInput, memberIDsSet(cr), nil
}

// IsManagementGroupUpToDate returns true if the Group is up-to-date or false otherwise
func IsManagementGroupUpToDate(cr *v1alpha1.ManagementGroup, observed sdkgo.Group, observedMembersIDs Set[string]) bool { // nolint:gocyclo
	switch {
	case cr == nil && observed.Properties == nil:
		return true
	case cr == nil && observed.Properties != nil:
		return false
	case cr != nil && observed.Properties == nil:
		return false
	}
	configuredMemberIDs := memberIDsSet(cr)

	return utils.IsEqSdkPropertiesToCR(cr.Spec.ForProvider, *observed.Properties) && observedMembersIDs.EqualTo(configuredMemberIDs)
}

func memberIDsSet(cr *v1alpha1.ManagementGroup) Set[string] {
	mCount := len(cr.Spec.ForProvider.UserCfg)
	memberIDs := Set[string]{}
	for i := 0; i < mCount; i++ {
		memberIDs.Add(cr.Spec.ForProvider.UserCfg[i].UserID)
	}
	return memberIDs
}
