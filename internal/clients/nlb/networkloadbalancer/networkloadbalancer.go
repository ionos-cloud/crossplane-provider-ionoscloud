package networkloadbalancer

import (
	"context"
	"errors"
	"fmt"

	"k8s.io/apimachinery/pkg/util/sets"

	sdkgo "github.com/ionos-cloud/sdk-go/v6"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/nlb/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"
)

const (
	nlbGetByIDErr    = "failed to get nlb by ID: %w"
	nlbListErr       = "failed to get nlb list of datacenter: %w"
	nlbCreateErr     = "failed to create nlb: %w"
	nlbCreateWaitErr = "error while waiting for nlb create request: %w"
	nlbUpdateErr     = "failed to update nlb: %w"
	nlbUpdateWaitErr = "error while waiting for nlb update request: %w"
	nlbDeleteErr     = "failed to delete nlb: %w"
	nlbDeleteWaitErr = "error while waiting for nlb delete request: %w"
)

// ErrNotFound no load balancer has been found
var ErrNotFound = errors.New("network load balancer not found")

// APIClient is a wrapper around IONOS Service
type APIClient struct {
	*clients.IonosServices
}

// Client is a wrapper around IONOS Service Network Load Balancer methods
type Client interface {
	CheckDuplicateNetworkLoadBalancer(ctx context.Context, datacenterID, nlbName string) (string, error)
	GetNetworkLoadBalancerByID(ctx context.Context, datacenterID, NetworkLoadBalancerID string) (sdkgo.NetworkLoadBalancer, *sdkgo.APIResponse, error)
	CreateNetworkLoadBalancer(ctx context.Context, datacenterID string, nlb sdkgo.NetworkLoadBalancer) (sdkgo.NetworkLoadBalancer, *sdkgo.APIResponse, error)
	UpdateNetworkLoadBalancer(ctx context.Context, datacenterID, nlbID string, nlbProperties sdkgo.NetworkLoadBalancerProperties) (sdkgo.NetworkLoadBalancer, *sdkgo.APIResponse, error)
	DeleteNetworkLoadBalancer(ctx context.Context, datacenterID, nlbID string) (*sdkgo.APIResponse, error)
}

// CheckDuplicateNetworkLoadBalancer returns the ID of the duplicate Network Load Balancer if any,
// or an error if multiple Network Load Balancers with the same name are found
func (cp *APIClient) CheckDuplicateNetworkLoadBalancer(ctx context.Context, datacenterID, nlbName string) (string, error) {
	networkLoadBalancers, _, err := cp.ComputeClient.NetworkLoadBalancersApi.DatacentersNetworkloadbalancersGet(ctx, datacenterID).Depth(utils.DepthQueryParam).Execute()
	if err != nil {
		return "", fmt.Errorf(nlbListErr, err)
	}

	matchedItems := make([]sdkgo.NetworkLoadBalancer, 0)

	if networkLoadBalancers.Items != nil {
		for _, item := range *networkLoadBalancers.Items {
			if item.Properties != nil && item.Properties.Name != nil && *item.Properties.Name == nlbName {
				matchedItems = append(matchedItems, item)
			}
		}
	}

	if len(matchedItems) == 0 {
		return "", nil
	}
	if len(matchedItems) > 1 {
		return "", fmt.Errorf("error: found multiple Network Load Balancers with the name %v", nlbName)
	}
	if matchedItems[0].Id == nil {
		return "", fmt.Errorf("error getting ID for Network Load Balancer named: %v", nlbName)
	}
	return *matchedItems[0].Id, nil
}

// GetNetworkLoadBalancerByID based on Datacenter ID and NetworkLoadBalancer ID
func (cp *APIClient) GetNetworkLoadBalancerByID(ctx context.Context, datacenterID, nlbID string) (sdkgo.NetworkLoadBalancer, *sdkgo.APIResponse, error) {
	nlb, apiResponse, err := cp.ComputeClient.NetworkLoadBalancersApi.DatacentersNetworkloadbalancersFindByNetworkLoadBalancerId(ctx, datacenterID, nlbID).Depth(utils.DepthQueryParam).Execute()
	if err != nil {
		err = ErrNotFound
		if !apiResponse.HttpNotFound() {
			err = fmt.Errorf(nlbGetByIDErr, err)
		}
	}
	return nlb, apiResponse, err
}

