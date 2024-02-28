package usergroup

import (
	"context"

	ionosdk "github.com/ionos-cloud/sdk-go/v6"
	"github.com/pkg/errors"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients"
)

const errRequestWait = "error waiting for request"
const errRemoveUserGroup = " failed to remove user from group"

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
	/*
		AddUserToGroup(ctx context.Context, groupID string, userID string) (ionosdk.User, *ionosdk.APIResponse, error)
		DeleteUserFromGroup(ctx context.Context, groupID string, userID string) error
		GetUserGroups(ctx context.Context, userID string) ([]string, error)
	GetAPIClient() *ionosdk.APIClient*/
}

// GetGroup retrieves a user group via its id.
func (ac *apiClient) GetGroup(ctx context.Context, id string) (ionosdk.Group, *ionosdk.APIResponse, error) {
	return ac.svc.UserManagementApi.UmGroupsFindById(ctx, id).Execute()
}

// CreateGroup creates a user group in the ionoscloud.
func (ac *apiClient) CreateGroup(ctx context.Context, p v1alpha1.GroupParameters) (ionosdk.Group, *ionosdk.APIResponse, error) {
	props := ionosdk.NewGroupProperties()
	props.SetName(p.Name)

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
	resp, err := ac.svc.UserManagementApi.UmUsersDelete(ctx, id).Execute()
	if err != nil {
		return resp, err
	}

	return resp, errors.Wrap(ac.waitForRequest(ctx, ac.svc, resp), errRequestWait)
}

// UpdateGroup updates a user group.
func (ac *apiClient) UpdateGroup(ctx context.Context, id string, p v1alpha1.GroupParameters) (ionosdk.Group, *ionosdk.APIResponse, error) {
	props := ionosdk.NewGroupProperties()
	props.SetName(p.Name)

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
