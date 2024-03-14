package forwardingrule

import (
	"context"
	"errors"
	"fmt"

	sdkgo "github.com/ionos-cloud/sdk-go/v6"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/nlb/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"
)

const (
	ruleGetByIDErr    = "failed to get nlb forwarding rule by ID: %w"
	ruleListErr       = "failed to get nlb forwarding rules list: %w"
	ruleCreateErr     = "failed to create nlb forwarding rule: %w"
	ruleCreateWaitErr = "error while waiting for nlb forwarding rule create request: %w"
	ruleUpdateErr     = "failed to update nlb forwarding rule: %w"
	ruleUpdateWaitErr = "error while waiting for nlb forwarding rule update request: %w"
	ruleDeleteErr     = "failed to delete nlb forwarding rule: %w"
	ruleDeleteWaitErr = "error while waiting for nlb delete request: %w"
)

// ErrNotFound no Network Load Balancer ForwardingRule rule has been found
var ErrNotFound = errors.New("forwarding rule not found")

// APIClient is a wrapper around IONOS Service
type APIClient struct {
	*clients.IonosServices
}

// Client is a wrapper around IONOS Service Network Load Balancer ForwardingRule methods
type Client interface {
	CheckDuplicateForwardingRule(ctx context.Context, datacenterID, nlbID, ruleName string) (string, error)
	GetForwardingRuleByID(ctx context.Context, datacenterID, nlbID, ruleID string) (sdkgo.NetworkLoadBalancerForwardingRule, *sdkgo.APIResponse, error)
	CreateForwardingRule(ctx context.Context, datacenterID, nlbID string, rule sdkgo.NetworkLoadBalancerForwardingRule) (sdkgo.NetworkLoadBalancerForwardingRule, *sdkgo.APIResponse, error)
	UpdateForwardingRule(ctx context.Context, datacenterID, nlbID, ruleID string, ruleProperties sdkgo.NetworkLoadBalancerForwardingRuleProperties) (sdkgo.NetworkLoadBalancerForwardingRule, *sdkgo.APIResponse, error)
	DeleteForwardingRule(ctx context.Context, datacenterID, nlbID, ruleID string) (*sdkgo.APIResponse, error)
}

// CheckDuplicateForwardingRule returns the ID of the duplicate Forwarding Rule if any,
// or an error if multiple Forwarding Rules with the same name are found
func (cp *APIClient) CheckDuplicateForwardingRule(ctx context.Context, datacenterID, nlbID, ruleName string) (string, error) {
	ForwardingRules, _, err := cp.ComputeClient.NetworkLoadBalancersApi.DatacentersNetworkloadbalancersForwardingrulesGet(ctx, datacenterID, nlbID).Depth(utils.DepthQueryParam).Execute()
	if err != nil {
		return "", fmt.Errorf(ruleListErr, err)
	}

	matchedItems := make([]sdkgo.NetworkLoadBalancerForwardingRule, 0)

	if ForwardingRules.Items != nil {
		for _, item := range *ForwardingRules.Items {
			if item.Properties != nil && item.Properties.Name != nil && *item.Properties.Name == ruleName {
				matchedItems = append(matchedItems, item)
			}
		}
	}

	if len(matchedItems) == 0 {
		return "", nil
	}
	if len(matchedItems) > 1 {
		return "", fmt.Errorf("error: found multiple Forwarding Rules with the name %v", ruleName)
	}
	if matchedItems[0].Id == nil {
		return "", fmt.Errorf("error getting ID for Forwarding Rule named: %v", ruleName)
	}
	return *matchedItems[0].Id, nil
}

// GetForwardingRuleByID based on Datacenter ID, NetworkLoadBalancer ID and ForwardingRule ID
func (cp *APIClient) GetForwardingRuleByID(ctx context.Context, datacenterID, nlbID, ruleID string) (sdkgo.NetworkLoadBalancerForwardingRule, *sdkgo.APIResponse, error) {
	rule, apiResponse, err := cp.ComputeClient.NetworkLoadBalancersApi.DatacentersNetworkloadbalancersForwardingrulesFindByForwardingRuleId(ctx, datacenterID, nlbID, ruleID).Depth(utils.DepthQueryParam).Execute()
	if err != nil {
		err = ErrNotFound
		if !apiResponse.HttpNotFound() {
			err = fmt.Errorf(ruleGetByIDErr, err)
		}
	}
	return rule, apiResponse, err
}

