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
	GetLan(ctx context.Context, datacenterID, lanID string) (sdkgo.Lan, *sdkgo.APIResponse, error)
	GetLanIPFailovers(ctx context.Context, datacenterID, lanID string) ([]sdkgo.IPFailover, error)
	CreateLan(ctx context.Context, datacenterID string, lan sdkgo.LanPost) (sdkgo.LanPost, *sdkgo.APIResponse, error)
	UpdateLan(ctx context.Context, datacenterID, lanID string, lan sdkgo.LanProperties) (sdkgo.Lan, *sdkgo.APIResponse, error)
	DeleteLan(ctx context.Context, datacenterID, lanID string) (*sdkgo.APIResponse, error)
	GetAPIClient() *sdkgo.APIClient
}

// GetLan based on datacenterID, lanID
func (cp *APIClient) GetLan(ctx context.Context, datacenterID, lanID string) (sdkgo.Lan, *sdkgo.APIResponse, error) {
	return cp.ComputeClient.LANsApi.DatacentersLansFindById(ctx, datacenterID, lanID).Depth(utils.DepthQueryParam).Execute()
}

// GetLanIPFailovers based on datacenterID, lanID
func (cp *APIClient) GetLanIPFailovers(ctx context.Context, datacenterID, lanID string) ([]sdkgo.IPFailover, error) {
	lan, _, err := cp.ComputeClient.LANsApi.DatacentersLansFindById(ctx, datacenterID, lanID).Depth(utils.DepthQueryParam).Execute()
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
func (cp *APIClient) CreateLan(ctx context.Context, datacenterID string, lan sdkgo.LanPost) (sdkgo.LanPost, *sdkgo.APIResponse, error) {
	return cp.ComputeClient.LANsApi.DatacentersLansPost(ctx, datacenterID).Lan(lan).Execute()
}

// UpdateLan based on datacenterID, lanID and Lan properties
func (cp *APIClient) UpdateLan(ctx context.Context, datacenterID, lanID string, lan sdkgo.LanProperties) (sdkgo.Lan, *sdkgo.APIResponse, error) {
	return cp.ComputeClient.LANsApi.DatacentersLansPatch(ctx, datacenterID, lanID).Lan(lan).Execute()
}

// DeleteLan based on datacenterID, lanID
func (cp *APIClient) DeleteLan(ctx context.Context, datacenterID, lanID string) (*sdkgo.APIResponse, error) {
	resp, err := cp.ComputeClient.LANsApi.DatacentersLansDelete(ctx, datacenterID, lanID).Execute()
	return resp, err
}

// GetAPIClient gets the APIClient
func (cp *APIClient) GetAPIClient() *sdkgo.APIClient {
	return cp.ComputeClient
}

// GenerateCreateLanInput returns sdkgo.LanPost based on the CR spec
func GenerateCreateLanInput(cr *v1alpha1.Lan) (*sdkgo.LanPost, error) {
	instanceCreateInput := sdkgo.LanPost{
		Properties: &sdkgo.LanPropertiesPost{
			Public: &cr.Spec.ForProvider.Public,
		},
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.Name)) {
		instanceCreateInput.Properties.SetName(cr.Spec.ForProvider.Name)
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.Pcc)) {
		instanceCreateInput.Properties.SetPcc(cr.Spec.ForProvider.Pcc)
	}
	return &instanceCreateInput, nil
}

// GenerateUpdateLanInput returns sdkgo.LanProperties based on the CR spec modifications
func GenerateUpdateLanInput(cr *v1alpha1.Lan) (*sdkgo.LanProperties, error) {
	instanceUpdateInput := sdkgo.LanProperties{
		Public: &cr.Spec.ForProvider.Public,
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.Name)) {
		instanceUpdateInput.SetName(cr.Spec.ForProvider.Name)
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.Pcc)) {
		instanceUpdateInput.SetPcc(cr.Spec.ForProvider.Pcc)
	}
	return &instanceUpdateInput, nil
}

// IsLanUpToDate returns true if the Lan is up-to-date or false if it does not
func IsLanUpToDate(cr *v1alpha1.Lan, lan sdkgo.Lan) bool { // nolint:gocyclo
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
	case lan.Properties.Public != nil && *lan.Properties.Public != cr.Spec.ForProvider.Public:
		return false
	case lan.Properties.Pcc != nil && *lan.Properties.Pcc != cr.Spec.ForProvider.Pcc:
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
