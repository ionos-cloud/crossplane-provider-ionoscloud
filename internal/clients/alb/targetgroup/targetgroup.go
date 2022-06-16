package targetgroup

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

// Client is a wrapper around IONOS Service TargetGroup methods
type Client interface {
	GetTargetGroup(ctx context.Context, targetGroupID string) (sdkgo.TargetGroup, *sdkgo.APIResponse, error)
	CreateTargetGroup(ctx context.Context, targetGroup sdkgo.TargetGroup) (sdkgo.TargetGroup, *sdkgo.APIResponse, error)
	UpdateTargetGroup(ctx context.Context, targetGroupID string, targetGroup sdkgo.TargetGroupPut) (sdkgo.TargetGroup, *sdkgo.APIResponse, error)
	DeleteTargetGroup(ctx context.Context, targetGroupID string) (*sdkgo.APIResponse, error)
	GetAPIClient() *sdkgo.APIClient
}

// GetTargetGroup based on targetGroupID
func (cp *APIClient) GetTargetGroup(ctx context.Context, targetGroupID string) (sdkgo.TargetGroup, *sdkgo.APIResponse, error) {
	return cp.ComputeClient.TargetGroupsApi.TargetgroupsFindByTargetGroupId(ctx, targetGroupID).Depth(utils.DepthQueryParam).Execute()
}

// CreateTargetGroup based on TargetGroup
func (cp *APIClient) CreateTargetGroup(ctx context.Context, targetGroup sdkgo.TargetGroup) (sdkgo.TargetGroup, *sdkgo.APIResponse, error) {
	return cp.ComputeClient.TargetGroupsApi.TargetgroupsPost(ctx).TargetGroup(targetGroup).Execute()
}

// UpdateTargetGroup based on targetGroupID and TargetGroupProperties
func (cp *APIClient) UpdateTargetGroup(ctx context.Context, targetGroupID string, targetGroup sdkgo.TargetGroupPut) (sdkgo.TargetGroup, *sdkgo.APIResponse, error) {
	return cp.ComputeClient.TargetGroupsApi.TargetgroupsPut(ctx, targetGroupID).TargetGroup(targetGroup).Execute()
}

// DeleteTargetGroup based on targetGroupID
func (cp *APIClient) DeleteTargetGroup(ctx context.Context, targetGroupID string) (*sdkgo.APIResponse, error) {
	return cp.ComputeClient.TargetGroupsApi.TargetGroupsDelete(ctx, targetGroupID).Execute()
}

// GetAPIClient gets the APIClient
func (cp *APIClient) GetAPIClient() *sdkgo.APIClient {
	return cp.ComputeClient
}

// GetTargetGroupTargets set by the user based on targetGroupID
func (cp *APIClient) GetTargetGroupTargets(ctx context.Context, targetGroupID string) (sdkgo.TargetGroup, *sdkgo.APIResponse, error) {
	return cp.ComputeClient.TargetGroupsApi.TargetgroupsFindByTargetGroupId(ctx, targetGroupID).Depth(utils.DepthQueryParam).Execute()
}

// GenerateCreateTargetGroupInput returns sdkgo.TargetGroup based on the CR spec
func GenerateCreateTargetGroupInput(cr *v1alpha1.TargetGroup) (*sdkgo.TargetGroup, error) {
	instanceCreateInput := sdkgo.TargetGroup{
		Properties: &sdkgo.TargetGroupProperties{
			Name:      &cr.Spec.ForProvider.Name,
			Algorithm: &cr.Spec.ForProvider.Algorithm,
			Protocol:  &cr.Spec.ForProvider.Protocol,
		},
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.Targets)) {
		instanceCreateInput.Properties.SetTargets(getTargets(cr.Spec.ForProvider.Targets))
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.HealthCheck)) {
		instanceCreateInput.Properties.SetHealthCheck(getHealthCheck(cr.Spec.ForProvider.HealthCheck))
	} else {
		instanceCreateInput.Properties.HealthCheck = nil
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.HTTPHealthCheck)) {
		instanceCreateInput.Properties.SetHttpHealthCheck(getHTTPHealthCheck(cr.Spec.ForProvider.HTTPHealthCheck))
	} else {
		instanceCreateInput.Properties.HttpHealthCheck = nil
	}
	return &instanceCreateInput, nil
}

