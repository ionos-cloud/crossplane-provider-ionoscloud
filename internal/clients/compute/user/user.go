package user

import (
	"context"
	stderrors "errors"
	"fmt"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/sets"

	ionosdk "github.com/ionos-cloud/sdk-go/v6"

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
	GetUser(ctx context.Context, id string) (ionosdk.User, *ionosdk.APIResponse, error)
	CreateUser(ctx context.Context, p v1alpha1.UserParameters, passw string) (ionosdk.User, *ionosdk.APIResponse, error)
	UpdateUser(ctx context.Context, id string, p v1alpha1.UserParameters, passw string) (ionosdk.User, *ionosdk.APIResponse, error)
	DeleteUser(ctx context.Context, id string) (*ionosdk.APIResponse, error)
	AddUserToGroup(ctx context.Context, groupID string, userID string) (ionosdk.User, *ionosdk.APIResponse, error)
	DeleteUserFromGroup(ctx context.Context, groupID string, userID string) error
	UpdateUserGroups(ctx context.Context, userID string, observedGroupIDs []string, configuredGroupIDs *[]string) error
	GetUserGroups(ctx context.Context, userID string) ([]string, error)
	GetAPIClient() *ionosdk.APIClient
}

// GetUser retrieves a user via its id.
func (ac *apiClient) GetUser(ctx context.Context, id string) (ionosdk.User, *ionosdk.APIResponse, error) {
	return ac.svc.UserManagementApi.UmUsersFindById(ctx, id).Execute()
}

// CreateUser creates a user in the ionoscloud.
func (ac *apiClient) CreateUser(ctx context.Context, p v1alpha1.UserParameters, passw string) (ionosdk.User, *ionosdk.APIResponse, error) {
	props := ionosdk.NewUserPropertiesPost()
	props.SetFirstname(p.FirstName)
	props.SetLastname(p.LastName)
	props.SetEmail(p.Email)
	props.SetAdministrator(p.Administrator)
	props.SetForceSecAuth(p.ForceSecAuth)
	if passw != "" {
		props.SetPassword(passw)
	}
	props.SetActive(p.Active)
	u := *ionosdk.NewUserPost(*props)
	user, resp, err := ac.svc.UserManagementApi.UmUsersPost(ctx).User(u).Execute()
	if err != nil {
		return ionosdk.User{}, resp, err
	}

	if rerr := ac.waitForRequest(ctx, ac.GetAPIClient(), resp); rerr != nil {
		return ionosdk.User{}, resp, errors.Wrap(rerr, errRequestWait)
	}
	return user, resp, err
}

// UpdateUser updates a user.
func (ac *apiClient) UpdateUser(ctx context.Context, id string, p v1alpha1.UserParameters, passw string) (ionosdk.User, *ionosdk.APIResponse, error) {
	props := ionosdk.NewUserPropertiesPut()
	props.SetFirstname(p.FirstName)
	props.SetLastname(p.LastName)
	props.SetEmail(p.Email)
	props.SetAdministrator(p.Administrator)
	props.SetForceSecAuth(p.ForceSecAuth)
	if passw != "" {
		props.SetPassword(passw)
	}
	props.SetActive(p.Active)
	props.SetSecAuthActive(p.SecAuthActive)
	u := *ionosdk.NewUserPut(*props)
	user, resp, err := ac.svc.UserManagementApi.UmUsersPut(ctx, id).User(u).Execute()
	if err != nil {
		return ionosdk.User{}, resp, err
	}

	if err = ac.waitForRequest(ctx, ac.GetAPIClient(), resp); err != nil {
		return user, resp, errors.Wrap(err, errRequestWait)
	}

	return user, resp, err
}

// DeleteUser deletes a user.
func (ac *apiClient) DeleteUser(ctx context.Context, id string) (*ionosdk.APIResponse, error) {
	resp, err := ac.svc.UserManagementApi.UmUsersDelete(ctx, id).Execute()
	if err != nil {
		return resp, err
	}

	return resp, errors.Wrap(ac.waitForRequest(ctx, ac.GetAPIClient(), resp), errRequestWait)
}

