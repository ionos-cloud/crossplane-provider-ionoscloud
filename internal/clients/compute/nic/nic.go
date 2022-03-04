package nic

import (
	"context"
	"reflect"

	"github.com/rung/go-safecast"

	sdkgo "github.com/ionos-cloud/sdk-go/v6"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"
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
func GenerateCreateNicInput(cr *v1alpha1.Nic) (*sdkgo.Nic, error) { // nolint:gocyclo
	lanID, err := safecast.Atoi32(cr.Spec.ForProvider.LanCfg.LanID)
	if err != nil {
		return nil, err
	}
	instanceCreateInput := sdkgo.Nic{
		Properties: &sdkgo.NicProperties{
			Lan: &lanID,
		},
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.Name)) {
		instanceCreateInput.Properties.SetName(cr.Spec.ForProvider.Name)
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.Mac)) {
		instanceCreateInput.Properties.SetMac(cr.Spec.ForProvider.Mac)
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.Ips)) {
		instanceCreateInput.Properties.SetIps(cr.Spec.ForProvider.Ips)
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.Dhcp)) {
		instanceCreateInput.Properties.SetDhcp(cr.Spec.ForProvider.Dhcp)
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.FirewallActive)) {
		instanceCreateInput.Properties.SetFirewallActive(cr.Spec.ForProvider.FirewallActive)
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.FirewallType)) {
		instanceCreateInput.Properties.SetFirewallType(cr.Spec.ForProvider.FirewallType)
	}
	return &instanceCreateInput, nil
}

// GenerateUpdateNicInput returns PatchNicRequest based on the CR spec modifications
func GenerateUpdateNicInput(cr *v1alpha1.Nic) (*sdkgo.NicProperties, error) { // nolint:gocyclo
	lanID, err := safecast.Atoi32(cr.Spec.ForProvider.LanCfg.LanID)
	if err != nil {
		return nil, err
	}
	instanceUpdateInput := sdkgo.NicProperties{
		Lan: &lanID,
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.Name)) {
		instanceUpdateInput.SetName(cr.Spec.ForProvider.Name)
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.Mac)) {
		instanceUpdateInput.SetMac(cr.Spec.ForProvider.Mac)
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.Ips)) {
		instanceUpdateInput.SetIps(cr.Spec.ForProvider.Ips)
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.Dhcp)) {
		instanceUpdateInput.SetDhcp(cr.Spec.ForProvider.Dhcp)
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.FirewallActive)) {
		instanceUpdateInput.SetFirewallActive(cr.Spec.ForProvider.FirewallActive)
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.FirewallType)) {
		instanceUpdateInput.SetFirewallType(cr.Spec.ForProvider.FirewallType)
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