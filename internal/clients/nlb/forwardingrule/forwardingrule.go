package forwardingrule

import (
	"context"
	"fmt"

	sdkgo "github.com/ionos-cloud/sdk-go/v6"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/nlb/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"
)

// APIClient is a wrapper around IONOS Service
type APIClient struct {
	*clients.IonosServices
}

// Client is a wrapper around IONOS Service Network Load Balancer methods
type Client interface {
	CheckDuplicateForwardingRule(ctx context.Context, datacenterID, nlbName string) (string, error)
	GetForwardingRuleByID(ctx context.Context, datacenterID, ruleID string) (sdkgo.NetworkLoadBalancerForwardingRule, *sdkgo.APIResponse, error)
	CreateForwardingRule(ctx context.Context, datacenterID string, nlb sdkgo.NetworkLoadBalancerForwardingRule) (sdkgo.NetworkLoadBalancerForwardingRule, *sdkgo.APIResponse, error)
	UpdateForwardingRule(ctx context.Context, datacenterID, nlbID string, nlbProperties sdkgo.NetworkLoadBalancerForwardingRuleProperties) (sdkgo.NetworkLoadBalancerForwardingRule, *sdkgo.APIResponse, error)
	DeleteForwardingRule(ctx context.Context, datacenterID, nlbID string) (*sdkgo.APIResponse, error)
}

// CheckDuplicateForwardingRule returns the ID of the duplicate Forwarding Rule if any,
// or an error if multiple Forwarding Rules with the same name are found
func (cp *APIClient) CheckDuplicateForwardingRule(ctx context.Context, datacenterID, nlbID, fwRuleName string) (string, error) {
	ForwardingRules, _, err := cp.ComputeClient.NetworkLoadBalancersApi.DatacentersNetworkloadbalancersForwardingrulesGet(ctx, datacenterID, nlbID).Depth(utils.DepthQueryParam).Execute()
	if err != nil {
		return "", err
	}

	matchedItems := make([]sdkgo.NetworkLoadBalancerForwardingRule, 0)

	if ForwardingRules.Items != nil {
		for _, item := range *ForwardingRules.Items {
			if item.Properties != nil && item.Properties.Name != nil && *item.Properties.Name == fwRuleName {
				matchedItems = append(matchedItems, item)
			}
		}
	}

	if len(matchedItems) == 0 {
		return "", nil
	}
	if len(matchedItems) > 1 {
		return "", fmt.Errorf("error: found multiple Forwarding Rules with the name %v", fwRuleName)
	}
	if matchedItems[0].Id == nil {
		return "", fmt.Errorf("error getting ID for Forwarding Rule named: %v", fwRuleName)
	}
	return *matchedItems[0].Id, nil
}

// GetForwardingRuleByID based on Datacenter ID, NetworkLoadBalancer ID and ForwardingRule ID
func (cp *APIClient) GetForwardingRuleByID(ctx context.Context, datacenterID, nlbID, ruleID string) (sdkgo.NetworkLoadBalancerForwardingRule, *sdkgo.APIResponse, error) {
	return cp.ComputeClient.NetworkLoadBalancersApi.DatacentersNetworkloadbalancersForwardingrulesFindByForwardingRuleId(ctx, datacenterID, nlbID, ruleID).Depth(utils.DepthQueryParam).Execute()
}

// CreateForwardingRule based on Datacenter ID, NetworkLoadBalancer ID and ForwardingRule
func (cp *APIClient) CreateForwardingRule(ctx context.Context, datacenterID, nlbID string, rule sdkgo.NetworkLoadBalancerForwardingRule) (sdkgo.NetworkLoadBalancerForwardingRule, *sdkgo.APIResponse, error) {
	return cp.ComputeClient.NetworkLoadBalancersApi.DatacentersNetworkloadbalancersForwardingrulesPost(ctx, datacenterID, nlbID).NetworkLoadBalancerForwardingRule(rule).Execute()
}

// UpdateForwardingRule based on Datacenter ID, NetworkLoadBalancer ID, ForwardingRule ID and ForwardingRule
func (cp *APIClient) UpdateForwardingRule(ctx context.Context, datacenterID, nlbID, ruleID string, rule sdkgo.NetworkLoadBalancerForwardingRulePut) (sdkgo.NetworkLoadBalancerForwardingRule, *sdkgo.APIResponse, error) {
	return cp.ComputeClient.NetworkLoadBalancersApi.DatacentersNetworkloadbalancersForwardingrulesPut(ctx, datacenterID, nlbID, ruleID).NetworkLoadBalancerForwardingRule(rule).Execute()
}

// DeleteForwardingRule based on Datacenter ID, NetworkLoadBalancer ID and ForwardingRule ID
func (cp *APIClient) DeleteForwardingRule(ctx context.Context, datacenterID, nlbID, ruleID string) (*sdkgo.APIResponse, error) {
	return cp.ComputeClient.NetworkLoadBalancersApi.DatacentersNetworkloadbalancersForwardingrulesDelete(ctx, datacenterID, nlbID, ruleID).Execute()
}

func IsUpToDate(cr *v1alpha1.ForwardingRule, observed sdkgo.NetworkLoadBalancerForwardingRule) bool { // nolint:gocyclo
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
	}

	return true
}
