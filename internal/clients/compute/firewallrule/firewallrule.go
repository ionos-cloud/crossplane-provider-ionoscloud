package firewallrule

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

// Client is a wrapper around IONOS Service FirewallRule methods
type Client interface {
	CheckDuplicateFirewallRule(ctx context.Context, datacenterID, serverID, nicID, firewallRuleName, protocol string) (*sdkgo.FirewallRule, error)
	GetFirewallRuleID(firewallRule *sdkgo.FirewallRule) (string, error)
	GetFirewallRule(ctx context.Context, datacenterID, serverID, nicID, firewallRuleID string) (sdkgo.FirewallRule, *sdkgo.APIResponse, error)
	CreateFirewallRule(ctx context.Context, datacenterID, serverID, nicID string, firewallRule sdkgo.FirewallRule) (sdkgo.FirewallRule, *sdkgo.APIResponse, error)
	UpdateFirewallRule(ctx context.Context, datacenterID, serverID, nicID, firewallRuleID string, firewallRule sdkgo.FirewallruleProperties) (sdkgo.FirewallRule, *sdkgo.APIResponse, error)
	DeleteFirewallRule(ctx context.Context, datacenterID, serverID, nicID, firewallRuleID string) (*sdkgo.APIResponse, error)
	GetAPIClient() *sdkgo.APIClient
}