// CreateForwardingRule based on Datacenter ID, NetworkLoadBalancer ID and ForwardingRule
func (cp *APIClient) CreateForwardingRule(ctx context.Context, datacenterID, nlbID string, rule sdkgo.NetworkLoadBalancerForwardingRule) (sdkgo.NetworkLoadBalancerForwardingRule, *sdkgo.APIResponse, error) {
	rule, apiResponse, err := cp.ComputeClient.NetworkLoadBalancersApi.DatacentersNetworkloadbalancersForwardingrulesPost(ctx, datacenterID, nlbID).NetworkLoadBalancerForwardingRule(rule).Execute()
	if err != nil {
		return sdkgo.NetworkLoadBalancerForwardingRule{}, apiResponse, fmt.Errorf(ruleCreateErr, err)
	}
	if err = compute.WaitForRequest(ctx, cp.ComputeClient, apiResponse); err != nil {
		return sdkgo.NetworkLoadBalancerForwardingRule{}, apiResponse, fmt.Errorf(ruleCreateWaitErr, err)
	}
	return rule, apiResponse, err
}

// UpdateForwardingRule based on Datacenter ID, NetworkLoadBalancer ID, ForwardingRule ID and ForwardingRule
func (cp *APIClient) UpdateForwardingRule(ctx context.Context, datacenterID, nlbID, ruleID string, ruleProperties sdkgo.NetworkLoadBalancerForwardingRuleProperties) (sdkgo.NetworkLoadBalancerForwardingRule, *sdkgo.APIResponse, error) {
	rule, apiResponse, err := cp.ComputeClient.NetworkLoadBalancersApi.DatacentersNetworkloadbalancersForwardingrulesPatch(ctx, datacenterID, nlbID, ruleID).NetworkLoadBalancerForwardingRuleProperties(ruleProperties).Execute()
	if err != nil {
		return sdkgo.NetworkLoadBalancerForwardingRule{}, apiResponse, fmt.Errorf(ruleUpdateErr, err)
	}
	if err = compute.WaitForRequest(ctx, cp.ComputeClient, apiResponse); err != nil {
		return sdkgo.NetworkLoadBalancerForwardingRule{}, apiResponse, fmt.Errorf(ruleUpdateWaitErr, err)
	}
	return rule, apiResponse, nil
}

// DeleteForwardingRule based on Datacenter ID, NetworkLoadBalancer ID and ForwardingRule ID
func (cp *APIClient) DeleteForwardingRule(ctx context.Context, datacenterID, nlbID, ruleID string) (*sdkgo.APIResponse, error) {
	apiResponse, err := cp.ComputeClient.NetworkLoadBalancersApi.DatacentersNetworkloadbalancersForwardingrulesDelete(ctx, datacenterID, nlbID, ruleID).Execute()
	if err != nil {
		if apiResponse.HttpNotFound() {
			return apiResponse, ErrNotFound
		}
		return apiResponse, fmt.Errorf(ruleDeleteErr, err)
	}
	if err = compute.WaitForRequest(ctx, cp.ComputeClient, apiResponse); err != nil {
		return apiResponse, fmt.Errorf(ruleDeleteWaitErr, err)
	}
	return apiResponse, nil
}

