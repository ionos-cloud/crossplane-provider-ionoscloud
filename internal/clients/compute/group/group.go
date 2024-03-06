package group

import (
	"context"
	"errors"
	"fmt"

	sdkgo "github.com/ionos-cloud/sdk-go/v6"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"

	"k8s.io/apimachinery/pkg/util/sets"
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
	GetGroupMembers(ctx context.Context, groupID string) ([]string, *sdkgo.APIResponse, error)
	GetGroupResourceShares(ctx context.Context, groupID string) ([]v1alpha1.ResourceShare, *sdkgo.APIResponse, error)
	CreateGroup(ctx context.Context, group sdkgo.Group) (sdkgo.Group, *sdkgo.APIResponse, error)
	UpdateGroup(ctx context.Context, groupID string, group sdkgo.Group) (sdkgo.Group, *sdkgo.APIResponse, error)
	AddGroupMember(ctx context.Context, groupID, userID string) (*sdkgo.APIResponse, error)
	RemoveGroupMember(ctx context.Context, groupID, userID string) (*sdkgo.APIResponse, error)
	UpdateGroupMembers(ctx context.Context, groupID string, membersIn MembersUpdateOp) error
	AddResourceShare(ctx context.Context, groupID string, share v1alpha1.ResourceShare) (*sdkgo.APIResponse, error)
	RemoveResourceShare(ctx context.Context, groupID string, share v1alpha1.ResourceShare) (*sdkgo.APIResponse, error)
	UpdateResourceShare(ctx context.Context, groupID string, share v1alpha1.ResourceShare) (*sdkgo.APIResponse, error)
	UpdateGroupResourceShares(ctx context.Context, groupID string, sharesIn SharesUpdateOp) error
	DeleteGroup(ctx context.Context, groupID string) (*sdkgo.APIResponse, error)
	GetAPIClient() *sdkgo.APIClient
}

// MembersUpdateOp groups memberIDs in sets depending on the operation in which they will be used
type MembersUpdateOp struct {
	Add, Remove sets.Set[string]
}

// SharesUpdateOp groups resource shares in sets depending on the operation in which they will be used
type SharesUpdateOp struct {
	Add, Update, Remove sets.Set[v1alpha1.ResourceShare]
}