// GenerateUpdateTargetGroupInput returns sdkgo.TargetGroupProperties based on the CR spec modifications
func GenerateUpdateTargetGroupInput(cr *v1alpha1.TargetGroup) (*sdkgo.TargetGroupPut, error) {
	instanceUpdateInput := sdkgo.TargetGroupPut{
		Properties: &sdkgo.TargetGroupProperties{
			Name:      &cr.Spec.ForProvider.Name,
			Algorithm: &cr.Spec.ForProvider.Algorithm,
			Protocol:  &cr.Spec.ForProvider.Protocol,
		},
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.Targets)) {
		instanceUpdateInput.Properties.SetTargets(getTargets(cr.Spec.ForProvider.Targets))
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.HealthCheck)) {
		instanceUpdateInput.Properties.SetHealthCheck(getHealthCheck(cr.Spec.ForProvider.HealthCheck))
	} else {
		instanceUpdateInput.Properties.HealthCheck = nil
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.HTTPHealthCheck)) {
		instanceUpdateInput.Properties.SetHttpHealthCheck(getHTTPHealthCheck(cr.Spec.ForProvider.HTTPHealthCheck))
	} else {
		instanceUpdateInput.Properties.HttpHealthCheck = nil
	}
	return &instanceUpdateInput, nil
}

// IsTargetGroupUpToDate returns true if the TargetGroup is up-to-date or false if it does not
func IsTargetGroupUpToDate(cr *v1alpha1.TargetGroup, targetGroup sdkgo.TargetGroup) bool { // nolint:gocyclo
	switch {
	case cr == nil && targetGroup.Properties == nil:
		return true
	case cr == nil && targetGroup.Properties != nil:
		return false
	case cr != nil && targetGroup.Properties == nil:
		return false
	case targetGroup.Metadata.State != nil && *targetGroup.Metadata.State == "BUSY" || *targetGroup.Metadata.State == "DEPLOYING":
		return true
	case targetGroup.Properties.Name != nil && *targetGroup.Properties.Name != cr.Spec.ForProvider.Name:
		return false
	case targetGroup.Properties.Protocol != nil && *targetGroup.Properties.Protocol != cr.Spec.ForProvider.Protocol:
		return false
	case targetGroup.Properties.Algorithm != nil && *targetGroup.Properties.Algorithm != cr.Spec.ForProvider.Algorithm:
		return false
	case targetGroup.Properties.Targets != nil && !equalTargetGroupTargets(cr.Spec.ForProvider.Targets, *targetGroup.Properties.Targets):
		return false
	case targetGroup.Properties.HealthCheck != nil && !equalTargetGroupHealthCheck(cr.Spec.ForProvider.HealthCheck, *targetGroup.Properties.HealthCheck):
		return false
	case targetGroup.Properties.HttpHealthCheck != nil && !equalTargetGroupHTTPHealthCheck(cr.Spec.ForProvider.HTTPHealthCheck, *targetGroup.Properties.HttpHealthCheck):
		return false
	default:
		return true
	}
}

func getTargets(targetGroupTargets []v1alpha1.TargetGroupTarget) []sdkgo.TargetGroupTarget {
	if len(targetGroupTargets) == 0 {
		return nil
	}
	targets := make([]sdkgo.TargetGroupTarget, len(targetGroupTargets))
	for i, targetGroupTarget := range targetGroupTargets {
		targets[i] = sdkgo.TargetGroupTarget{
			Ip:                 sdkgo.PtrString(targetGroupTarget.IP),
			Port:               sdkgo.PtrInt32(targetGroupTarget.Port),
			Weight:             sdkgo.PtrInt32(targetGroupTarget.Weight),
			HealthCheckEnabled: sdkgo.PtrBool(targetGroupTarget.HealthCheckEnabled),
			MaintenanceEnabled: sdkgo.PtrBool(targetGroupTarget.MaintenanceEnabled),
		}
	}
	return targets
}

func equalTargetGroupTargets(targetGroupTargets []v1alpha1.TargetGroupTarget, targets []sdkgo.TargetGroupTarget) bool {
	if len(targetGroupTargets) != len(targets) {
		return false
	}
	for _, target := range targetGroupTargets {
		if !equalTargetGroupTarget(target, targets) {
			return false
		}
	}
	return true
}

