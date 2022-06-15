package applicationloadbalancer

import (
	"context"
	"reflect"

	sdkgo "github.com/ionos-cloud/sdk-go/v6"
	"github.com/rung/go-safecast"

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
	GetApplicationLoadBalancer(ctx context.Context, datacenterID, applicationloadbalancerID string) (sdkgo.ApplicationLoadBalancer, *sdkgo.APIResponse, error)
	CreateApplicationLoadBalancer(ctx context.Context, datacenterID string, applicationloadbalancer sdkgo.ApplicationLoadBalancer) (sdkgo.ApplicationLoadBalancer, *sdkgo.APIResponse, error)
	UpdateApplicationLoadBalancer(ctx context.Context, datacenterID, applicationloadbalancerID string, applicationloadbalancer sdkgo.ApplicationLoadBalancerProperties) (sdkgo.ApplicationLoadBalancer, *sdkgo.APIResponse, error)
	DeleteApplicationLoadBalancer(ctx context.Context, datacenterID, applicationloadbalancerID string) (*sdkgo.APIResponse, error)
	GetAPIClient() *sdkgo.APIClient
}

// GetApplicationLoadBalancer based on applicationloadbalancerID
func (cp *APIClient) GetApplicationLoadBalancer(ctx context.Context, datacenterID, applicationloadbalancerID string) (sdkgo.ApplicationLoadBalancer, *sdkgo.APIResponse, error) {
	return cp.ComputeClient.ApplicationLoadBalancersApi.DatacentersApplicationloadbalancersFindByApplicationLoadBalancerId(ctx, datacenterID, applicationloadbalancerID).Depth(utils.DepthQueryParam).Execute()
}

// CreateApplicationLoadBalancer based on ApplicationLoadBalancer
func (cp *APIClient) CreateApplicationLoadBalancer(ctx context.Context, datacenterID string, applicationloadbalancer sdkgo.ApplicationLoadBalancer) (sdkgo.ApplicationLoadBalancer, *sdkgo.APIResponse, error) {
	return cp.ComputeClient.ApplicationLoadBalancersApi.DatacentersApplicationloadbalancersPost(ctx, datacenterID).ApplicationLoadBalancer(applicationloadbalancer).Execute()
}

// UpdateApplicationLoadBalancer based on applicationloadbalancerID and ApplicationLoadBalancerProperties
func (cp *APIClient) UpdateApplicationLoadBalancer(ctx context.Context, datacenterID, applicationloadbalancerID string, applicationloadbalancer sdkgo.ApplicationLoadBalancerProperties) (sdkgo.ApplicationLoadBalancer, *sdkgo.APIResponse, error) {
	return cp.ComputeClient.ApplicationLoadBalancersApi.DatacentersApplicationloadbalancersPatch(ctx, datacenterID, applicationloadbalancerID).ApplicationLoadBalancerProperties(applicationloadbalancer).Execute()
}

// DeleteApplicationLoadBalancer based on applicationloadbalancerID
func (cp *APIClient) DeleteApplicationLoadBalancer(ctx context.Context, datacenterID, applicationloadbalancerID string) (*sdkgo.APIResponse, error) {
	resp, err := cp.ComputeClient.ApplicationLoadBalancersApi.DatacentersApplicationloadbalancersDelete(ctx, datacenterID, applicationloadbalancerID).Execute()
	return resp, err
}

// GetAPIClient gets the APIClient
func (cp *APIClient) GetAPIClient() *sdkgo.APIClient {
	return cp.ComputeClient
}

// GenerateCreateApplicationLoadBalancerInput returns sdkgo.ApplicationLoadBalancer based on the CR spec
func GenerateCreateApplicationLoadBalancerInput(cr *v1alpha1.ApplicationLoadBalancer, ips []string) (*sdkgo.ApplicationLoadBalancer, error) {
	listenerLanID, err := safecast.Atoi32(cr.Spec.ForProvider.ListenerLanCfg.LanID)
	if err != nil {
		return nil, err
	}
	targetLanID, err := safecast.Atoi32(cr.Spec.ForProvider.TargetLanCfg.LanID)
	if err != nil {
		return nil, err
	}
	instanceCreateInput := sdkgo.ApplicationLoadBalancer{
		Properties: &sdkgo.ApplicationLoadBalancerProperties{
			Name:        &cr.Spec.ForProvider.Name,
			ListenerLan: &listenerLanID,
			TargetLan:   &targetLanID,
		},
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.LbPrivateIps)) {
		instanceCreateInput.Properties.SetLbPrivateIps(cr.Spec.ForProvider.LbPrivateIps)
	}
	if !utils.IsEmptyValue(reflect.ValueOf(ips)) {
		instanceCreateInput.Properties.SetIps(ips)
	}
	return &instanceCreateInput, nil
}

