package nic

import (
	"context"
	"fmt"
	"net"
	"reflect"

	sdkgo "github.com/ionos-cloud/sdk-go/v6"
	"github.com/rung/go-safecast"

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
	CheckDuplicateNic(ctx context.Context, datacenterID, serverID, nicName string) (*sdkgo.Nic, error)
	GetNicID(nic *sdkgo.Nic) (string, error)
	GetNic(ctx context.Context, datacenterID, serverID, nicID string) (sdkgo.Nic, *sdkgo.APIResponse, error)
	CreateNic(ctx context.Context, datacenterID, serverID string, nic sdkgo.Nic) (sdkgo.Nic, *sdkgo.APIResponse, error)
	UpdateNic(ctx context.Context, datacenterID, serverID, nicID string, nicProperties sdkgo.NicProperties) (sdkgo.Nic, *sdkgo.APIResponse, error)
	DeleteNic(ctx context.Context, datacenterID, serverID, nicID string) (*sdkgo.APIResponse, error)
	GetAPIClient() *sdkgo.APIClient
}

// CheckDuplicateNic based on datacenterID, serverID, nicName and the immutable property location
func (cp *APIClient) CheckDuplicateNic(ctx context.Context, datacenterID, serverID, nicName string) (*sdkgo.Nic, error) { // nolint: gocyclo
	nics, _, err := cp.ComputeClient.NetworkInterfacesApi.DatacentersServersNicsGet(ctx, datacenterID, serverID).Depth(utils.DepthQueryParam).Execute()
	if err != nil {
		return nil, err
	}
	matchedItems := make([]sdkgo.Nic, 0)
	if itemsOk, ok := nics.GetItemsOk(); ok && itemsOk != nil {
		for _, item := range *itemsOk {
			if propertiesOk, ok := item.GetPropertiesOk(); ok && propertiesOk != nil {
				if nameOk, ok := propertiesOk.GetNameOk(); ok && nameOk != nil {
					if *nameOk == nicName {
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
		return nil, fmt.Errorf("error: found multiple nics with the name %v", nicName)
	}
	return &matchedItems[0], nil
}

// GetNicID based on nic
func (cp *APIClient) GetNicID(nic *sdkgo.Nic) (string, error) {
	if nic != nil {
		if idOk, ok := nic.GetIdOk(); ok && idOk != nil {
			return *idOk, nil
		}
		return "", fmt.Errorf("error: getting nic id")
	}
	return "", nil
}

// GetNic based on datacenterID, serverID, nicID
func (cp *APIClient) GetNic(ctx context.Context, datacenterID, serverID, nicID string) (sdkgo.Nic, *sdkgo.APIResponse, error) {
	return cp.ComputeClient.NetworkInterfacesApi.DatacentersServersNicsFindById(ctx, datacenterID, serverID, nicID).Depth(utils.DepthQueryParam).Execute()
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

// LateInitializer fills the empty fields in *v1alpha1.NicParameters with
// the values seen in sdkgo.Nic.
func LateInitializer(in *v1alpha1.NicParameters, sg *sdkgo.Nic) {
	if sg == nil {
		return
	}
	if propertiesOk, ok := sg.GetPropertiesOk(); ok && propertiesOk != nil {
		if firewallTypeOk, ok := propertiesOk.GetFirewallTypeOk(); ok && firewallTypeOk != nil {
			if utils.IsEmptyValue(reflect.ValueOf(in.FirewallType)) {
				in.FirewallType = *firewallTypeOk
			}
		}
	}
}

// LateStatusInitializer fills the empty fields in *v1alpha1.NicObservation with
// the values seen in sdkgo.Nic.
func LateStatusInitializer(in *v1alpha1.NicObservation, nic *sdkgo.Nic) {
	if nic == nil {
		return
	}
	// Add options to the Spec, if they were updated by the API
	if propertiesOk, ok := nic.GetPropertiesOk(); ok && propertiesOk != nil {
		if PCISlotOk, ok := propertiesOk.GetPciSlotOk(); ok && PCISlotOk != nil {
			if in.PCISlot == 0 {
				in.PCISlot = *PCISlotOk
			}
		}
		if nic.Properties.HasIps() {
			in.IPs = *nic.Properties.Ips
		}
		if nic.Properties.HasMac() {
			in.Mac = *nic.Properties.Mac
		}
	}
}

// GenerateCreateNicInput returns sdkgo.Nic based on the CR spec
func GenerateCreateNicInput(cr *v1alpha1.Nic, ips []string) (*sdkgo.Nic, error) { // nolint:gocyclo
	properties, err := GenerateUpdateNicInput(cr, ips)
	if err != nil {
		return nil, err
	}

	return &sdkgo.Nic{
		Properties: properties,
	}, nil
}

// GenerateUpdateNicInput returns sdkgo.NicProperties based on the CR spec modifications
func GenerateUpdateNicInput(cr *v1alpha1.Nic, ips []string) (*sdkgo.NicProperties, error) { // nolint:gocyclo
	lanID, err := safecast.Atoi32(cr.Spec.ForProvider.LanCfg.LanID)
	if err != nil {
		return nil, err
	}
	instanceUpdateInput := sdkgo.NicProperties{
		Lan:            &lanID,
		FirewallActive: &cr.Spec.ForProvider.FirewallActive,
		Dhcp:           &cr.Spec.ForProvider.Dhcp,
	}

	if cr.Spec.ForProvider.DhcpV6 == nil {
		instanceUpdateInput.SetDhcpv6Nil()
	} else {
		instanceUpdateInput.Dhcpv6 = cr.Spec.ForProvider.DhcpV6
	}

	if cr.Spec.ForProvider.Name != "" {
		instanceUpdateInput.SetName(cr.Spec.ForProvider.Name)
	}
	if len(ips) > 0 {
		ipv4s, ipv6s := GetIPvSlices(ips)
		if len(ipv4s) > 0 {
			instanceUpdateInput.SetIps(ips)
		}
		if len(ipv6s) > 0 {
			instanceUpdateInput.SetIpv6Ips(ips)
		}
	}
	if cr.Spec.ForProvider.FirewallType != "" {
		instanceUpdateInput.SetFirewallType(cr.Spec.ForProvider.FirewallType)
	}
	if cr.Spec.ForProvider.Vnet != "" {
		instanceUpdateInput.SetVnet(cr.Spec.ForProvider.Vnet)
	}

	return &instanceUpdateInput, nil
}

// IsNicUpToDate returns true if the Nic is up-to-date or false if it does not
func IsNicUpToDate(cr *v1alpha1.Nic, nic sdkgo.Nic, ips []string) bool { // nolint:gocyclo
	var ipv4s []string
	var ipv6s []string

	if len(ips) > 0 {
		ipv4s, ipv6s = GetIPvSlices(ips)
	}

	switch {
	case cr == nil && nic.Properties == nil:
		return true
	case cr == nil && nic.Properties != nil:
		return false
	case cr != nil && nic.Properties == nil:
		return false
	case nic.Metadata != nil && nic.Metadata.State != nil && *nic.Metadata.State == sdkgo.Busy:
		return true
	case nic.Properties.Name != nil && *nic.Properties.Name != cr.Spec.ForProvider.Name:
		return false
	case nic.Properties.Name == nil && cr.Spec.ForProvider.Name != "":
		return false
	case nic.Properties.Dhcp != nil && *nic.Properties.Dhcp != cr.Spec.ForProvider.Dhcp:
		return false
	case nic.Properties.Dhcpv6 != nil && cr.Spec.ForProvider.DhcpV6 != nil && *nic.Properties.Dhcpv6 != *cr.Spec.ForProvider.DhcpV6:
		return false
	case nic.Properties.FirewallActive != nil && *nic.Properties.FirewallActive != cr.Spec.ForProvider.FirewallActive:
		return false
	case nic.Properties.FirewallType != nil && *nic.Properties.FirewallType != cr.Spec.ForProvider.FirewallType:
		return false
	case nic.Properties.Vnet != nil && *nic.Properties.Vnet != cr.Spec.ForProvider.Vnet:
		return false
	case len(ipv4s) != 0 && nic.Properties.HasIps() && !utils.ContainsStringSlices(ipv4s, *nic.Properties.Ips):
		return false
	case len(ipv6s) != 0 && nic.Properties.HasIpv6Ips() && !utils.ContainsStringSlices(ipv6s, *nic.Properties.Ipv6Ips):
		return false
	default:
		return true
	}
}

// GetIPvSlices splits ips into ipv4 / ipv6 slices
func GetIPvSlices(ips []string) (ipv4s []string, ipv6s []string) {
	for _, ip := range ips {
		parsedIP := net.ParseIP(ip)
		if parsedIP == nil {
			continue
		}

		if parsedIP.To4() != nil {
			ipv4s = append(ipv4s, ip)
		} else {
			ipv6s = append(ipv6s, ip)
		}
	}

	return ipv4s, ipv6s
}
