package flowlog

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
	CheckDuplicateFlowLog(ctx context.Context, datacenterID, nlbID, flowLogName string) (string, error)
	GetFlowLogByID(ctx context.Context, datacenterID, nlbID, flowLogID string) (sdkgo.FlowLog, *sdkgo.APIResponse, error)
	CreateFlowLog(ctx context.Context, datacenterID, nlbID string, flowLog sdkgo.FlowLog) (sdkgo.FlowLog, *sdkgo.APIResponse, error)
	UpdateFlowLog(ctx context.Context, datacenterID, nlbID, flowLogID string, flowLog sdkgo.FlowLogProperties) (sdkgo.FlowLog, *sdkgo.APIResponse, error)
	DeleteFlowLog(ctx context.Context, datacenterID, nlbID, flowLogID string) (*sdkgo.APIResponse, error)
}

// CheckDuplicateFlowLog returns the ID of the duplicate Flow Log if any,
// or an error if multiple Flow Logs with the same name are found
func (cp *APIClient) CheckDuplicateFlowLog(ctx context.Context, datacenterID, nlbID, flowLogName string) (string, error) {
	FlowLogs, _, err := cp.ComputeClient.NetworkLoadBalancersApi.DatacentersNetworkloadbalancersFlowlogsGet(ctx, datacenterID, nlbID).Depth(utils.DepthQueryParam).Execute()
	if err != nil {
		return "", err
	}

	matchedItems := make([]sdkgo.FlowLog, 0)

	if FlowLogs.Items != nil {
		for _, item := range *FlowLogs.Items {
			if item.Properties != nil && item.Properties.Name != nil && *item.Properties.Name == flowLogName {
				matchedItems = append(matchedItems, item)
			}
		}
	}

	if len(matchedItems) == 0 {
		return "", nil
	}
	if len(matchedItems) > 1 {
		return "", fmt.Errorf("error: found multiple Flow Logs with the name %v", flowLogName)
	}
	if matchedItems[0].Id == nil {
		return "", fmt.Errorf("error getting ID for Flow Log named: %v", flowLogName)
	}
	return *matchedItems[0].Id, nil
}

// GetFlowLogByID based on Datacenter ID, NetworkLoadBalancer ID and FlowLog ID
func (cp *APIClient) GetFlowLogByID(ctx context.Context, datacenterID, nlbID, flowLogID string) (sdkgo.FlowLog, *sdkgo.APIResponse, error) {
	return cp.ComputeClient.NetworkLoadBalancersApi.DatacentersNetworkloadbalancersFlowlogsFindByFlowLogId(ctx, datacenterID, nlbID, flowLogID).Depth(utils.DepthQueryParam).Execute()
}

// CreateFlowLog based on Datacenter ID, NetworkLoadBalancer ID and FlowLog
func (cp *APIClient) CreateFlowLog(ctx context.Context, datacenterID, nlbID string, flowLog sdkgo.FlowLog) (sdkgo.FlowLog, *sdkgo.APIResponse, error) {
	return cp.ComputeClient.NetworkLoadBalancersApi.DatacentersNetworkloadbalancersFlowlogsPost(ctx, datacenterID, nlbID).NetworkLoadBalancerFlowLog(flowLog).Execute()
}

// UpdateFlowLog based on Datacenter ID, NetworkLoadBalancer ID, FlowLog ID, and FlowLog
func (cp *APIClient) UpdateFlowLog(ctx context.Context, datacenterID, nlbID, flowLogID string, flowLog sdkgo.FlowLogProperties) (sdkgo.FlowLog, *sdkgo.APIResponse, error) {
	return cp.ComputeClient.NetworkLoadBalancersApi.DatacentersNetworkloadbalancersFlowlogsPatch(ctx, datacenterID, nlbID, flowLogID).NetworkLoadBalancerFlowLogProperties(flowLog).Execute()
}

// DeleteFlowLog based on Datacenter ID, NetworkLoadBalancer ID and FlowLog ID
func (cp *APIClient) DeleteFlowLog(ctx context.Context, datacenterID, nlbID, flowLogID string) (*sdkgo.APIResponse, error) {
	return cp.ComputeClient.NetworkLoadBalancersApi.DatacentersNetworkloadbalancersFlowlogsDelete(ctx, datacenterID, nlbID, flowLogID).Execute()
}

func IsUpToDate(cr *v1alpha1.FlowLog, observed sdkgo.FlowLog) bool { // nolint:gocyclo
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