// LateInitializer fills the empty fields in *v1alpha1.ForwardingRuleParameters with
// values that might have been provided by the API in sdkgo.NetworkLoadBalancerForwardingRule
func LateInitializer(in *v1alpha1.ForwardingRuleParameters, rule sdkgo.NetworkLoadBalancerForwardingRule) bool {
	// Don't initialize fields if the API hasn't set anything or
	// values have already been provided in the NetworkLoadBalancerParameters
	var lateInitialized bool
	if rule.Properties == nil {
		return false
	}
	if rule.Properties.HealthCheck != nil {
		if rule.Properties.HealthCheck.ClientTimeout != nil && in.HealthCheck.ClientTimeout == 0 {
			in.HealthCheck.ClientTimeout = *rule.Properties.HealthCheck.ClientTimeout
			lateInitialized = true
		}
		if rule.Properties.HealthCheck.ConnectTimeout != nil && in.HealthCheck.ConnectTimeout == 0 {
			in.HealthCheck.ConnectTimeout = *rule.Properties.HealthCheck.ConnectTimeout
			lateInitialized = true
		}
		if rule.Properties.HealthCheck.TargetTimeout != nil && in.HealthCheck.TargetTimeout == 0 {
			in.HealthCheck.TargetTimeout = *rule.Properties.HealthCheck.TargetTimeout
			lateInitialized = true
		}
		if rule.Properties.HealthCheck.Retries != nil && in.HealthCheck.Retries == 0 {
			in.HealthCheck.Retries = *rule.Properties.HealthCheck.Retries
			lateInitialized = true
		}
	}
	return lateInitialized
}

func SetStatus(in *v1alpha1.ForwardingRuleObservation, rule sdkgo.NetworkLoadBalancerForwardingRule) {
	if rule.Metadata != nil && rule.Metadata.State != nil {
		in.State = *rule.Metadata.State
	}
}

// GenerateCreateInput returns sdkgo.NetworkLoadBalancer for Create requests based on CR spec
func GenerateCreateInput(cr *v1alpha1.ForwardingRule, listenerIP string, targetsIPs map[string]v1alpha1.ForwardingRuleTarget) sdkgo.NetworkLoadBalancerForwardingRule {
	ruleProperties := GenerateUpdateInput(cr, listenerIP, targetsIPs)
	instanceCreateInput := sdkgo.NetworkLoadBalancerForwardingRule{
		Properties: &ruleProperties,
	}
	return instanceCreateInput
}

// GenerateUpdateInput returns sdkgo.NetworkLoadBalancerProperties for Update requests based on CR spec
func GenerateUpdateInput(cr *v1alpha1.ForwardingRule, listenerIP string, targetsIPs map[string]v1alpha1.ForwardingRuleTarget) sdkgo.NetworkLoadBalancerForwardingRuleProperties {
	instanceUpdateInput := sdkgo.NetworkLoadBalancerForwardingRuleProperties{
		Name:         &cr.Spec.ForProvider.Name,
		Algorithm:    &cr.Spec.ForProvider.Algorithm,
		Protocol:     &cr.Spec.ForProvider.Protocol,
		ListenerIp:   &listenerIP,
		ListenerPort: &cr.Spec.ForProvider.ListenerPort,
		HealthCheck:  ruleHealthCheckInput(cr.Spec.ForProvider.HealthCheck),
		Targets:      ruleTargetsInput(targetsIPs),
	}

	return instanceUpdateInput
}

func IsUpToDate(cr *v1alpha1.ForwardingRule, observed sdkgo.NetworkLoadBalancerForwardingRule, listenerIp string, targetsIPs map[string]v1alpha1.ForwardingRuleTarget) bool { // nolint:gocyclo
	switch {
	case cr == nil && observed.Properties == nil:
		return true
	case cr == nil && observed.Properties != nil:
		return false
	case cr != nil && observed.Properties == nil:
		return false
	case observed.Metadata != nil && observed.Metadata.State != nil && (*observed.Metadata.State == compute.BUSY || *observed.Metadata.State == compute.UPDATING):
		return true
	case observed.Properties.Name != nil && *observed.Properties.Name != cr.Spec.ForProvider.Name:
		return false
	case observed.Properties.Name == nil && cr.Spec.ForProvider.Name != "":
		return false
	case observed.Properties.Algorithm != nil && *observed.Properties.Algorithm != cr.Spec.ForProvider.Algorithm:
		return false
	case observed.Properties.Protocol != nil && *observed.Properties.Protocol != cr.Spec.ForProvider.Protocol:
		return false
	case observed.Properties.ListenerIp != nil && *observed.Properties.ListenerIp != listenerIp:
		return false
	case observed.Properties.ListenerPort != nil && *observed.Properties.ListenerPort != cr.Spec.ForProvider.ListenerPort:
		return false
	case !equalRuleHealthCheck(cr.Spec.ForProvider.HealthCheck, observed.Properties.HealthCheck):
		return false
	case !equalTargets(targetsIPs, observed.Properties.Targets):
		return false
	}

	return true
}

