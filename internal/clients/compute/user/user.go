package user

import (
	"context"

	ionosdk "github.com/ionos-cloud/sdk-go/v6"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients"
)

// APIClient is a wrapper around IONOS Service
type APIClient struct {
	*clients.IonosServices
}

// Client wraps the ionoscloud api for the user.
// Currently used for mocking the interaction with the client.
type Client interface {
	GetUser(ctx context.Context, id string) (ionosdk.User, *ionosdk.APIResponse, error)
	CreateUser(ctx context.Context, p v1alpha1.UserParameters) (ionosdk.User, *ionosdk.APIResponse, error)
	UpdateUser(ctx context.Context, id string, p v1alpha1.UserParameters) (ionosdk.User, *ionosdk.APIResponse, error)
	DeleteUser(ctx context.Context, id string) (*ionosdk.APIResponse, error)
	GetAPIClient() *ionosdk.APIClient
}

// GetUser retrieves a user via its id.
func (ac *APIClient) GetUser(ctx context.Context, id string) (ionosdk.User, *ionosdk.APIResponse, error) {
	return ac.ComputeClient.UserManagementApi.UmUsersFindById(ctx, id).Execute()
}

// CreateUser creates a user in the ionoscloud.
func (ac *APIClient) CreateUser(ctx context.Context, p v1alpha1.UserParameters) (ionosdk.User, *ionosdk.APIResponse, error) {
	props := ionosdk.NewUserPropertiesPost()
	setUserProperties(p, props)
	u := *ionosdk.NewUserPost(*props)
	return ac.ComputeClient.UserManagementApi.UmUsersPost(ctx).User(u).Execute()
}

// UpdateUser updates a user.
func (ac *APIClient) UpdateUser(ctx context.Context, id string, p v1alpha1.UserParameters) (ionosdk.User, *ionosdk.APIResponse, error) {
	props := ionosdk.NewUserPropertiesPut()
	setUserProperties(p, props)
	props.SetSecAuthActive(p.SecAuthActive)
	u := *ionosdk.NewUserPut(*props)
	return ac.ComputeClient.UserManagementApi.UmUsersPut(ctx, id).User(u).Execute()
}

// DeleteUser deletes a user.
func (ac *APIClient) DeleteUser(ctx context.Context, id string) (*ionosdk.APIResponse, error) {
	return ac.ComputeClient.UserManagementApi.UmUsersDelete(ctx, id).Execute()
}

// GetAPIClient returns the ionoscloud APIClient
func (ac *APIClient) GetAPIClient() *ionosdk.APIClient {
	return ac.ComputeClient
}

// IsUserUpToDate returns true if the User is up-to-date or false otherwise.
func IsUserUpToDate(params v1alpha1.UserParameters, observed ionosdk.User) bool {
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
	switch {
	case params.Administrator != *props.GetAdministrator():
		return false
	case params.Email != *props.GetEmail():
		return false
	case params.FirstName != *props.GetFirstname():
		return false
	case params.ForceSecAuth != *props.GetForceSecAuth():
		return false
	case params.LastName != *props.GetLastname():
		return false
	case params.Active != *props.GetActive():
		return false
	}
	return true
}

type userPropsSetter interface {
	SetFirstname(v string)
	SetLastname(v string)
	SetEmail(v string)
	SetAdministrator(v bool)
	SetForceSecAuth(v bool)
	SetSecAuthActive(v bool)
	SetPassword(v string)
	SetActive(v bool)
}

// setUserProperties sets the cr values into props.
func setUserProperties(p v1alpha1.UserParameters, props userPropsSetter) {
	props.SetFirstname(p.FirstName)
	props.SetLastname(p.LastName)
	props.SetEmail(p.Email)
	props.SetAdministrator(p.Administrator)
	props.SetForceSecAuth(p.ForceSecAuth)
	if pw := p.Password; pw != "" {
		props.SetPassword(pw)
	}
	props.SetActive(p.Active)
}
