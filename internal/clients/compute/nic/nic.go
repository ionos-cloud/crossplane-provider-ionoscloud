package nic

import (
	"context"

	sdkgo "github.com/ionos-cloud/sdk-go/v6"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients"
)

// APIClient is a wrapper around IONOS Service
type APIClient struct {
	*clients.IonosServices
}

// Client is a wrapper around IONOS Service Nic methods
type Client interface {
	GetNic(ctx context.Context, datacenterID, serverID, nicID string) (sdkgo.Nic, *sdkgo.APIResponse, error)
	CreateNic(ctx context.Context, datacenterID, serverID string, nic sdkgo.Nic) (sdkgo.Nic, *sdkgo.APIResponse, error)
	UpdateNic(ctx context.Context, datacenterID, serverID, nicID string, nicProperties sdkgo.NicProperties) (sdkgo.Nic, *sdkgo.APIResponse, error)
	DeleteNic(ctx context.Context, datacenterID, serverID, nicID string) (*sdkgo.APIResponse, error)
	GetAPIClient() *sdkgo.APIClient
}

// GetNic based on datacenterID, serverID, nicID
func (cp *APIClient) GetNic(ctx context.Context, datacenterID, serverID, nicID string) (sdkgo.Nic, *sdkgo.APIResponse, error) {
	return cp.ComputeClient.NetworkInterfacesApi.DatacentersServersNicsFindById(ctx, datacenterID, serverID, nicID).Execute()
}

// CreateNic based on Nic properties, using datacenterID and serverID
func (cp *APIClient) CreateNic(ctx context.Context, datacenterID, serverID string, nic sdkgo.Nic) (sdkgo.Nic, *sdkgo.APIResponse, error) {
	return cp.ComputeClient.NetworkInterfacesApi.DatacentersServersNicsPost(ctx, datacenterID, serverID).Nic(nic).Execute()
}

// UpdateNic based on datacenterID, serverID, nicID and Nic properties
func (cp *APIClient) UpdateNic(ctx context.Context, datacenterID, serverID, nicID string, nicProperties sdkgo.NicProperties) (sdkgo.Nic, *sdkgo.APIResponse, error) {
	return cp.ComputeClient.NetworkInterfacesApi.DatacentersServersNicsPatch(ctx, datacenterID, serverID, nicID).Nic(nicProperties).Execute()
}

// DeleteNic based on datacenterID, serverID, nicID
func (cp *APIClient) DeleteNic(ctx context.Context, datacenterID, serverID, nicID string) (*sdkgo.APIResponse, error) {
	return cp.ComputeClient.NetworkInterfacesApi.DatacentersServersNicsDelete(ctx, datacenterID, serverID, nicID).Execute()
}

// GetAPIClient gets the APIClient
func (cp *APIClient) GetAPIClient() *sdkgo.APIClient {
	return cp.ComputeClient
}

// GenerateCreateNicInput returns CreateNicRequest based on the CR spec
func GenerateCreateNicInput(cr *v1alpha1.Nic) (*sdkgo.Nic, error) {
	instanceCreateInput := sdkgo.Nic{
		Properties: &sdkgo.NicProperties{
			Name:           &cr.Spec.ForProvider.Name,
			Mac:            &cr.Spec.ForProvider.Mac,
			Ips:            &cr.Spec.ForProvider.Ips,
			Dhcp:           &cr.Spec.ForProvider.Dhcp,
			Lan:            &cr.Spec.ForProvider.Lan,
			FirewallActive: &cr.Spec.ForProvider.FirewallActive,
			FirewallType:   &cr.Spec.ForProvider.FirewallType,
		},
	}
	return &instanceCreateInput, nil
}

// GenerateUpdateNicInput returns PatchNicRequest based on the CR spec modifications
func GenerateUpdateNicInput(cr *v1alpha1.Nic) (*sdkgo.NicProperties, error) {
	instanceUpdateInput := sdkgo.NicProperties{
		Name:           &cr.Spec.ForProvider.Name,
		Mac:            &cr.Spec.ForProvider.Mac,
		Ips:            &cr.Spec.ForProvider.Ips,
		Dhcp:           &cr.Spec.ForProvider.Dhcp,
		Lan:            &cr.Spec.ForProvider.Lan,
		FirewallActive: &cr.Spec.ForProvider.FirewallActive,
		FirewallType:   &cr.Spec.ForProvider.FirewallType,
	}
	return &instanceUpdateInput, nil
}

// IsNicUpToDate returns true if the Nic is up-to-date or false if it does not
func IsNicUpToDate(cr *v1alpha1.Nic, nic sdkgo.Nic) bool { // nolint:gocyclo
	switch {
	case cr == nil && nic.Properties == nil:
		return true
	case cr == nil && nic.Properties != nil:
		return false
	case cr != nil && nic.Properties == nil:
		return false
	case nic.Metadata != nil && *nic.Metadata.State == "BUSY":
		return true
	case nic.Properties != nil && *nic.Properties.Name != cr.Spec.ForProvider.Name:
		return false
	default:
		return true
	}
}