// CheckDuplicateFirewallRule based on firewallRuleName, and the immutable property protocol
func (cp *APIClient) CheckDuplicateFirewallRule(ctx context.Context, datacenterID, serverID, nicID, firewallRuleName, protocol string) (*sdkgo.FirewallRule, error) { // nolint: gocyclo
	firewallRules, _, err := cp.IonosServices.ComputeClient.FirewallRulesApi.DatacentersServersNicsFirewallrulesGet(ctx, datacenterID, serverID, nicID).Depth(utils.DepthQueryParam).Execute()
	if err != nil {
		return nil, err
	}
	matchedItems := make([]sdkgo.FirewallRule, 0)
	if itemsOk, ok := firewallRules.GetItemsOk(); ok && itemsOk != nil {
		for _, item := range *itemsOk {
			if propertiesOk, ok := item.GetPropertiesOk(); ok && propertiesOk != nil {
				if nameOk, ok := propertiesOk.GetNameOk(); ok && nameOk != nil {
					if *nameOk == firewallRuleName {
						// After checking the name, check the immutable properties
						if protocolOk, ok := propertiesOk.GetProtocolOk(); ok && protocolOk != nil {
							if *protocolOk != protocol {
								return nil, fmt.Errorf("error: found firewall rule with the name %v, but immutable property protocol different. expected: %v actual: %v", firewallRuleName, protocol, *protocolOk)
							}
						}
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
		return nil, fmt.Errorf("error: found multiple firewall rules with the name %v", firewallRuleName)
	}
	return &matchedItems[0], nil
}

// GetFirewallRuleID based on FirewallRule
func (cp *APIClient) GetFirewallRuleID(firewallRule *sdkgo.FirewallRule) (string, error) {
	if firewallRule != nil {
		if idOk, ok := firewallRule.GetIdOk(); ok && idOk != nil {
			return *idOk, nil
		}
		return "", fmt.Errorf("error: getting firewall rule id")
	}
	return "", nil
}

// GetFirewallRule based on firewallRuleID
func (cp *APIClient) GetFirewallRule(ctx context.Context, datacenterID, serverID, nicID, firewallRuleID string) (sdkgo.FirewallRule, *sdkgo.APIResponse, error) {
	return cp.IonosServices.ComputeClient.FirewallRulesApi.DatacentersServersNicsFirewallrulesFindById(ctx, datacenterID, serverID, nicID, firewallRuleID).Depth(utils.DepthQueryParam).Execute()
}

// CreateFirewallRule based on FirewallRule properties
func (cp *APIClient) CreateFirewallRule(ctx context.Context, datacenterID, serverID, nicID string, firewallRule sdkgo.FirewallRule) (sdkgo.FirewallRule, *sdkgo.APIResponse, error) {
	return cp.IonosServices.ComputeClient.FirewallRulesApi.DatacentersServersNicsFirewallrulesPost(ctx, datacenterID, serverID, nicID).Firewallrule(firewallRule).Execute()
}

// UpdateFirewallRule based on firewallRuleID and FirewallRule properties
func (cp *APIClient) UpdateFirewallRule(ctx context.Context, datacenterID, serverID, nicID, firewallRuleID string, firewallRule sdkgo.FirewallruleProperties) (sdkgo.FirewallRule, *sdkgo.APIResponse, error) {
	return cp.IonosServices.ComputeClient.FirewallRulesApi.DatacentersServersNicsFirewallrulesPatch(ctx, datacenterID, serverID, nicID, firewallRuleID).Firewallrule(firewallRule).Execute()
}

// DeleteFirewallRule based on firewallRuleID
func (cp *APIClient) DeleteFirewallRule(ctx context.Context, datacenterID, serverID, nicID, firewallRuleID string) (*sdkgo.APIResponse, error) {
	resp, err := cp.IonosServices.ComputeClient.FirewallRulesApi.DatacentersServersNicsFirewallrulesDelete(ctx, datacenterID, serverID, nicID, firewallRuleID).Execute()
	return resp, err
}

// GetAPIClient gets the APIClient
func (cp *APIClient) GetAPIClient() *sdkgo.APIClient {
	return cp.IonosServices.ComputeClient
}

// GenerateCreateFirewallRuleInput returns sdkgo.FirewallRule based on the CR spec
func GenerateCreateFirewallRuleInput(cr *v1alpha1.FirewallRule, sourceIP, targetIP string) (*sdkgo.FirewallRule, error) { // nolint:gocyclo
	instanceCreateInput := sdkgo.FirewallRule{
		Properties: &sdkgo.FirewallruleProperties{
			Protocol: &cr.Spec.ForProvider.Protocol,
		},
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.Name)) {
		instanceCreateInput.Properties.SetName(cr.Spec.ForProvider.Name)
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.SourceMac)) {
		instanceCreateInput.Properties.SetSourceMac(cr.Spec.ForProvider.SourceMac)
	}
	if !utils.IsEmptyValue(reflect.ValueOf(sourceIP)) {
		instanceCreateInput.Properties.SetSourceIp(sourceIP)
	}
	if !utils.IsEmptyValue(reflect.ValueOf(targetIP)) {
		instanceCreateInput.Properties.SetTargetIp(targetIP)
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.IcmpCode)) {
		instanceCreateInput.Properties.SetIcmpType(cr.Spec.ForProvider.IcmpCode)
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.IcmpType)) {
		instanceCreateInput.Properties.SetIcmpType(cr.Spec.ForProvider.IcmpType)
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.PortRangeStart)) {
		instanceCreateInput.Properties.SetPortRangeStart(cr.Spec.ForProvider.PortRangeStart)
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.PortRangeEnd)) {
		instanceCreateInput.Properties.SetPortRangeEnd(cr.Spec.ForProvider.PortRangeEnd)
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.Type)) {
		instanceCreateInput.Properties.SetType(cr.Spec.ForProvider.Type)
	}
	return &instanceCreateInput, nil
}

