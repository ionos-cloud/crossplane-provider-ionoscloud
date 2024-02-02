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

type Client interface {
	GetUser(ctx context.Context, id string) (ionosdk.User, *ionosdk.APIResponse, error)
	CreateUser(ctx context.Context, u ionosdk.UserPost) (ionosdk.User, *ionosdk.APIResponse, error)
	UpdateUser(ctx context.Context, id string, u ionosdk.UserPut) (ionosdk.User, *ionosdk.APIResponse, error)
	DeleteUser(ctx context.Context, id string) (*ionosdk.APIResponse, error)
	GetAPIClient() *ionosdk.APIClient
}

// GetUser retrieves a user via its id.
func (ac *APIClient) GetUser(ctx context.Context, id string) (ionosdk.User, *ionosdk.APIResponse, error) {
	return ac.ComputeClient.UserManagementApi.UmUsersFindById(ctx, id).Execute()
}

// CreateUser creates a user in the ionoscloud.
func (ac *APIClient) CreateUser(ctx context.Context, u ionosdk.UserPost) (ionosdk.User, *ionosdk.APIResponse, error) {
	return ac.ComputeClient.UserManagementApi.UmUsersPost(ctx).User(u).Execute()
}

// UpdateUser updates a user.
func (ac *APIClient) UpdateUser(ctx context.Context, id string, u ionosdk.UserPut) (ionosdk.User, *ionosdk.APIResponse, error) {
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
func IsUserUpToDate(cr *v1alpha1.User, observed ionosdk.User) bool { // nolint:gocyclo
	if !observed.HasProperties() || cr == nil {
		return false
	}

	// After creation the password is stored as a connection detail secret
	// and removed from the cr. If the cr has a password it means
	// the client wants to update it.
	if cr.Spec.ForProvider.Password != "" {
		return false
	}

	props := observed.GetProperties()
	switch {
	case cr.Spec.ForProvider.Administrator != *props.GetAdministrator():
		return false
	case cr.Spec.ForProvider.Email != *props.GetEmail():
		return false
	case cr.Spec.ForProvider.FirstName != *props.GetFirstname():
		return false
	case cr.Spec.ForProvider.ForceSecAuth != *props.GetForceSecAuth():
		return false
	case cr.Spec.ForProvider.LastName != *props.GetLastname():
		return false
	case cr.Spec.ForProvider.SecAuthActive != *props.GetSecAuthActive():
		return false
	case cr.Spec.ForProvider.Active != *props.GetActive():
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

func SetUserProperties(cr v1alpha1.User, props userPropsSetter) {
	props.SetFirstname(cr.Spec.ForProvider.FirstName)
	props.SetLastname(cr.Spec.ForProvider.LastName)
	props.SetEmail(cr.Spec.ForProvider.Email)
	props.SetAdministrator(cr.Spec.ForProvider.Administrator)
	props.SetForceSecAuth(cr.Spec.ForProvider.ForceSecAuth)
	props.SetSecAuthActive(cr.Spec.ForProvider.SecAuthActive)
	props.SetPassword(cr.Spec.ForProvider.Password)
	props.SetActive(cr.Spec.ForProvider.Active)
}