func equalTargetGroupTarget(target v1alpha1.TargetGroupTarget, targets []sdkgo.TargetGroupTarget) bool { // nolint: gocyclo
	if len(targets) == 0 {
		return false
	}
	for _, t := range targets {
		// All properties are available post creation
		if t.HasIp() && t.HasPort() && t.HasWeight() && t.HasMaintenanceEnabled() && t.HasHealthCheckEnabled() {
			if *t.Ip == target.IP && *t.Port == target.Port && *t.Weight == target.Weight &&
				*t.MaintenanceEnabled == target.MaintenanceEnabled && *t.HealthCheckEnabled == target.HealthCheckEnabled {
				return true
			}
		}
	}
	return false
}

func getHealthCheck(healthCheck v1alpha1.TargetGroupHealthCheck) sdkgo.TargetGroupHealthCheck {
	targetHealthCheck := sdkgo.TargetGroupHealthCheck{}
	if !utils.IsEmptyValue(reflect.ValueOf(healthCheck.CheckInterval)) {
		targetHealthCheck.SetCheckInterval(healthCheck.CheckInterval)
	}
	if !utils.IsEmptyValue(reflect.ValueOf(healthCheck.CheckTimeout)) {
		targetHealthCheck.SetCheckTimeout(healthCheck.CheckTimeout)
	}
	if !utils.IsEmptyValue(reflect.ValueOf(healthCheck.Retries)) {
		targetHealthCheck.SetRetries(healthCheck.Retries)
	}
	return targetHealthCheck
}

func equalTargetGroupHealthCheck(healthCheck v1alpha1.TargetGroupHealthCheck, targetHealthCheck sdkgo.TargetGroupHealthCheck) bool {
	if targetHealthCheck.HasCheckInterval(); healthCheck.CheckInterval != *targetHealthCheck.CheckInterval {
		return false
	}
	if targetHealthCheck.HasCheckTimeout(); healthCheck.CheckTimeout != *targetHealthCheck.CheckTimeout {
		return false
	}
	if targetHealthCheck.HasRetries(); healthCheck.Retries != *targetHealthCheck.Retries {
		return false
	}
	return true
}

func getHTTPHealthCheck(httpHealthCheck v1alpha1.TargetGroupHTTPHealthCheck) sdkgo.TargetGroupHttpHealthCheck {
	targetGroupHTTPHealthCheck := sdkgo.TargetGroupHttpHealthCheck{
		Negate: sdkgo.PtrBool(httpHealthCheck.Negate),
	}
	if !utils.IsEmptyValue(reflect.ValueOf(httpHealthCheck.Path)) {
		targetGroupHTTPHealthCheck.SetPath(httpHealthCheck.Path)
	}
	if !utils.IsEmptyValue(reflect.ValueOf(httpHealthCheck.Method)) {
		targetGroupHTTPHealthCheck.SetMethod(httpHealthCheck.Method)
	}
	if !utils.IsEmptyValue(reflect.ValueOf(httpHealthCheck.MatchType)) {
		targetGroupHTTPHealthCheck.SetMatchType(httpHealthCheck.MatchType)
	}
	if !utils.IsEmptyValue(reflect.ValueOf(httpHealthCheck.Response)) {
		targetGroupHTTPHealthCheck.SetResponse(httpHealthCheck.Response)
	}
	if !utils.IsEmptyValue(reflect.ValueOf(httpHealthCheck.Regex)) {
		targetGroupHTTPHealthCheck.SetRegex(httpHealthCheck.Regex)
	}
	return targetGroupHTTPHealthCheck
}

func equalTargetGroupHTTPHealthCheck(httpHealthCheck v1alpha1.TargetGroupHTTPHealthCheck, targetGroupHTTPHealthCheck sdkgo.TargetGroupHttpHealthCheck) bool {
	if targetGroupHTTPHealthCheck.HasPath(); httpHealthCheck.Path != *targetGroupHTTPHealthCheck.Path {
		return false
	}
	if targetGroupHTTPHealthCheck.HasMethod(); httpHealthCheck.Method != *targetGroupHTTPHealthCheck.Method {
		return false
	}
	if targetGroupHTTPHealthCheck.HasMatchType(); httpHealthCheck.MatchType != *targetGroupHTTPHealthCheck.MatchType {
		return false
	}
	if targetGroupHTTPHealthCheck.HasResponse(); httpHealthCheck.Response != *targetGroupHTTPHealthCheck.Response {
		return false
	}
	if targetGroupHTTPHealthCheck.HasRegex(); httpHealthCheck.Regex != *targetGroupHTTPHealthCheck.Regex {
		return false
	}
	return true
}