// DeleteUserFromGroup deletes the user from the group.
func (ac *apiClient) DeleteUserFromGroup(ctx context.Context, groupID string, userID string) error {
	resp, err := ac.svc.UserManagementApi.UmGroupsUsersDelete(ctx, groupID, userID).Execute()
	if err != nil {
		return errors.Wrap(err, errRemoveUserGroup)
	}
	return errors.Wrap(ac.waitForRequest(ctx, ac.svc, resp), errRequestWait)
}

// AddUserToGroup adds userID to the group of groupID.
func (ac *apiClient) AddUserToGroup(ctx context.Context, groupID string, userID string) (ionosdk.User, *ionosdk.APIResponse, error) {
	u := ionosdk.UserGroupPost{Id: &userID}
	user, resp, err := ac.svc.UserManagementApi.UmGroupsUsersPost(ctx, groupID).User(u).Execute()
	if err != nil {
		return ionosdk.User{}, resp, err
	}

	if rerr := ac.waitForRequest(ctx, ac.svc, resp); rerr != nil {
		return ionosdk.User{}, resp, errors.Wrap(rerr, errRequestWait)
	}
	return user, resp, err
}

// UpdateUserGroups adds or remove groups for the userID.
func (ac *apiClient) UpdateUserGroups(ctx context.Context, userID string, observedGroupIDs []string, configuredGroupIDs *[]string) error {
	if configuredGroupIDs == nil {
		return nil
	}

	configured := sets.New[string](*configuredGroupIDs...)
	observed := sets.New[string](observedGroupIDs...)

	errs := make([]error, 0, len(observed)+len(configured))

	for groupID := range observed.Difference(configured) {
		if err := ac.DeleteUserFromGroup(ctx, groupID, userID); err != nil {
			err = fmt.Errorf("failed to remove the user from a group: %w", err)
			errs = append(errs, err)
		}
	}

	for groupID := range configured.Difference(observed) {
		if _, _, err := ac.AddUserToGroup(ctx, groupID, userID); err != nil {
			err = fmt.Errorf("failed to add the user to a group: %w", err)
			errs = append(errs, err)
		}
	}

	return stderrors.Join(errs...)
}

func (ac *apiClient) GetUserGroups(ctx context.Context, userID string) ([]string, error) {
	rgroups, _, err := ac.svc.UserManagementApi.UmUsersGroupsGet(ctx, userID).Execute()
	if err != nil {
		return nil, err
	}
	if !rgroups.HasItems() {
		return nil, nil
	}
	ids := make([]string, 0)
	for _, g := range *rgroups.Items {
		ids = append(ids, *g.Id)
	}
	return ids, nil
}

// GetAPIClient returns the ionoscloud apiClient.
func (ac *apiClient) GetAPIClient() *ionosdk.APIClient {
	return ac.svc
}

// IsUserUpToDate returns true if the User is up-to-date or false otherwise.
func IsUserUpToDate(params v1alpha1.UserParameters, observed ionosdk.User, observedGroups []string) bool { //nolint:gocyclo
	if !observed.HasProperties() {
		return false
	}

	// After creation the password is stored as a connection detail secret
	// and removed from the cr. If the cr has a password it means
	// the client wants to update it.
	if params.Password != "" {
		return false
	}

	props := observed.GetProperties()
	adm := props.GetAdministrator()
	email := props.GetEmail()
	fname := props.GetFirstname()
	fsec := props.GetForceSecAuth()
	lname := props.GetLastname()
	active := props.GetActive()

	switch {
	case adm != nil && params.Administrator != *adm:
		return false
	case email != nil && params.Email != *email:
		return false
	case fname != nil && params.FirstName != *fname:
		return false
	case fsec != nil && params.ForceSecAuth != *fsec:
		return false
	case lname != nil && params.LastName != *lname:
		return false
	case active != nil && params.Active != *active:
		return false
	}

	if params.GroupIDs == nil {
		return true
	}

	configuredGroups := sets.New[string](*params.GroupIDs...)
	return configuredGroups.Equal(sets.New[string](observedGroups...))
}
