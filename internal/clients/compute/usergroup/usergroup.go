package usergroup

import (
	"context"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"
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
const errFailedToGetProperties = "could not get user group properties"

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
	GetPrivilegesMap(props *ionosdk.GroupProperties) map[string]bool
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
	ionosGroup, resp, err := ac.GetGroup(ctx, id)
	if err != nil {
		return ionosdk.Group{}, resp, err
	}
	props := ionosGroup.GetProperties()
	if props == nil {
		return ionosdk.Group{}, nil, errors.New(errFailedToGetProperties)
	}
	ac.removePermissions(p, props)
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

func (ac *apiClient) GetPrivilegesMap(props *ionosdk.GroupProperties) map[string]bool {
	m := make(map[string]bool)
	m[v1alpha1.CreateDataCenter] = utils.DereferenceOrZero(props.GetCreateDataCenter())
	m[v1alpha1.CreateSnapshot] = utils.DereferenceOrZero(props.GetCreateSnapshot())
	m[v1alpha1.ReserveIp] = utils.DereferenceOrZero(props.GetReserveIp())
	m[v1alpha1.AccessActivityLog] = utils.DereferenceOrZero(props.GetAccessActivityLog())
	m[v1alpha1.CreatePcc] = utils.DereferenceOrZero(props.GetCreatePcc())
	m[v1alpha1.S3Privilege] = utils.DereferenceOrZero(props.GetS3Privilege())
	m[v1alpha1.CreateBackupUnit] = utils.DereferenceOrZero(props.GetCreateBackupUnit())
	m[v1alpha1.CreateInternetAccess] = utils.DereferenceOrZero(props.GetCreateInternetAccess())
	m[v1alpha1.CreateK8sCluster] = utils.DereferenceOrZero(props.GetCreateK8sCluster())
	m[v1alpha1.CreateFlowLog] = utils.DereferenceOrZero(props.GetCreateFlowLog())
	m[v1alpha1.AccessAndManageMonitoring] = utils.DereferenceOrZero(props.GetAccessAndManageMonitoring())
	m[v1alpha1.AccessAndManageCertificates] = utils.DereferenceOrZero(props.GetAccessAndManageCertificates())
	m[v1alpha1.ManageDBaaS] = utils.DereferenceOrZero(props.GetManageDBaaS())

	return m
}
func (ac *apiClient) removePermissions(p v1alpha1.GroupParameters, props *ionosdk.GroupProperties) {
	propertiesMap := ac.GetPrivilegesMap(props)
	for _, privilege := range p.Privileges {
		propertiesMap[strings.ToLower(privilege)] = false
	}
	for privilege, v := range propertiesMap {
		if v {
			setPrivilege(privilege, props, false)
		}
	}
}
func setPrivilege(privilege string, props *ionosdk.GroupProperties, value bool) {
	switch strings.ToLower(privilege) {
	case v1alpha1.CreateDataCenter:
		props.SetCreateDataCenter(value)
	case v1alpha1.CreateSnapshot:
		props.CreateSnapshot = pointer.Bool(value)
	case v1alpha1.ReserveIp:
		props.SetReserveIp(value)
	case v1alpha1.AccessActivityLog:
		props.SetAccessActivityLog(value)
	case v1alpha1.CreatePcc:
		props.SetCreatePcc(value)
	case v1alpha1.S3Privilege:
		props.SetS3Privilege(value)
	case v1alpha1.CreateBackupUnit:
		props.SetCreateBackupUnit(value)
	case v1alpha1.CreateInternetAccess:
		props.SetCreateInternetAccess(value)
	case v1alpha1.CreateK8sCluster:
		props.SetCreateK8sCluster(value)
	case v1alpha1.CreateFlowLog:
		props.SetCreateFlowLog(value)
	case v1alpha1.AccessAndManageMonitoring:
		props.SetAccessAndManageMonitoring(value)
	case v1alpha1.AccessAndManageCertificates:
		props.SetAccessAndManageCertificates(value)
	case v1alpha1.ManageDBaaS:
		props.SetManageDBaaS(value)
		// case "accessandmanagedns": This is not defined in the SDK, but it is defined in the documentation
		// case "manageregistry":
		//case "manageDataplatform":

	}
}
func (ac *apiClient) setGroupPermissions(p v1alpha1.GroupParameters, props *ionosdk.GroupProperties) {
	for _, privilege := range p.Privileges {
		setPrivilege(privilege, props, true)
	}
}