// GenerateUpdateFirewallRuleInput returns sdkgo.FirewallRuleProperties based on the CR spec modifications
func GenerateUpdateFirewallRuleInput(cr *v1alpha1.FirewallRule, sourceIP, targetIP string) (*sdkgo.FirewallruleProperties, error) { // nolint:gocyclo
	instanceUpdateInput := sdkgo.FirewallruleProperties{
		Name: &cr.Spec.ForProvider.Name,
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.SourceMac)) {
		instanceUpdateInput.SetSourceMac(cr.Spec.ForProvider.SourceMac)
	}
	if !utils.IsEmptyValue(reflect.ValueOf(sourceIP)) {
		instanceUpdateInput.SetSourceIp(sourceIP)
	}
	if utils.IsEmptyValue(reflect.ValueOf(sourceIP)) && !utils.IsEmptyValue(reflect.ValueOf(cr.Status.AtProvider.SourceIP)) {
		instanceUpdateInput.SourceIp = nil
	}
	if !utils.IsEmptyValue(reflect.ValueOf(targetIP)) {
		instanceUpdateInput.SetTargetIp(targetIP)
	}
	if utils.IsEmptyValue(reflect.ValueOf(targetIP)) && !utils.IsEmptyValue(reflect.ValueOf(cr.Status.AtProvider.TargetIP)) {
		instanceUpdateInput.TargetIp = nil
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.IcmpCode)) {
		instanceUpdateInput.SetIcmpType(cr.Spec.ForProvider.IcmpCode)
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.IcmpType)) {
		instanceUpdateInput.SetIcmpType(cr.Spec.ForProvider.IcmpType)
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.PortRangeStart)) {
		instanceUpdateInput.SetPortRangeStart(cr.Spec.ForProvider.PortRangeStart)
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.PortRangeEnd)) {
		instanceUpdateInput.SetPortRangeEnd(cr.Spec.ForProvider.PortRangeEnd)
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.Type)) {
		instanceUpdateInput.SetType(cr.Spec.ForProvider.Type)
	}
	return &instanceUpdateInput, nil
}

// IsFirewallRuleUpToDate returns true if the FirewallRule is up-to-date or false if it does not
func IsFirewallRuleUpToDate(cr *v1alpha1.FirewallRule, firewallRule sdkgo.FirewallRule, sourceIP, targetIP string) bool { // nolint:gocyclo
	switch {
	case cr == nil && firewallRule.Properties == nil:
		return true
	case cr == nil && firewallRule.Properties != nil:
		return false
	case cr != nil && firewallRule.Properties == nil:
		return false
	case firewallRule.Metadata.State != nil && *firewallRule.Metadata.State == "BUSY":
		return true
	case firewallRule.Properties.Name != nil && *firewallRule.Properties.Name != cr.Spec.ForProvider.Name:
		return false
	case firewallRule.Properties.Name == nil && cr.Spec.ForProvider.Name != "":
		return false
	case firewallRule.Properties.SourceMac != nil && *firewallRule.Properties.SourceMac != cr.Spec.ForProvider.SourceMac:
		return false
	case firewallRule.Properties.SourceIp != nil && *firewallRule.Properties.SourceIp != sourceIP:
		return false
	case sourceIP != cr.Status.AtProvider.SourceIP:
		return false
	case firewallRule.Properties.TargetIp != nil && *firewallRule.Properties.TargetIp != targetIP:
		return false
	case targetIP != cr.Status.AtProvider.TargetIP:
		return false
	case firewallRule.Properties.IcmpCode != nil && *firewallRule.Properties.IcmpCode != cr.Spec.ForProvider.IcmpCode:
		return false
	case firewallRule.Properties.IcmpType != nil && *firewallRule.Properties.IcmpType != cr.Spec.ForProvider.IcmpType:
		return false
	case firewallRule.Properties.PortRangeStart != nil && *firewallRule.Properties.PortRangeStart != cr.Spec.ForProvider.PortRangeStart:
		return false
	case firewallRule.Properties.PortRangeEnd != nil && *firewallRule.Properties.PortRangeEnd != cr.Spec.ForProvider.PortRangeEnd:
		return false
	case firewallRule.Properties.Type != nil && *firewallRule.Properties.Type != cr.Spec.ForProvider.Type:
		return false
	default:
		return true
	}
}
