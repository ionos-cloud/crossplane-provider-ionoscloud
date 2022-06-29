package forwardingrule

import (
	"context"
	"reflect"

	sdkgo "github.com/ionos-cloud/sdk-go/v6"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/alb/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"
)

// APIClient is a wrapper around IONOS Service
type APIClient struct {
	*clients.IonosServices
}

// Client is a wrapper around IONOS Service ApplicationLoadBalancer methods
type Client interface {
	GetForwardingRule(ctx context.Context, datacenterID, applicationloadbalancerID, forwardingruleID string) (sdkgo.ApplicationLoadBalancerForwardingRule, *sdkgo.APIResponse, error)
	CreateForwardingRule(ctx context.Context, datacenterID, applicationloadbalancerID string, forwardingrule sdkgo.ApplicationLoadBalancerForwardingRule) (sdkgo.ApplicationLoadBalancerForwardingRule, *sdkgo.APIResponse, error)
	UpdateForwardingRule(ctx context.Context, datacenterID, applicationloadbalancerID, forwardingruleID string, forwardingrule sdkgo.ApplicationLoadBalancerForwardingRulePut) (sdkgo.ApplicationLoadBalancerForwardingRule, *sdkgo.APIResponse, error)
	DeleteForwardingRule(ctx context.Context, datacenterID, applicationloadbalancerID, forwardingruleID string) (*sdkgo.APIResponse, error)
	GetAPIClient() *sdkgo.APIClient
}

// GetForwardingRule based on datacenterID, applicationloadbalancerID, forwardingruleID
func (cp *APIClient) GetForwardingRule(ctx context.Context, datacenterID, applicationloadbalancerID, forwardingruleID string) (sdkgo.ApplicationLoadBalancerForwardingRule, *sdkgo.APIResponse, error) {
	return cp.ComputeClient.ApplicationLoadBalancersApi.DatacentersApplicationloadbalancersForwardingrulesFindByForwardingRuleId(ctx, datacenterID, applicationloadbalancerID, forwardingruleID).Depth(utils.DepthQueryParam).Execute()
}

// CreateForwardingRule based on datacenterID, applicationloadbalancerID, ApplicationLoadBalancerForwardingRule
func (cp *APIClient) CreateForwardingRule(ctx context.Context, datacenterID, applicationloadbalancerID string, forwardingrule sdkgo.ApplicationLoadBalancerForwardingRule) (sdkgo.ApplicationLoadBalancerForwardingRule, *sdkgo.APIResponse, error) {
	return cp.ComputeClient.ApplicationLoadBalancersApi.DatacentersApplicationloadbalancersForwardingrulesPost(ctx, datacenterID, applicationloadbalancerID).ApplicationLoadBalancerForwardingRule(forwardingrule).Execute()
}

// UpdateForwardingRule based on datacenterID, applicationloadbalancerID, forwardingruleID and ApplicationLoadBalancerForwardingRulePut
func (cp *APIClient) UpdateForwardingRule(ctx context.Context, datacenterID, applicationloadbalancerID, forwardingruleID string, applicationloadbalancer sdkgo.ApplicationLoadBalancerForwardingRulePut) (sdkgo.ApplicationLoadBalancerForwardingRule, *sdkgo.APIResponse, error) {
	return cp.ComputeClient.ApplicationLoadBalancersApi.DatacentersApplicationloadbalancersForwardingrulesPut(ctx, datacenterID, applicationloadbalancerID, forwardingruleID).ApplicationLoadBalancerForwardingRule(applicationloadbalancer).Execute()
}

// DeleteForwardingRule based on datacenterID, applicationloadbalancerID, forwardingruleID
func (cp *APIClient) DeleteForwardingRule(ctx context.Context, datacenterID, applicationloadbalancerID, forwardingruleID string) (*sdkgo.APIResponse, error) {
	resp, err := cp.ComputeClient.ApplicationLoadBalancersApi.DatacentersApplicationloadbalancersForwardingrulesDelete(ctx, datacenterID, applicationloadbalancerID, forwardingruleID).Execute()
	return resp, err
}

