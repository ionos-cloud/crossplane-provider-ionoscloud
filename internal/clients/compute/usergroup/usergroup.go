package usergroup

import (
	"context"
	"k8s.io/utils/pointer"
	"strings"

	ionosdk "github.com/ionos-cloud/sdk-go/v6"
	"github.com/pkg/errors"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients"
)

const errRequestWait = "error waiting for request"
const errRemoveUserGroup = " failed to remove user from group"
const depth = 5

// requestWaiter defines a type to wait for requests.
type requestWaiter func(ctx context.Context, client *ionosdk.APIClient, apiResponse *ionosdk.APIResponse) error

// apiClient is a wrapper around IONOS Service.
type apiClient struct {
	svc            *ionosdk.APIClient
	waitForRequest requestWaiter
}

// NewAPIClient returns a new client.
func NewAPIClient(svc *clients.IonosServices, fn requestWaiter) Client {
	return &apiClient{
		svc:            svc.ComputeClient,
		waitForRequest: fn,
	}
}

// Client wraps the ionoscloud api for the user.
// Currently used for mocking the interaction with the client.
type Client interface {
	GetGroup(ctx context.Context, id string) (ionosdk.Group, *ionosdk.APIResponse, error)
	CreateGroup(ctx context.Context, p v1alpha1.GroupParameters) (ionosdk.Group, *ionosdk.APIResponse, error)
	UpdateGroup(ctx context.Context, id string, p v1alpha1.GroupParameters) (ionosdk.Group, *ionosdk.APIResponse, error)
	DeleteGroup(ctx context.Context, id string) (*ionosdk.APIResponse, error)
	AddResource(ctx context.Context, groupID string, resource v1alpha1.Resource) (*ionosdk.APIResponse, error)
	GetResources(ctx context.Context, groupID string) (ionosdk.GroupShares, *ionosdk.APIResponse, error)
	RemoveResourceFromGroup(ctx context.Context, groupID, resourceID string) (*ionosdk.APIResponse, error)
	/*
		AddUserToGroup(ctx context.Context, groupID string, userID string) (ionosdk.User, *ionosdk.APIResponse, error)
		DeleteUserFromGroup(ctx context.Context, groupID string, userID string) error
		GetUserGroups(ctx context.Context, userID string) ([]string, error)
	GetAPIClient() *ionosdk.APIClient*/
}

// GetGroup retrieves a user group via its id.
func (ac *apiClient) GetGroup(ctx context.Context, id string) (ionosdk.Group, *ionosdk.APIResponse, error) {
	return ac.svc.UserManagementApi.UmGroupsFindById(ctx, id).Depth(depth).Execute()
}

// CreateGroup creates a user group in the ionoscloud.
func (ac *apiClient) CreateGroup(ctx context.Context, p v1alpha1.GroupParameters) (ionosdk.Group, *ionosdk.APIResponse, error) {
	props := ionosdk.NewGroupProperties()
	props.SetName(p.Name)
	ac.setGroupPermissions(p, props)

	u := *ionosdk.NewGroup(*props)
	user, resp, err := ac.svc.UserManagementApi.UmGroupsPost(ctx).Group(u).Execute()
	if err != nil {
		return ionosdk.Group{}, resp, err
	}

	if rerr := ac.waitForRequest(ctx, ac.svc, resp); rerr != nil {
		return ionosdk.Group{}, resp, errors.Wrap(rerr, errRequestWait)
	}
	return user, resp, err
}

// DeleteGroup deletes a user group.
func (ac *apiClient) DeleteGroup(ctx context.Context, id string) (*ionosdk.APIResponse, error) {
	resp, err := ac.svc.UserManagementApi.UmGroupsDelete(ctx, id).Execute()
	if err != nil {
		return resp, err
	}

	return resp, errors.Wrap(ac.waitForRequest(ctx, ac.svc, resp), errRequestWait)
}

// UpdateGroup updates a user group.
func (ac *apiClient) UpdateGroup(ctx context.Context, id string, p v1alpha1.GroupParameters) (ionosdk.Group, *ionosdk.APIResponse, error) {
	props := ionosdk.NewGroupProperties()
	props.SetName(p.Name)
	ac.setGroupPermissions(p, props)

	u := *ionosdk.NewGroup(*props)
	user, resp, err := ac.svc.UserManagementApi.UmGroupsPut(ctx, id).Group(u).Execute()
	if err != nil {
		return ionosdk.Group{}, resp, err
	}

	if err = ac.waitForRequest(ctx, ac.svc, resp); err != nil {
		return user, resp, errors.Wrap(err, errRequestWait)
	}

	return user, resp, err
}

func (ac *apiClient) AddResource(ctx context.Context, groupID string, resource v1alpha1.Resource) (*ionosdk.APIResponse, error) {
	props := ionosdk.NewGroupShareProperties()
	props.SetSharePrivilege(resource.SharePrivilege)
	props.SetEditPrivilege(resource.EditPrivilege)

	groupShare := ionosdk.NewGroupShare(*props)
	groupShare.SetId(resource.ID)

	_, responce, err := ac.svc.UserManagementApi.UmGroupsSharesPost(ctx, groupID, resource.ID).
		Resource(*groupShare).Execute()
	if err != nil {
		return responce, err
	}

	if err = ac.waitForRequest(ctx, ac.svc, responce); err != nil {
		return responce, errors.Wrap(err, errRequestWait)
	}

	return responce, nil
}

func (ac *apiClient) RemoveResourceFromGroup(ctx context.Context, groupID, resourceID string) (*ionosdk.APIResponse, error) {
	return ac.svc.UserManagementApi.UmGroupsSharesDelete(ctx, groupID, resourceID).Execute()
}

func (ac *apiClient) GetResources(ctx context.Context, groupID string) (ionosdk.GroupShares, *ionosdk.APIResponse, error) {
	return ac.svc.UserManagementApi.UmGroupsSharesGet(ctx, groupID).Depth(depth).Execute()
}

func (ac *apiClient) setGroupPermissions(p v1alpha1.GroupParameters, props *ionosdk.GroupProperties) {
	for _, privilege := range p.Privileges {
		switch strings.ToLower(privilege) {
		case v1alpha1.CreateDataCenter:
			props.SetCreateDataCenter(true)
		case v1alpha1.CreateSnapshot:
			props.CreateSnapshot = pointer.Bool(true)
		case v1alpha1.ReserveIp:
			props.SetReserveIp(true)
		case v1alpha1.AccessActivityLog:
			props.SetAccessActivityLog(true)
		case v1alpha1.CreatePcc:
			props.SetCreatePcc(true)
		case v1alpha1.S3Privilege:
			props.SetS3Privilege(true)
		case v1alpha1.CreateBackupUnit:
			props.SetCreateBackupUnit(true)
		case v1alpha1.CreateInternetAccess:
			props.SetCreateInternetAccess(true)
		case v1alpha1.CreateK8sCluster:
			props.SetCreateK8sCluster(true)
		case v1alpha1.CreateFlowLog:
			props.SetCreateFlowLog(true)
		case v1alpha1.AccessAndManageMonitoring:
			props.SetAccessAndManageMonitoring(true)
		case v1alpha1.AccessAndManageCertificates:
			props.SetAccessAndManageCertificates(true)
		case v1alpha1.ManageDBaaS:
			props.SetManageDBaaS(true)
			// case "accessandmanagedns": This is not defined in the SDK, but it is defined in the documentation
			// case "manageregistry":
			//case "manageDataplatform":

		}
	}
}
