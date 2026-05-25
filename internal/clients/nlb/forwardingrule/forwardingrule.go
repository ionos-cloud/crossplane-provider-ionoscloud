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
	ruleDeleteWaitErr = "error while waiting for nlb forwarding rule delete request: %w"
)

var (
	zeroRuleHealthCheck   = v1alpha1.ForwardingRuleHealthCheck{}
	zeroTargetHealthCheck = v1alpha1.ForwardingRuleTargetHealthCheck{}
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
	GetForwardingRuleByID(ctx context.Context, datacenterID, nlbID, ruleID string) (sdkgo.NetworkLoadBalancerForwardingRule, error)
	CreateForwardingRule(ctx context.Context, datacenterID, nlbID string, rule sdkgo.NetworkLoadBalancerForwardingRule) (sdkgo.NetworkLoadBalancerForwardingRule, error)
	UpdateForwardingRule(ctx context.Context, datacenterID, nlbID, ruleID string, ruleProperties sdkgo.NetworkLoadBalancerForwardingRuleProperties) (sdkgo.NetworkLoadBalancerForwardingRule, error)
	DeleteForwardingRule(ctx context.Context, datacenterID, nlbID, ruleID string) error
}

// CheckDuplicateForwardingRule returns the ID of the duplicate Forwarding Rule if any,
// or an error if multiple Forwarding Rules with the same name are found
func (cp *APIClient) CheckDuplicateForwardingRule(ctx context.Context, datacenterID, nlbID, ruleName string) (string, error) {
	ForwardingRules, _, err := cp.IonosServices.ComputeClient.NetworkLoadBalancersApi.DatacentersNetworkloadbalancersForwardingrulesGet(ctx, datacenterID, nlbID).Depth(utils.DepthQueryParam).Execute()
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
func (cp *APIClient) GetForwardingRuleByID(ctx context.Context, datacenterID, nlbID, ruleID string) (sdkgo.NetworkLoadBalancerForwardingRule, error) {
	rule, apiResponse, err := cp.IonosServices.ComputeClient.NetworkLoadBalancersApi.DatacentersNetworkloadbalancersForwardingrulesFindByForwardingRuleId(ctx, datacenterID, nlbID, ruleID).Depth(utils.DepthQueryParam).Execute()
	if err != nil {
		err = ErrNotFound
		if !apiResponse.HttpNotFound() {
			err = fmt.Errorf(ruleGetByIDErr, err)
		}
	}
	return rule, err
}

// CreateForwardingRule based on Datacenter ID, NetworkLoadBalancer ID and ForwardingRule
func (cp *APIClient) CreateForwardingRule(ctx context.Context, datacenterID, nlbID string, rule sdkgo.NetworkLoadBalancerForwardingRule) (sdkgo.NetworkLoadBalancerForwardingRule, error) {
	rule, apiResponse, err := cp.IonosServices.ComputeClient.NetworkLoadBalancersApi.DatacentersNetworkloadbalancersForwardingrulesPost(ctx, datacenterID, nlbID).NetworkLoadBalancerForwardingRule(rule).Execute()
	if err != nil {
		return sdkgo.NetworkLoadBalancerForwardingRule{}, fmt.Errorf(ruleCreateErr, err)
	}
	if err = compute.WaitForRequest(ctx, cp.IonosServices.ComputeClient, apiResponse); err != nil {
		return sdkgo.NetworkLoadBalancerForwardingRule{}, fmt.Errorf(ruleCreateWaitErr, err)
	}
	return rule, err
}

// UpdateForwardingRule based on Datacenter ID, NetworkLoadBalancer ID, ForwardingRule ID and ForwardingRule
func (cp *APIClient) UpdateForwardingRule(ctx context.Context, datacenterID, nlbID, ruleID string, ruleProperties sdkgo.NetworkLoadBalancerForwardingRuleProperties) (sdkgo.NetworkLoadBalancerForwardingRule, error) {
	rule, apiResponse, err := cp.IonosServices.ComputeClient.NetworkLoadBalancersApi.DatacentersNetworkloadbalancersForwardingrulesPatch(ctx, datacenterID, nlbID, ruleID).NetworkLoadBalancerForwardingRuleProperties(ruleProperties).Execute()
	if err != nil {
		return sdkgo.NetworkLoadBalancerForwardingRule{}, fmt.Errorf(ruleUpdateErr, err)
	}
	if err = compute.WaitForRequest(ctx, cp.IonosServices.ComputeClient, apiResponse); err != nil {
		return sdkgo.NetworkLoadBalancerForwardingRule{}, fmt.Errorf(ruleUpdateWaitErr, err)
	}
	return rule, nil
}

// DeleteForwardingRule based on Datacenter ID, NetworkLoadBalancer ID and ForwardingRule ID
func (cp *APIClient) DeleteForwardingRule(ctx context.Context, datacenterID, nlbID, ruleID string) error {
	apiResponse, err := cp.IonosServices.ComputeClient.NetworkLoadBalancersApi.DatacentersNetworkloadbalancersForwardingrulesDelete(ctx, datacenterID, nlbID, ruleID).Execute()
	if err != nil {
		if apiResponse.HttpNotFound() {
			return ErrNotFound
		}
		return fmt.Errorf(ruleDeleteErr, err)
	}
	if err = compute.WaitForRequest(ctx, cp.IonosServices.ComputeClient, apiResponse); err != nil {
		return fmt.Errorf(ruleDeleteWaitErr, err)
	}
	return nil
}

// SetStatus sets fields of the ForwardingRuleObservation based on sdkgo.NetworkLoadBalancerForwardingRule
func SetStatus(in *v1alpha1.ForwardingRuleObservation, rule sdkgo.NetworkLoadBalancerForwardingRule) {
	if rule.Metadata != nil && rule.Metadata.State != nil {
		in.State = *rule.Metadata.State
	}
	if rule.Properties != nil && rule.Properties.ListenerIp != nil && rule.Properties.ListenerPort != nil {
		in.ListenerIP = *rule.Properties.ListenerIp
		in.ListenerPort = *rule.Properties.ListenerPort
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


func ruleHealthCheckInput(cr v1alpha1.ForwardingRuleHealthCheck) *sdkgo.NetworkLoadBalancerForwardingRuleHealthCheck {
	// Don't include 0-value rule health check
	if cr == zeroRuleHealthCheck {
		return nil
	}
	return &sdkgo.NetworkLoadBalancerForwardingRuleHealthCheck{
		Retries:        &cr.Retries,
		ClientTimeout:  &cr.ClientTimeout,
		ConnectTimeout: &cr.ConnectTimeout,
		TargetTimeout:  &cr.TargetTimeout,
	}
}

func ruleTargetsInput(targetsIPs map[string]v1alpha1.ForwardingRuleTarget) *[]sdkgo.NetworkLoadBalancerForwardingRuleTarget {
	targetsInput := make([]sdkgo.NetworkLoadBalancerForwardingRuleTarget, 0, len(targetsIPs))
	for k, v := range targetsIPs {
		k := k
		v := v
		target := sdkgo.NetworkLoadBalancerForwardingRuleTarget{
			Ip:            &k,
			Port:          &v.Port,
			Weight:        &v.Weight,
			ProxyProtocol: &v.ProxyProtocol,
		}
		// Don't include 0-value target health check
		if v.HealthCheck != zeroTargetHealthCheck {
			target.HealthCheck = &sdkgo.NetworkLoadBalancerForwardingRuleTargetHealthCheck{
				Check:         &v.HealthCheck.Check,
				CheckInterval: &v.HealthCheck.CheckInterval,
				Maintenance:   &v.HealthCheck.Maintenance,
			}
		}

		targetsInput = append(targetsInput, target)
	}
	return &targetsInput
}