// GetAPIClient gets the APIClient
func (cp *APIClient) GetAPIClient() *sdkgo.APIClient {
	return cp.ComputeClient
}

// GenerateCreateForwardingRuleInput returns sdkgo.ApplicationLoadBalancer based on the CR spec
func GenerateCreateForwardingRuleInput(cr *v1alpha1.ForwardingRule, listenerIP string) (*sdkgo.ApplicationLoadBalancerForwardingRule, error) {
	instanceCreateInput := sdkgo.ApplicationLoadBalancerForwardingRule{
		Properties: &sdkgo.ApplicationLoadBalancerForwardingRuleProperties{
			Name:         &cr.Spec.ForProvider.Name,
			Protocol:     &cr.Spec.ForProvider.Protocol,
			ListenerIp:   &listenerIP,
			ListenerPort: &cr.Spec.ForProvider.ListenerPort,
		},
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.ClientTimeout)) {
		instanceCreateInput.Properties.SetClientTimeout(cr.Spec.ForProvider.ClientTimeout)
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.ServerCertificatesIDs)) {
		instanceCreateInput.Properties.SetServerCertificates(cr.Spec.ForProvider.ServerCertificatesIDs)
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.HTTPRules)) {
		instanceCreateInput.Properties.SetHttpRules(getHTTPRules(cr.Spec.ForProvider.HTTPRules))
	}
	return &instanceCreateInput, nil
}

// GenerateUpdateForwardingRuleInput returns sdkgo.ApplicationLoadBalancerProperties based on the CR spec modifications
func GenerateUpdateForwardingRuleInput(cr *v1alpha1.ForwardingRule, listenerIP string) (*sdkgo.ApplicationLoadBalancerForwardingRulePut, error) {
	instanceUpdateInput := sdkgo.ApplicationLoadBalancerForwardingRulePut{
		Properties: &sdkgo.ApplicationLoadBalancerForwardingRuleProperties{
			Name:         &cr.Spec.ForProvider.Name,
			Protocol:     &cr.Spec.ForProvider.Protocol,
			ListenerIp:   &listenerIP,
			ListenerPort: &cr.Spec.ForProvider.ListenerPort,
		},
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.ClientTimeout)) {
		instanceUpdateInput.Properties.SetClientTimeout(cr.Spec.ForProvider.ClientTimeout)
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.ServerCertificatesIDs)) {
		instanceUpdateInput.Properties.SetServerCertificates(cr.Spec.ForProvider.ServerCertificatesIDs)
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.HTTPRules)) {
		instanceUpdateInput.Properties.SetHttpRules(getHTTPRules(cr.Spec.ForProvider.HTTPRules))
	}
	return &instanceUpdateInput, nil
}

// LateInitializer fills the empty fields in *v1alpha1.ApplicationLoadBalancerParameters with
// the values seen in sdkgo.ApplicationLoadBalancer.
func LateInitializer(in *v1alpha1.ForwardingRuleParameters, forwardingRule *sdkgo.ApplicationLoadBalancerForwardingRule) { // nolint:gocyclo
	if forwardingRule == nil {
		return
	}
	// Add Properties to the Spec, if they were set by the API
	if propertiesOk, ok := forwardingRule.GetPropertiesOk(); ok && propertiesOk != nil {
		if clientTimeoutOk, ok := propertiesOk.GetClientTimeoutOk(); ok && clientTimeoutOk != nil {
			if utils.IsEmptyValue(reflect.ValueOf(in.ClientTimeout)) {
				in.ClientTimeout = *clientTimeoutOk
			}
		}
	}
}