// GenerateUpdateApplicationLoadBalancerInput returns sdkgo.ApplicationLoadBalancerProperties based on the CR spec modifications
func GenerateUpdateApplicationLoadBalancerInput(cr *v1alpha1.ApplicationLoadBalancer, ips []string) (*sdkgo.ApplicationLoadBalancerProperties, error) {
	listenerLanID, err := safecast.Atoi32(cr.Spec.ForProvider.ListenerLanCfg.LanID)
	if err != nil {
		return nil, err
	}
	targetLanID, err := safecast.Atoi32(cr.Spec.ForProvider.TargetLanCfg.LanID)
	if err != nil {
		return nil, err
	}
	instanceUpdateInput := sdkgo.ApplicationLoadBalancerProperties{
		Name:        &cr.Spec.ForProvider.Name,
		ListenerLan: &listenerLanID,
		TargetLan:   &targetLanID,
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.LbPrivateIps)) {
		instanceUpdateInput.SetLbPrivateIps(cr.Spec.ForProvider.LbPrivateIps)
	}
	if !utils.IsEmptyValue(reflect.ValueOf(ips)) {
		instanceUpdateInput.SetIps(ips)
	}
	return &instanceUpdateInput, nil
}

// LateInitializer fills the empty fields in *v1alpha1.ApplicationLoadBalancerParameters with
// the values seen in sdkgo.ApplicationLoadBalancer.
func LateInitializer(in *v1alpha1.ApplicationLoadBalancerParameters, alb *sdkgo.ApplicationLoadBalancer) { // nolint:gocyclo
	if alb == nil {
		return
	}
	// Add Properties to the Spec, if they were set by the API
	if propertiesOk, ok := alb.GetPropertiesOk(); ok && propertiesOk != nil {
		if lbPrivateIpsOk, ok := propertiesOk.GetLbPrivateIpsOk(); ok && lbPrivateIpsOk != nil {
			if utils.IsEmptyValue(reflect.ValueOf(in.LbPrivateIps)) {
				in.LbPrivateIps = *lbPrivateIpsOk
			}
		}
	}
}

// IsApplicationLoadBalancerUpToDate returns true if the ApplicationLoadBalancer is up-to-date or false if it does not
func IsApplicationLoadBalancerUpToDate(cr *v1alpha1.ApplicationLoadBalancer, applicationloadbalancer sdkgo.ApplicationLoadBalancer, listenerLan, targetLan int32, ips []string) bool { // nolint:gocyclo
	switch {
	case cr == nil && applicationloadbalancer.Properties == nil:
		return true
	case cr == nil && applicationloadbalancer.Properties != nil:
		return false
	case cr != nil && applicationloadbalancer.Properties == nil:
		return false
	case applicationloadbalancer.Metadata.State != nil && *applicationloadbalancer.Metadata.State == "BUSY" || *applicationloadbalancer.Metadata.State == "DEPLOYING":
		return true
	case applicationloadbalancer.Properties.Name != nil && *applicationloadbalancer.Properties.Name != cr.Spec.ForProvider.Name:
		return false
	case applicationloadbalancer.Properties.ListenerLan != nil && *applicationloadbalancer.Properties.ListenerLan != listenerLan:
		return false
	case applicationloadbalancer.Properties.TargetLan != nil && *applicationloadbalancer.Properties.TargetLan != targetLan:
		return false
	case applicationloadbalancer.Properties.Ips != nil && !utils.ContainsStringSlices(*applicationloadbalancer.Properties.Ips, cr.Status.AtProvider.PublicIPs):
		return false
	case !utils.ContainsStringSlices(ips, cr.Status.AtProvider.PublicIPs):
		return false
	case applicationloadbalancer.Properties.LbPrivateIps != nil && !utils.ContainsStringSlices(*applicationloadbalancer.Properties.LbPrivateIps, cr.Spec.ForProvider.LbPrivateIps):
		return false
	default:
		return true
	}
}
