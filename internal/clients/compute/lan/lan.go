package lan

import (
	"context"
	"fmt"
	"reflect"

	sdkgo "github.com/ionos-cloud/sdk-go/v6"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"
)

// APIClient is a wrapper around IONOS Service
type APIClient struct {
	*clients.IonosServices
}

// Client is a wrapper around IONOS Service Lan methods
type Client interface {
	CheckDuplicateLan(ctx context.Context, datacenterID, lanName string) (*sdkgo.Lan, error)
	GetLanID(lan *sdkgo.Lan) (string, error)
	GetLan(ctx context.Context, datacenterID, lanID string) (sdkgo.Lan, *sdkgo.APIResponse, error)
	GetLanIPFailovers(ctx context.Context, datacenterID, lanID string) ([]sdkgo.IPFailover, error)
	CreateLan(ctx context.Context, datacenterID string, lan sdkgo.Lan) (sdkgo.Lan, *sdkgo.APIResponse, error)
	UpdateLan(ctx context.Context, datacenterID, lanID string, lan sdkgo.LanProperties) (sdkgo.Lan, *sdkgo.APIResponse, error)
	DeleteLan(ctx context.Context, datacenterID, lanID string) (*sdkgo.APIResponse, error)
	GetAPIClient() *sdkgo.APIClient
}