func equalRuleHealthCheck(cr v1alpha1.ForwardingRuleHealthCheck, observed *sdkgo.NetworkLoadBalancerForwardingRuleHealthCheck) bool {
	if observed == nil {
		return true
	}
	switch {
	case observed.Retries != nil && *observed.Retries != cr.Retries:
		return false
	case observed.ClientTimeout != nil && *observed.ClientTimeout != cr.ClientTimeout:
		return false
	case observed.ConnectTimeout != nil && *observed.ConnectTimeout != cr.ConnectTimeout:
		return false
	case observed.TargetTimeout != nil && *observed.TargetTimeout != cr.TargetTimeout:
		return false
	}
	return true
}

func equalTargets(configured map[string]v1alpha1.ForwardingRuleTarget, observed *[]sdkgo.NetworkLoadBalancerForwardingRuleTarget) bool {
	if observed == nil {
		return len(configured) == 0
	} else if len(*observed) != len(configured) {
		return false
	}

	for _, obsTarget := range *observed {
		if obsTarget.Ip == nil {
			continue
		}
		cfgTarget, ok := configured[*obsTarget.Ip]
		if !ok {
			return false
		}
		if !equalTarget(cfgTarget, &obsTarget, *obsTarget.Ip) {
			return false
		}
	}

	return true
}

func equalTarget(cr v1alpha1.ForwardingRuleTarget, observed *sdkgo.NetworkLoadBalancerForwardingRuleTarget, ip string) bool {
	if observed == nil {
		return true
	}
	switch {
	case observed.Ip != nil && *observed.Ip != ip:
		return false
	case observed.Port != nil && *observed.Port != cr.Port:
		return false
	case observed.Weight != nil && *observed.Weight != cr.Weight:
		return false
	case observed.ProxyProtocol != nil && *observed.ProxyProtocol != cr.ProxyProtocol:
		return false
	case !equalTargetHealthCheck(cr.HealthCheck, observed.HealthCheck):
		return false
	}
	return true
}

func equalTargetHealthCheck(cr v1alpha1.ForwardingRuleTargetHealthCheck, observed *sdkgo.NetworkLoadBalancerForwardingRuleTargetHealthCheck) bool {
	if observed == nil {
		return true
	}
	switch {
	case observed.Check != nil && *observed.Check != cr.Check:
		return false
	case observed.CheckInterval != nil && *observed.CheckInterval != cr.CheckInterval:
		return false
	case observed.Maintenance != nil && *observed.Maintenance != cr.Maintenance:
		return false
	}
	return true
}

func ruleHealthCheckInput(cr v1alpha1.ForwardingRuleHealthCheck) *sdkgo.NetworkLoadBalancerForwardingRuleHealthCheck {
	return &sdkgo.NetworkLoadBalancerForwardingRuleHealthCheck{
		Retries:        &cr.Retries,
		ClientTimeout:  &cr.ClientTimeout,
		ConnectTimeout: &cr.ConnectTimeout,
		TargetTimeout:  &cr.TargetTimeout,
	}
}

func ruleTargetsInput(targetsIPs map[string]v1alpha1.ForwardingRuleTarget) *[]sdkgo.NetworkLoadBalancerForwardingRuleTarget {
	targetsInput := make([]sdkgo.NetworkLoadBalancerForwardingRuleTarget, len(targetsIPs))
	for k, v := range targetsIPs {
		target := sdkgo.NetworkLoadBalancerForwardingRuleTarget{
			Ip:            &k,
			Port:          &v.Port,
			Weight:        &v.Weight,
			ProxyProtocol: &v.ProxyProtocol,
			HealthCheck: &sdkgo.NetworkLoadBalancerForwardingRuleTargetHealthCheck{
				Check:         &v.HealthCheck.Check,
				CheckInterval: &v.HealthCheck.CheckInterval,
				Maintenance:   &v.HealthCheck.Maintenance,
			},
		}
		targetsInput = append(targetsInput, target)
	}
	return &targetsInput
}