// IsForwardingRuleUpToDate returns true if the ApplicationLoadBalancer is up-to-date or false if it does not
func IsForwardingRuleUpToDate(cr *v1alpha1.ForwardingRule, forwardingRule sdkgo.ApplicationLoadBalancerForwardingRule, listenerIP string) bool { // nolint:gocyclo
	switch {
	case cr == nil && forwardingRule.Properties == nil:
		return true
	case cr == nil && forwardingRule.Properties != nil:
		return false
	case cr != nil && forwardingRule.Properties == nil:
		return false
	case forwardingRule.Metadata.State != nil && *forwardingRule.Metadata.State == "BUSY" || *forwardingRule.Metadata.State == "DEPLOYING":
		return true
	case forwardingRule.Properties.Name != nil && *forwardingRule.Properties.Name != cr.Spec.ForProvider.Name:
		return false
	case forwardingRule.Properties.Protocol != nil && *forwardingRule.Properties.Protocol != cr.Spec.ForProvider.Protocol:
		return false
	case forwardingRule.Properties.ListenerIp != nil && *forwardingRule.Properties.ListenerIp != listenerIP:
		return false
	case forwardingRule.Properties.ListenerPort != nil && *forwardingRule.Properties.ListenerPort != cr.Spec.ForProvider.ListenerPort:
		return false
	case forwardingRule.Properties.ClientTimeout != nil && *forwardingRule.Properties.ClientTimeout != cr.Spec.ForProvider.ClientTimeout:
		return false
	case forwardingRule.Properties.ServerCertificates != nil && !utils.ContainsStringSlices(*forwardingRule.Properties.ServerCertificates, cr.Spec.ForProvider.ServerCertificatesIDs):
		return false
	case !equalHTTPRules(cr.Spec.ForProvider.HTTPRules, forwardingRule.Properties.HttpRules):
		return false
	default:
		return true
	}
}

func getHTTPRules(httpRules []v1alpha1.ApplicationLoadBalancerHTTPRule) []sdkgo.ApplicationLoadBalancerHttpRule {
	if len(httpRules) == 0 {
		return nil
	}
	applicationLoadBalancerHTTPRules := make([]sdkgo.ApplicationLoadBalancerHttpRule, len(httpRules))
	for i, rule := range httpRules {
		applicationLoadBalancerHTTPRules[i] = sdkgo.ApplicationLoadBalancerHttpRule{
			Name: sdkgo.PtrString(rule.Name),
			Type: sdkgo.PtrString(rule.Type),
		}
		if !utils.IsEmptyValue(reflect.ValueOf(rule.TargetGroupCfg.TargetGroupID)) {
			applicationLoadBalancerHTTPRules[i].SetTargetGroup(rule.TargetGroupCfg.TargetGroupID)
		}
		if !utils.IsEmptyValue(reflect.ValueOf(rule.DropQuery)) {
			applicationLoadBalancerHTTPRules[i].SetDropQuery(rule.DropQuery)
		}
		if !utils.IsEmptyValue(reflect.ValueOf(rule.Location)) {
			applicationLoadBalancerHTTPRules[i].SetLocation(rule.Location)
		}
		if !utils.IsEmptyValue(reflect.ValueOf(rule.StatusCode)) {
			applicationLoadBalancerHTTPRules[i].SetStatusCode(rule.StatusCode)
		}
		if !utils.IsEmptyValue(reflect.ValueOf(rule.ResponseMessage)) {
			applicationLoadBalancerHTTPRules[i].SetResponseMessage(rule.ResponseMessage)
		}
		if !utils.IsEmptyValue(reflect.ValueOf(rule.ContentType)) {
			applicationLoadBalancerHTTPRules[i].SetContentType(rule.ContentType)
		}
		if !utils.IsEmptyValue(reflect.ValueOf(rule.Conditions)) {
			applicationLoadBalancerHTTPRules[i].SetConditions(getHTTPRuleConditions(rule.Conditions))
		}
	}
	return applicationLoadBalancerHTTPRules
}