// CheckDuplicateGroup based on groupName
func (cp *APIClient) CheckDuplicateGroup(ctx context.Context, groupName string) (*sdkgo.Group, error) {
	groups, _, err := cp.ComputeClient.UserManagementApi.UmGroupsGet(ctx).Depth(utils.DepthQueryParam).Execute()
	if err != nil {
		return nil, err
	}
	matchedItems := make([]sdkgo.Group, 0)

	if groups.Items != nil {
		for _, item := range *groups.Items {
			if item.Properties != nil && item.Properties.Name != nil && *item.Properties.Name == groupName {
				matchedItems = append(matchedItems, item)
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
func (cp *APIClient) GetGroupMembers(ctx context.Context, groupID string) ([]string, *sdkgo.APIResponse, error) {
	members, apiResponse, err := cp.ComputeClient.UserManagementApi.UmGroupsUsersGet(ctx, groupID).Execute()
	if err != nil {
		return nil, apiResponse, err
	}
	var memberIDs []string
	if !members.HasItems() {
		return memberIDs, apiResponse, nil
	}
	memberIDs = make([]string, 0, len(*members.Items))
	for _, item := range *members.Items {
		if item.Id != nil {
			memberIDs = append(memberIDs, *item.Id)
		}
	}
	return memberIDs, apiResponse, nil
}

// GetGroupResourceShares retrieves resources shares that have been added to the group
func (cp *APIClient) GetGroupResourceShares(ctx context.Context, groupID string) ([]v1alpha1.ResourceShare, *sdkgo.APIResponse, error) {
	shares, apiResponse, err := cp.ComputeClient.UserManagementApi.UmGroupsSharesGet(ctx, groupID).Depth(2).Execute()
	if err != nil {
		return nil, apiResponse, err
	}
	var resourceShares []v1alpha1.ResourceShare
	if !shares.HasItems() {
		return resourceShares, apiResponse, err
	}
	resourceShares = make([]v1alpha1.ResourceShare, 0, len(*shares.Items))
	for _, item := range *shares.Items {
		if item.Id != nil && item.Properties != nil {
			share := v1alpha1.ResourceShare{ResourceID: *item.Id}

			if item.Properties.EditPrivilege != nil {
				share.EditPrivilege = *item.Properties.EditPrivilege
			}
			if item.Properties.SharePrivilege != nil {
				share.SharePrivilege = *item.Properties.SharePrivilege
			}
			resourceShares = append(resourceShares, share)
		}
	}
	return resourceShares, apiResponse, nil
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
func (cp *APIClient) AddGroupMember(ctx context.Context, groupID, userID string) (*sdkgo.APIResponse, error) {
	_, apiResponse, err := cp.ComputeClient.UserManagementApi.UmGroupsUsersPost(ctx, groupID).User(sdkgo.User{Id: &userID}).Execute()
	if err != nil {
		return apiResponse, err
	}
	if err = compute.WaitForRequest(ctx, cp.GetAPIClient(), apiResponse); err != nil {
		return apiResponse, err
	}
	return apiResponse, nil
}

// RemoveGroupMember removes the User referenced by userID from the Group with groupID
func (cp *APIClient) RemoveGroupMember(ctx context.Context, groupID, userID string) (*sdkgo.APIResponse, error) {
	return cp.ComputeClient.UserManagementApi.UmGroupsUsersDelete(ctx, groupID, userID).Execute()
}

// UpdateGroupMembers updates the members of Group depending on modFn using the userIDs set
func (cp *APIClient) UpdateGroupMembers(ctx context.Context, groupID string, membersIn MembersUpdateOp) error {
	errs := make([]error, 0, len(membersIn.Add)+len(membersIn.Remove))
	for memberID := range membersIn.Add {
		apiResponse, err := cp.AddGroupMember(ctx, groupID, memberID)
		if err != nil {
			err = fmt.Errorf("failed to add member: %w", compute.AddAPIResponseInfo(apiResponse, err))
			errs = append(errs, err)
		}
	}
	for memberID := range membersIn.Remove {
		apiResponse, err := cp.RemoveGroupMember(ctx, groupID, memberID)
		if err != nil {
			err = fmt.Errorf("failed to remove member: %w", compute.AddAPIResponseInfo(apiResponse, err))
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// AddResourceShare adds a ResourceShare to the Group with groupID
func (cp *APIClient) AddResourceShare(ctx context.Context, groupID string, share v1alpha1.ResourceShare) (*sdkgo.APIResponse, error) {
	groupShare := sdkgo.GroupShare{Properties: &sdkgo.GroupShareProperties{EditPrivilege: &share.EditPrivilege, SharePrivilege: &share.SharePrivilege}}
	_, apiResponse, err := cp.ComputeClient.UserManagementApi.UmGroupsSharesPost(ctx, groupID, share.ResourceID).Resource(groupShare).Execute()
	if err != nil {
		return apiResponse, err
	}
	if err = compute.WaitForRequest(ctx, cp.GetAPIClient(), apiResponse); err != nil {
		return apiResponse, err
	}
	return apiResponse, nil
}

// UpdateResourceShare updates a ResourceShare of the Group with groupID
func (cp *APIClient) UpdateResourceShare(ctx context.Context, groupID string, share v1alpha1.ResourceShare) (*sdkgo.APIResponse, error) {
	groupShare := sdkgo.GroupShare{Properties: &sdkgo.GroupShareProperties{EditPrivilege: &share.EditPrivilege, SharePrivilege: &share.SharePrivilege}}
	_, apiResponse, err := cp.ComputeClient.UserManagementApi.UmGroupsSharesPut(ctx, groupID, share.ResourceID).Resource(groupShare).Execute()
	if err != nil {
		return apiResponse, err
	}
	if err = compute.WaitForRequest(ctx, cp.GetAPIClient(), apiResponse); err != nil {
		return apiResponse, err
	}
	return apiResponse, nil
}

// RemoveResourceShare removes a ResourceShare from the Group with groupID
func (cp *APIClient) RemoveResourceShare(ctx context.Context, groupID string, share v1alpha1.ResourceShare) (*sdkgo.APIResponse, error) {
	apiResponse, err := cp.ComputeClient.UserManagementApi.UmGroupsSharesDelete(ctx, groupID, share.ResourceID).Execute()
	if err != nil {
		return apiResponse, err
	}
	if err = compute.WaitForRequest(ctx, cp.GetAPIClient(), apiResponse); err != nil {
		return apiResponse, err
	}
	return apiResponse, nil
}

// UpdateGroupResourceShares updates the shared resource set of the Group with groupID with the update data in sharesIn
func (cp *APIClient) UpdateGroupResourceShares(ctx context.Context, groupID string, sharesIn SharesUpdateOp) error {
	errs := make([]error, 0, len(sharesIn.Update)+len(sharesIn.Add)+len(sharesIn.Remove))
	for share := range sharesIn.Add {
		apiResponse, err := cp.AddResourceShare(ctx, groupID, share)
		if err != nil {
			err = fmt.Errorf("failed to add share: %w", compute.AddAPIResponseInfo(apiResponse, err))
			errs = append(errs, err)
		}
	}
	for share := range sharesIn.Update {
		apiResponse, err := cp.UpdateResourceShare(ctx, groupID, share)
		if err != nil {
			err = fmt.Errorf("failed to update share: %w", compute.AddAPIResponseInfo(apiResponse, err))
			errs = append(errs, err)
		}
	}
	for share := range sharesIn.Remove {
		apiResponse, err := cp.RemoveResourceShare(ctx, groupID, share)
		if err != nil {
			err = fmt.Errorf("failed to remove share: %w", compute.AddAPIResponseInfo(apiResponse, err))
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
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

// GenerateUpdateGroupInput returns sdkgo.Group and members that need to be added and deleted based or CR and observed member IDs
func GenerateUpdateGroupInput(cr *v1alpha1.Group, observedMemberIDs []string, observedShares []v1alpha1.ResourceShare) (*sdkgo.Group, MembersUpdateOp, SharesUpdateOp) {
	group, configuredM, configuredS := GenerateCreateGroupInput(cr)

	observedM := sets.New[string](observedMemberIDs...)
	observedS := sets.New[v1alpha1.ResourceShare](observedShares...)
	membersOp := MembersUpdateOp{
		Add:    configuredM.Difference(observedM),
		Remove: observedM.Difference(configuredM),
	}

	sharesOp := sharesUpdateOp(observedS, configuredS)

	return group, membersOp, sharesOp
}

// GenerateCreateGroupInput returns sdkgo.Group and members that need to be added based on CR
func GenerateCreateGroupInput(cr *v1alpha1.Group) (*sdkgo.Group, sets.Set[string], sets.Set[v1alpha1.ResourceShare]) {
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

	return &instanceCreateInput, memberIDsSet(cr), resourceSharesSet(cr)
}

// IsGroupUpToDate returns true if the Group is up-to-date or false otherwise
func IsGroupUpToDate(cr *v1alpha1.Group, observed sdkgo.Group) bool { // nolint:gocyclo
	switch {
	case cr == nil && observed.Properties == nil:
		return true
	case cr == nil && observed.Properties != nil:
		return false
	case cr != nil && observed.Properties == nil:
		return false
	case observed.Properties.Name != nil && *observed.Properties.Name != cr.Spec.ForProvider.Name:
		return false
	case observed.Properties.AccessActivityLog != nil && *observed.Properties.AccessActivityLog != cr.Spec.ForProvider.AccessActivityLog:
		return false
	case observed.Properties.AccessAndManageCertificates != nil && *observed.Properties.AccessAndManageCertificates != cr.Spec.ForProvider.AccessAndManageCertificates:
		return false
	case observed.Properties.AccessAndManageDns != nil && *observed.Properties.AccessAndManageDns != cr.Spec.ForProvider.AccessAndManageDNS:
		return false
	case observed.Properties.AccessAndManageMonitoring != nil && *observed.Properties.AccessAndManageMonitoring != cr.Spec.ForProvider.AccessAndManageMonitoring:
		return false
	case observed.Properties.CreateBackupUnit != nil && *observed.Properties.CreateBackupUnit != cr.Spec.ForProvider.CreateBackupUnit:
		return false
	case observed.Properties.CreateDataCenter != nil && *observed.Properties.CreateDataCenter != cr.Spec.ForProvider.CreateDataCenter:
		return false
	case observed.Properties.CreateFlowLog != nil && *observed.Properties.CreateFlowLog != cr.Spec.ForProvider.CreateFlowLog:
		return false
	case observed.Properties.CreateInternetAccess != nil && *observed.Properties.CreateInternetAccess != cr.Spec.ForProvider.CreateInternetAccess:
		return false
	case observed.Properties.CreateK8sCluster != nil && *observed.Properties.CreateK8sCluster != cr.Spec.ForProvider.CreateK8sCluster:
		return false
	case observed.Properties.CreatePcc != nil && *observed.Properties.CreatePcc != cr.Spec.ForProvider.CreatePcc:
		return false
	case observed.Properties.CreateSnapshot != nil && *observed.Properties.CreateSnapshot != cr.Spec.ForProvider.CreateSnapshot:
		return false
	case observed.Properties.ManageDBaaS != nil && *observed.Properties.ManageDBaaS != cr.Spec.ForProvider.ManageDBaaS:
		return false
	case observed.Properties.ManageDataplatform != nil && *observed.Properties.ManageDataplatform != cr.Spec.ForProvider.ManageDataPlatform:
		return false
	case observed.Properties.ManageRegistry != nil && *observed.Properties.ManageRegistry != cr.Spec.ForProvider.ManageRegistry:
		return false
	case observed.Properties.ReserveIp != nil && *observed.Properties.ReserveIp != cr.Spec.ForProvider.ReserveIP:
		return false
	case observed.Properties.S3Privilege != nil && *observed.Properties.S3Privilege != cr.Spec.ForProvider.S3Privilege:
		return false
	}
	configuredMemberIDs := memberIDsSet(cr)
	observedMemberIDs := sets.New[string](cr.Status.AtProvider.UserIDs...)
	if !observedMemberIDs.Equal(configuredMemberIDs) {
		return false
	}

	configuredResourceShares := resourceSharesSet(cr)
	observedResourceShares := sets.New[v1alpha1.ResourceShare](cr.Status.AtProvider.ResourceShares...)
	return observedResourceShares.Equal(configuredResourceShares)
}

func memberIDsSet(cr *v1alpha1.Group) sets.Set[string] {
	mCount := len(cr.Spec.ForProvider.UserCfg)
	memberIDs := sets.Set[string]{}
	for i := 0; i < mCount; i++ {
		memberIDs.Insert(cr.Spec.ForProvider.UserCfg[i].UserID)
	}
	return memberIDs
}

func resourceSharesSet(cr *v1alpha1.Group) sets.Set[v1alpha1.ResourceShare] {
	rsCount := len(cr.Spec.ForProvider.ResourceShareCfg)
	resourceShares := sets.Set[v1alpha1.ResourceShare]{}
	ids := sets.Set[string]{}
	for i := 0; i < rsCount; i++ {
		resourceShareID := cr.Spec.ForProvider.ResourceShareCfg[i].ResourceID
		if resourceShareID != "" && !ids.Has(resourceShareID) {
			share := v1alpha1.ResourceShare{
				ResourceID:     resourceShareID,
				EditPrivilege:  cr.Spec.ForProvider.ResourceShareCfg[i].EditPrivilege,
				SharePrivilege: cr.Spec.ForProvider.ResourceShareCfg[i].SharePrivilege,
			}
			resourceShares.Insert(share)
			ids.Insert(resourceShareID)
		}
	}
	return resourceShares
}

func sharesUpdateOp(observed, configured sets.Set[v1alpha1.ResourceShare]) SharesUpdateOp {
	ids := func(s sets.Set[v1alpha1.ResourceShare]) (_ids sets.Set[string]) {
		_ids = make(sets.Set[string], len(s))
		for i := range s {
			_ids.Insert(i.ResourceID)
		}
		return _ids
	}
	// Shares that have modified permissions will appear in both Differences between the observed and configured sets so to decide between
	// each operation type (add, update, remove) we need to see if the ID of the Share still exists or not in the other structure
	observedIds := ids(observed)
	configuredIds := ids(configured)
	op := SharesUpdateOp{
		Add:    sets.Set[v1alpha1.ResourceShare]{},
		Update: sets.Set[v1alpha1.ResourceShare]{},
		Remove: sets.Set[v1alpha1.ResourceShare]{},
	}
	for share := range configured.Difference(observed) {
		if observedIds.Has(share.ResourceID) {
			op.Update.Insert(share)
		} else {
			op.Add.Insert(share)
		}
	}
	for share := range observed.Difference(configured) {
		if !configuredIds.Has(share.ResourceID) {
			op.Remove.Insert(share)
		}
	}
	return op
}
