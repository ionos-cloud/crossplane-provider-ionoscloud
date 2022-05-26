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
	if utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.HealthCheck)) {
		instanceCreateInput.Properties.HealthCheck = nil
	}
	if utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.HTTPHealthCheck)) {
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
	if utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.HealthCheck)) {
		instanceUpdateInput.Properties.HealthCheck = nil
	}
	if utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.HTTPHealthCheck)) {
		instanceUpdateInput.Properties.HttpHealthCheck = nil
	}
	return &instanceUpdateInput, nil
}

// // LateInitializer fills the empty fields in *v1alpha1.TargetGroupParameters with
// // the values seen in sdkgo.TargetGroup.
// func LateInitializer(in *v1alpha1.TargetGroupParameters, alb *sdkgo.TargetGroup) { // nolint:gocyclo
//	if alb == nil {
//		return
//	}
//	// Add Properties to the Spec, if they were set by the API
//	if propertiesOk, ok := alb.GetPropertiesOk(); ok && propertiesOk != nil {
//		if lbPrivateIpsOk, ok := propertiesOk.GetLbPrivateIpsOk(); ok && lbPrivateIpsOk != nil {
//			if utils.IsEmptyValue(reflect.ValueOf(in.LbPrivateIps)) {
//				in.LbPrivateIps = *lbPrivateIpsOk
//			}
//		}
//	}
// }

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
	default:
		return true
	}
}

func getTargets(targetGroupTargets []v1alpha1.TargetGroupTarget) []sdkgo.TargetGroupTarget {
	if len(targetGroupTargets) == 0 {
		return nil
	}
	targets := make([]sdkgo.TargetGroupTarget, 0)
	for _, targetGroupTarget := range targetGroupTargets {
		httpRule := sdkgo.TargetGroupTarget{
			Ip:                 &targetGroupTarget.IP,
			Port:               &targetGroupTarget.Port,
			Weight:             &targetGroupTarget.Weight,
			HealthCheckEnabled: &targetGroupTarget.HealthCheckEnabled,
			MaintenanceEnabled: &targetGroupTarget.MaintenanceEnabled,
		}
		targets = append(targets, httpRule)
	}
	return targets
}