func getHTTPRuleConditions(conditions []v1alpha1.ApplicationLoadBalancerHTTPRuleCondition) []sdkgo.ApplicationLoadBalancerHttpRuleCondition {
	if len(conditions) == 0 {
		return nil
	}
	httpRuleConditions := make([]sdkgo.ApplicationLoadBalancerHttpRuleCondition, len(conditions))
	for i, condition := range conditions {
		httpRuleConditions[i] = sdkgo.ApplicationLoadBalancerHttpRuleCondition{
			Type:      sdkgo.PtrString(condition.Type),
			Condition: sdkgo.PtrString(condition.Condition),
			Negate:    sdkgo.PtrBool(condition.Negate),
		}
		if !utils.IsEmptyValue(reflect.ValueOf(condition.Key)) {
			httpRuleConditions[i].SetKey(condition.Key)
		}
		if !utils.IsEmptyValue(reflect.ValueOf(condition.Value)) {
			httpRuleConditions[i].SetValue(condition.Value)
		}
	}
	return httpRuleConditions
}

func equalHTTPRules(httpRules []v1alpha1.ApplicationLoadBalancerHTTPRule, albHTTPRules *[]sdkgo.ApplicationLoadBalancerHttpRule) bool {
	if albHTTPRules == nil && len(httpRules) == 0 {
		return true
	}
	if albHTTPRules == nil && len(httpRules) != 0 {
		return false
	}
	if len(httpRules) != len(*albHTTPRules) {
		return false
	}
	for _, httpRule := range httpRules {
		if !equalHTTPRule(httpRule, *albHTTPRules) {
			return false
		}
	}
	return true
}

func equalHTTPRule(target v1alpha1.ApplicationLoadBalancerHTTPRule, targets []sdkgo.ApplicationLoadBalancerHttpRule) bool { // nolint: gocyclo
	if len(targets) == 0 {
		return false
	}
	for _, t := range targets {
		if t.HasName() && *t.Name == target.Name {
			if t.HasType() && *t.Type != target.Type {
				return false
			}
			if t.HasTargetGroup() && *t.TargetGroup != target.TargetGroupCfg.TargetGroupID {
				return false
			}
			if t.HasDropQuery() && *t.DropQuery != target.DropQuery {
				return false
			}
			if t.HasLocation() && *t.Location != target.Location {
				return false
			}
			if t.HasStatusCode() && *t.StatusCode != target.StatusCode {
				return false
			}
			if t.HasResponseMessage() && *t.ResponseMessage != target.ResponseMessage {
				return false
			}
			if t.HasContentType() && *t.ContentType != target.ContentType {
				return false
			}
			if !equalHTTPRuleConditions(target.Conditions, t.Conditions) {
				return false
			}
			return true
		}
	}
	return false
}

func equalHTTPRuleConditions(httpRuleConditions []v1alpha1.ApplicationLoadBalancerHTTPRuleCondition, albHTTPRuleConditions *[]sdkgo.ApplicationLoadBalancerHttpRuleCondition) bool {
	if albHTTPRuleConditions == nil && len(httpRuleConditions) == 0 {
		return true
	}
	if albHTTPRuleConditions == nil && len(httpRuleConditions) != 0 {
		return false
	}
	if len(httpRuleConditions) != len(*albHTTPRuleConditions) {
		return false
	}
	for _, httpRule := range httpRuleConditions {
		if !equalHTTPRuleCondition(httpRule, *albHTTPRuleConditions) {
			return false
		}
	}
	return true
}

func equalHTTPRuleCondition(condition v1alpha1.ApplicationLoadBalancerHTTPRuleCondition, conditions []sdkgo.ApplicationLoadBalancerHttpRuleCondition) bool { // nolint: gocyclo
	if len(conditions) == 0 {
		return false
	}
	for _, ruleCondition := range conditions {
		// Type and Condition are required
		if ruleCondition.HasType() && ruleCondition.HasCondition() {
			if *ruleCondition.Type == condition.Type && *ruleCondition.Condition == condition.Condition {
				if ruleCondition.HasNegate() && *ruleCondition.Negate != condition.Negate {
					return false
				}
				if ruleCondition.HasKey() && *ruleCondition.Key != condition.Key {
					return false
				}
				if ruleCondition.HasValue() && *ruleCondition.Value != condition.Value {
					return false
				}
				return true
			}
		}
	}
	return false
}