// CheckDuplicateLan based on datacenterID, lanName
func (cp *APIClient) CheckDuplicateLan(ctx context.Context, datacenterID, lanName string) (*sdkgo.Lan, error) { // nolint: gocyclo
	lans, _, err := cp.IonosServices.ComputeClient.LANsApi.DatacentersLansGet(ctx, datacenterID).Depth(utils.DepthQueryParam).Execute()
	if err != nil {
		return nil, err
	}
	matchedItems := make([]sdkgo.Lan, 0)
	if itemsOk, ok := lans.GetItemsOk(); ok && itemsOk != nil {
		for _, item := range *itemsOk {
			if propertiesOk, ok := item.GetPropertiesOk(); ok && propertiesOk != nil {
				if nameOk, ok := propertiesOk.GetNameOk(); ok && nameOk != nil {
					if *nameOk == lanName {
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
		return nil, fmt.Errorf("error: found multiple lans with the name %v", lanName)
	}
	return &matchedItems[0], nil
}

// GetLanID based on lan
func (cp *APIClient) GetLanID(lan *sdkgo.Lan) (string, error) {
	if lan != nil {
		if idOk, ok := lan.GetIdOk(); ok && idOk != nil {
			return *idOk, nil
		}
		return "", fmt.Errorf("error: getting lan id")
	}
	return "", nil
}

// GetLan based on datacenterID, lanID
func (cp *APIClient) GetLan(ctx context.Context, datacenterID, lanID string) (sdkgo.Lan, *sdkgo.APIResponse, error) {
	return cp.IonosServices.ComputeClient.LANsApi.DatacentersLansFindById(ctx, datacenterID, lanID).Depth(utils.DepthQueryParam).Execute()
}

// GetLanIPFailovers based on datacenterID, lanID
func (cp *APIClient) GetLanIPFailovers(ctx context.Context, datacenterID, lanID string) ([]sdkgo.IPFailover, error) {
	lan, _, err := cp.IonosServices.ComputeClient.LANsApi.DatacentersLansFindById(ctx, datacenterID, lanID).Depth(utils.DepthQueryParam).Execute()
	if err != nil {
		return nil, err
	}
	if propertiesOk, ok := lan.GetPropertiesOk(); ok && propertiesOk != nil {
		if ipFailoversOk, ok := propertiesOk.GetIpFailoverOk(); ok && ipFailoversOk != nil && len(*ipFailoversOk) > 0 {
			return *ipFailoversOk, nil
		}
	}
	return nil, fmt.Errorf("error getting IP failovers from lan: %v", lanID)
}

// CreateLan based on datacenterID and Lan properties
func (cp *APIClient) CreateLan(ctx context.Context, datacenterID string, lan sdkgo.Lan) (sdkgo.Lan, *sdkgo.APIResponse, error) {
	return cp.IonosServices.ComputeClient.LANsApi.DatacentersLansPost(ctx, datacenterID).Lan(lan).Execute()
}

// UpdateLan based on datacenterID, lanID and Lan properties
func (cp *APIClient) UpdateLan(ctx context.Context, datacenterID, lanID string, lan sdkgo.LanProperties) (sdkgo.Lan, *sdkgo.APIResponse, error) {
	return cp.IonosServices.ComputeClient.LANsApi.DatacentersLansPatch(ctx, datacenterID, lanID).Lan(lan).Execute()
}

// DeleteLan based on datacenterID, lanID
func (cp *APIClient) DeleteLan(ctx context.Context, datacenterID, lanID string) (*sdkgo.APIResponse, error) {
	resp, err := cp.IonosServices.ComputeClient.LANsApi.DatacentersLansDelete(ctx, datacenterID, lanID).Execute()
	return resp, err
}

// GetAPIClient gets the APIClient
func (cp *APIClient) GetAPIClient() *sdkgo.APIClient {
	return cp.IonosServices.ComputeClient
}

// GenerateCreateLanInput returns sdkgo.LanPost based on the CR spec
func GenerateCreateLanInput(cr *v1alpha1.Lan) (*sdkgo.Lan, error) {
	instanceCreateInput := sdkgo.Lan{
		Properties: &sdkgo.LanProperties{
			Public: &cr.Spec.ForProvider.Public,
		},
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.Name)) {
		instanceCreateInput.Properties.SetName(cr.Spec.ForProvider.Name)
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.Pcc)) {
		instanceCreateInput.Properties.SetPcc(cr.Spec.ForProvider.Pcc.PrivateCrossConnectID)
	}
	if cr.Spec.ForProvider.Ipv6Cidr != "" {
		instanceCreateInput.Properties.SetIpv6CidrBlock(cr.Spec.ForProvider.Ipv6Cidr)
	} else {
		instanceCreateInput.Properties.SetIpv6CidrBlockNil()
	}
	return &instanceCreateInput, nil
}

// GenerateUpdateLanInput returns sdkgo.LanProperties based on the CR spec modifications
func GenerateUpdateLanInput(cr *v1alpha1.Lan) (*sdkgo.LanProperties, error) {
	instanceUpdateInput := sdkgo.LanProperties{
		Public: &cr.Spec.ForProvider.Public,
	}
	if cr.Spec.ForProvider.Name != "" {
		instanceUpdateInput.SetName(cr.Spec.ForProvider.Name)
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.Pcc)) {
		instanceUpdateInput.SetPcc(cr.Spec.ForProvider.Pcc.PrivateCrossConnectID)
	}
	if cr.Spec.ForProvider.Ipv6Cidr != "" {
		instanceUpdateInput.SetIpv6CidrBlock(cr.Spec.ForProvider.Ipv6Cidr)
	} else {
		instanceUpdateInput.SetIpv6CidrBlockNil()
	}
	return &instanceUpdateInput, nil
}

// NeedsUpDate returns a string indicating whether the Lan needs to be updated or not
func NeedsUpDate(cr *v1alpha1.Lan, lan sdkgo.Lan) string { // nolint:gocyclo
	switch {
	case cr == nil && lan.Properties == nil:
		return "Lan does not exist"
	case cr == nil && lan.Properties != nil:
		return "Lan exists but not managed by Crossplane"
	case lan.Metadata.State != nil && *lan.Metadata.State == "BUSY":
		return "Lan cannot be updated, it is in a busy state"
	case lan.Properties.Name != nil && *lan.Properties.Name != cr.Spec.ForProvider.Name:
		return "Lan name does not match " + fmt.Sprintf("%s != %s", *lan.Properties.Name, cr.Spec.ForProvider.Name)
	case lan.Properties.Name == nil && cr.Spec.ForProvider.Name != "":
		return "Lan name is not set, expected: " + cr.Spec.ForProvider.Name + ", got: nil"
	case lan.Properties.Public != nil && *lan.Properties.Public != cr.Spec.ForProvider.Public:
		return "Lan public property does not match: " + fmt.Sprintf("%t != %t", *lan.Properties.Public, cr.Spec.ForProvider.Public)
	case lan.Properties.Ipv6CidrBlock != nil && *lan.Properties.Ipv6CidrBlock != cr.Spec.ForProvider.Ipv6Cidr:
		return "Lan Ipv6CidrBlock does not match" + cr.Spec.ForProvider.Ipv6Cidr + " != " + *lan.Properties.Ipv6CidrBlock
	case lan.Properties.Pcc != nil && *lan.Properties.Pcc != cr.Spec.ForProvider.Pcc.PrivateCrossConnectID:
		return "Lan Pcc does not match: " + cr.Spec.ForProvider.Pcc.PrivateCrossConnectID + " != " + *lan.Properties.Pcc
	default:
		return ""
	}
}

// IsUpToDate returns true if the Lan is up-to-date or false if it does not
func IsUpToDate(cr *v1alpha1.Lan, lan sdkgo.Lan) bool { // nolint:gocyclo
	switch {
	case cr == nil && lan.Properties == nil:
		return true
	case cr == nil && lan.Properties != nil:
		return false
	case cr != nil && lan.Properties == nil:
		return false
	case lan.Metadata.State != nil && *lan.Metadata.State == "BUSY":
		return true
	case lan.Properties.Name != nil && *lan.Properties.Name != cr.Spec.ForProvider.Name:
		return false
	case lan.Properties.Name == nil && cr.Spec.ForProvider.Name != "":
		return false
	case lan.Properties.Public != nil && *lan.Properties.Public != cr.Spec.ForProvider.Public:
		return false
	case lan.Properties.Ipv6CidrBlock != nil && *lan.Properties.Ipv6CidrBlock != cr.Spec.ForProvider.Ipv6Cidr:
		return false
	case lan.Properties.Pcc != nil && *lan.Properties.Pcc != cr.Spec.ForProvider.Pcc.PrivateCrossConnectID:
		return false
	default:
		return true
	}
}

// GenerateCreateIPFailoverInput returns sdkgo.LanProperties based on ip, nicID and current IPFailovers
func GenerateCreateIPFailoverInput(ipFailovers []sdkgo.IPFailover, ip, nicID string) (*sdkgo.LanProperties, error) {
	var instanceCreateInput sdkgo.LanProperties
	ipFailoverNew := sdkgo.IPFailover{
		Ip:      &ip,
		NicUuid: &nicID,
	}
	if len(ipFailovers) > 0 {
		ipFailovers = append(ipFailovers, ipFailoverNew)
		instanceCreateInput.SetIpFailover(ipFailovers)
	} else {
		instanceCreateInput.SetIpFailover([]sdkgo.IPFailover{ipFailoverNew})
	}
	return &instanceCreateInput, nil
}

// GenerateUpdateIPFailoverInput returns sdkgo.LanProperties based on the new
// IP, old IP and nicID set by the user and the current IPFailovers
func GenerateUpdateIPFailoverInput(ipFailovers []sdkgo.IPFailover, newIP, oldIP, nicID string) (*sdkgo.LanProperties, error) {
	var instanceUpdateInput sdkgo.LanProperties
	if len(ipFailovers) == 0 {
		return nil, fmt.Errorf("error: ipfailovers set must not be nil")
	}
	setIPFailovers := make([]sdkgo.IPFailover, 0)
	for _, ipFailover := range ipFailovers {
		if ipFailover.HasIp() {
			// Get and Update IPFailover based on oldIP
			if *ipFailover.Ip == oldIP {
				ipFailover.SetIp(newIP)
				ipFailover.SetNicUuid(nicID)
			}
			setIPFailovers = append(setIPFailovers, ipFailover)
		}
	}
	instanceUpdateInput.SetIpFailover(setIPFailovers)
	return &instanceUpdateInput, nil
}

// GenerateRemoveIPFailoverInput returns sdkgo.LanProperties based on the ip and the current IPFailovers
func GenerateRemoveIPFailoverInput(ipFailovers []sdkgo.IPFailover, ip string) (*sdkgo.LanProperties, error) {
	var instanceRemoveInput sdkgo.LanProperties
	if len(ipFailovers) == 0 {
		return nil, fmt.Errorf("error: input ipFailovers must not be nil")
	}
	setIPFailovers := make([]sdkgo.IPFailover, 0)
	for _, ipFailover := range ipFailovers {
		if ipFailover.HasIp() && *ipFailover.Ip == ip {
			continue
		}
		setIPFailovers = append(setIPFailovers, ipFailover)
	}
	instanceRemoveInput.SetIpFailover(setIPFailovers)
	return &instanceRemoveInput, nil
}

// IsIPFailoverUpToDate returns true if the IPFailover is up-to-date or false if it does not
func IsIPFailoverUpToDate(cr *v1alpha1.IPFailover, lanIPFailovers []sdkgo.IPFailover, ipSetByUser string) bool { // nolint:gocyclo
	switch {
	case cr == nil:
		return false
	case cr.Status.AtProvider.IP != ipSetByUser:
		return false
	case cr.Status.AtProvider.State != "AVAILABLE":
		return false
	case IsIPFailoverPresent(lanIPFailovers, ipSetByUser, cr.Spec.ForProvider.NicCfg.NicID):
		return true
	default:
		return false
	}
}

// IsIPFailoverPresent returns true if the IPFailover exists in the specified Lan
func IsIPFailoverPresent(ipFailovers []sdkgo.IPFailover, ip, nicID string) bool { // nolint:gocyclo
	if ip == "" || nicID == "" {
		return false
	}
	for _, ipFailover := range ipFailovers {
		if ipFailover.HasIp() && ipFailover.HasNicUuid() {
			if *ipFailover.Ip == ip && *ipFailover.NicUuid == nicID {
				return true
			}
		}
	}
	return false
}

// GetIPFailoverIPs returns all the IPFailovers IPs set on Lan
func GetIPFailoverIPs(lan sdkgo.Lan) []string {
	ips := make([]string, 0)
	if propertiesOk, ok := lan.GetPropertiesOk(); ok && propertiesOk != nil {
		if ipFailoversOk, ok := propertiesOk.GetIpFailoverOk(); ok && ipFailoversOk != nil && len(*ipFailoversOk) > 0 {
			for _, ipFailover := range *ipFailoversOk {
				if ipFailover.HasIp() && ipFailover.HasNicUuid() {
					ips = append(ips, *ipFailover.Ip)
				}
			}
		}
	}
	return ips
}

// GetIpv4CidrBlock returns the Ipv4CidrBlock set on Lan
func GetIpv4CidrBlock(lan sdkgo.Lan) string {
	var cidr string
	if propertiesOk, ok := lan.GetPropertiesOk(); ok && propertiesOk != nil {
		if ipv4CidrBlock, ok := propertiesOk.GetIpv4CidrBlockOk(); ok && ipv4CidrBlock != nil {
			cidr = *ipv4CidrBlock
		}
	}

	return cidr
}