// CreateNetworkLoadBalancer based on Datacenter ID and NetworkLoadBalancer
func (cp *APIClient) CreateNetworkLoadBalancer(ctx context.Context, datacenterID string, nlb sdkgo.NetworkLoadBalancer) (sdkgo.NetworkLoadBalancer, *sdkgo.APIResponse, error) {
	nlb, apiResponse, err := cp.ComputeClient.NetworkLoadBalancersApi.DatacentersNetworkloadbalancersPost(ctx, datacenterID).NetworkLoadBalancer(nlb).Execute()
	if err != nil {
		return sdkgo.NetworkLoadBalancer{}, apiResponse, fmt.Errorf(nlbCreateErr, err)
	}
	if err = compute.WaitForRequest(ctx, cp.ComputeClient, apiResponse); err != nil {
		return sdkgo.NetworkLoadBalancer{}, apiResponse, fmt.Errorf(nlbCreateWaitErr, err)
	}
	return nlb, apiResponse, nil
}

// UpdateNetworkLoadBalancer based on  Datacenter ID, NetworkLoadBalancer ID and NetworkLoadBalancerProperties
func (cp *APIClient) UpdateNetworkLoadBalancer(ctx context.Context, datacenterID, nlbID string, nlbProperties sdkgo.NetworkLoadBalancerProperties) (sdkgo.NetworkLoadBalancer, *sdkgo.APIResponse, error) {
	nlb, apiResponse, err := cp.ComputeClient.NetworkLoadBalancersApi.DatacentersNetworkloadbalancersPatch(ctx, datacenterID, nlbID).NetworkLoadBalancerProperties(nlbProperties).Execute()
	if err != nil {
		return sdkgo.NetworkLoadBalancer{}, apiResponse, fmt.Errorf(nlbUpdateErr, err)
	}
	if err = compute.WaitForRequest(ctx, cp.ComputeClient, apiResponse); err != nil {
		return sdkgo.NetworkLoadBalancer{}, apiResponse, fmt.Errorf(nlbUpdateWaitErr, err)
	}
	return nlb, apiResponse, nil
}

// DeleteNetworkLoadBalancer based on Datacenter ID and NetworkLoadBalancer ID
func (cp *APIClient) DeleteNetworkLoadBalancer(ctx context.Context, datacenterID, nlbID string) (*sdkgo.APIResponse, error) {
	apiResponse, err := cp.ComputeClient.NetworkLoadBalancersApi.DatacentersNetworkloadbalancersDelete(ctx, datacenterID, nlbID).Execute()
	if err != nil {
		if apiResponse.HttpNotFound() {
			return apiResponse, ErrNotFound
		}
		return apiResponse, fmt.Errorf(nlbDeleteErr, err)
	}
	if err = compute.WaitForRequest(ctx, cp.ComputeClient, apiResponse); err != nil {
		return apiResponse, fmt.Errorf(nlbDeleteWaitErr, err)
	}
	return apiResponse, nil
}

// LateInitializer fills the empty fields in *v1alpha1.NetworkLoadBalancerParameters with
// values that might have been provided by the API in sdkgo.NetworkLoadBalancer
func LateInitializer(in *v1alpha1.NetworkLoadBalancerParameters, nlb sdkgo.NetworkLoadBalancer) bool {
	// Don't initialize fields if the API hasn't set anything or
	// values have already been provided in the NetworkLoadBalancerParameters
	if nlb.Properties == nil || nlb.Properties.LbPrivateIps == nil ||
		len(*nlb.Properties.LbPrivateIps) == 0 || len(in.LbPrivateIps) != 0 {
		return false
	}
	in.LbPrivateIps = *nlb.Properties.LbPrivateIps
	return true
}

// SetStatus sets fields of the NetworkLoadBalancerObservation based on sdkgo.NetworkLoadBalancer
func SetStatus(in *v1alpha1.NetworkLoadBalancerObservation, nlb sdkgo.NetworkLoadBalancer) {
	if nlb.Metadata != nil && nlb.Metadata.State != nil {
		in.State = *nlb.Metadata.State
	}
}

// GenerateCreateInput returns sdkgo.NetworkLoadBalancer for Create requests based on CR spec
func GenerateCreateInput(cr *v1alpha1.NetworkLoadBalancer, listenerLanID, targetLanID int32, publicIPs []string) sdkgo.NetworkLoadBalancer {
	nlbProperties := GenerateUpdateInput(cr, listenerLanID, targetLanID, publicIPs)
	instanceCreateInput := sdkgo.NetworkLoadBalancer{
		Properties: &nlbProperties,
	}
	return instanceCreateInput
}

// GenerateUpdateInput returns sdkgo.NetworkLoadBalancerProperties for Update requests based on CR spec
func GenerateUpdateInput(cr *v1alpha1.NetworkLoadBalancer, listenerLanID, targetLanID int32, publicIPs []string) sdkgo.NetworkLoadBalancerProperties {
	instanceUpdateInput := sdkgo.NetworkLoadBalancerProperties{
		Name:        &cr.Spec.ForProvider.Name,
		ListenerLan: &listenerLanID,
		TargetLan:   &targetLanID,
	}
	if len(publicIPs) != 0 {
		instanceUpdateInput.Ips = &publicIPs
	}
	if len(cr.Spec.ForProvider.LbPrivateIps) != 0 {
		instanceUpdateInput.LbPrivateIps = &cr.Spec.ForProvider.LbPrivateIps
	}
	return instanceUpdateInput
}

// IsUpToDate returns true if the NetworkLoadBalancer is up-to-date or false otherwise
func IsUpToDate(cr *v1alpha1.NetworkLoadBalancer, observed sdkgo.NetworkLoadBalancer, listenerLan, targetLan int32, ips []string) bool { // nolint:gocyclo
	switch {
	case cr == nil && observed.Properties == nil:
		return true
	case cr == nil && observed.Properties != nil:
		return false
	case cr != nil && observed.Properties == nil:
		return false
	case cr.Status.AtProvider.State == compute.BUSY || cr.Status.AtProvider.State == compute.UPDATING:
		return true
	case observed.Properties.Name != nil && *observed.Properties.Name != cr.Spec.ForProvider.Name:
		return false
	case observed.Properties.Name == nil && cr.Spec.ForProvider.Name != "":
		return false
	case observed.Properties.ListenerLan != nil && *observed.Properties.ListenerLan != listenerLan:
		return false
	case observed.Properties.TargetLan != nil && *observed.Properties.TargetLan != targetLan:
		return false
	}

	if observed.Properties.Ips != nil {
		if len(*observed.Properties.Ips) != len(ips) {
			return false
		}
		obsIPs := sets.New[string](*observed.Properties.Ips...)
		cfgIPs := sets.New[string](ips...)
		if !obsIPs.Equal(cfgIPs) {
			return false
		}
	} else if len(ips) != 0 {
		return false
	}

	if observed.Properties.LbPrivateIps != nil {
		if len(*observed.Properties.LbPrivateIps) != len(cr.Spec.ForProvider.LbPrivateIps) {
			return false
		}
		obsIPs := sets.New[string](*observed.Properties.LbPrivateIps...)
		cfgIPs := sets.New[string](cr.Spec.ForProvider.LbPrivateIps...)
		if !obsIPs.Equal(cfgIPs) {
			return false
		}
	} else if len(cr.Spec.ForProvider.LbPrivateIps) != 0 {
		return false
	}

	return true
}
