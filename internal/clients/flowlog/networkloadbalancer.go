package flowlog

import (
	"context"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"
	sdkgo "github.com/ionos-cloud/sdk-go/v6"
)

// NLBFlowLog wrapper flowLogClientfor NetworkLoadBalancer flow log methods
type NLBFlowLog interface {
	CheckDuplicateFlowLog(ctx context.Context, datacenterID, nlbID, flowLogName string) (string, error)
	GetFlowLogByID(ctx context.Context, datacenterID, nlbID, flowLogID string) (sdkgo.FlowLog, *sdkgo.APIResponse, error)
	CreateFlowLog(ctx context.Context, datacenterID, nlbID string, flowLog sdkgo.FlowLog) (sdkgo.FlowLog, *sdkgo.APIResponse, error)
	UpdateFlowLog(ctx context.Context, datacenterID, nlbID, flowLogID string, flowLog sdkgo.FlowLogProperties) (sdkgo.FlowLog, *sdkgo.APIResponse, error)
	DeleteFlowLog(ctx context.Context, datacenterID, nlbID, flowLogID string) (*sdkgo.APIResponse, error)
}

// CheckDuplicateFlowLog returns the ID of the duplicate Flow Log if any,
// or an error if multiple Flow Logs with the same name are found
func (c *client) CheckDuplicateFlowLog(ctx context.Context, datacenterID, nlbID, flowLogName string) (string, error) {
	listFn := c.ComputeClient.NetworkLoadBalancersApi.DatacentersNetworkloadbalancersFlowlogsGet(ctx, datacenterID, nlbID).Depth(utils.DepthQueryParam).Execute
	return c.flowLogClient.checkDuplicateFlowLog(flowLogName, listFn)
}

// GetFlowLogByID based on Datacenter ID, NetworkLoadBalancer ID and FlowLog ID
func (c *client) GetFlowLogByID(ctx context.Context, datacenterID, nlbID, flowLogID string) (sdkgo.FlowLog, *sdkgo.APIResponse, error) {
	byIdFn := c.ComputeClient.NetworkLoadBalancersApi.DatacentersNetworkloadbalancersFlowlogsFindByFlowLogId(ctx, datacenterID, nlbID, flowLogID).Depth(utils.DepthQueryParam).Execute
	return c.flowLogClient.getFlowLogByID(byIdFn)
}

// CreateFlowLog based on Datacenter ID, NetworkLoadBalancer ID and FlowLog
func (c *client) CreateFlowLog(ctx context.Context, datacenterID, nlbID string, flowLog sdkgo.FlowLog) (sdkgo.FlowLog, *sdkgo.APIResponse, error) {
	createFn := c.ComputeClient.NetworkLoadBalancersApi.DatacentersNetworkloadbalancersFlowlogsPost(ctx, datacenterID, nlbID).NetworkLoadBalancerFlowLog(flowLog).Execute
	return c.flowLogClient.createFlowLog(ctx, createFn)
}

// UpdateFlowLog based on Datacenter ID, NetworkLoadBalancer ID, FlowLog ID, and FlowLog
func (c *client) UpdateFlowLog(ctx context.Context, datacenterID, nlbID, flowLogID string, flowLog sdkgo.FlowLogProperties) (sdkgo.FlowLog, *sdkgo.APIResponse, error) {
	updateFn := c.ComputeClient.NetworkLoadBalancersApi.DatacentersNetworkloadbalancersFlowlogsPatch(ctx, datacenterID, nlbID, flowLogID).NetworkLoadBalancerFlowLogProperties(flowLog).Execute
	return c.flowLogClient.updateFlowLog(ctx, updateFn)
}

// DeleteFlowLog based on Datacenter ID, NetworkLoadBalancer ID and FlowLog ID
func (c *client) DeleteFlowLog(ctx context.Context, datacenterID, nlbID, flowLogID string) (*sdkgo.APIResponse, error) {
	deleteFn := c.ComputeClient.NetworkLoadBalancersApi.DatacentersNetworkloadbalancersFlowlogsDelete(ctx, datacenterID, nlbID, flowLogID).Execute
	return c.flowLogClient.deleteFlowLog(ctx, deleteFn)
}

// NLBClient returns a new NetworkLoadBalancer flow log client
func NLBClient(svc *clients.IonosServices) NLBFlowLog {
	return &client{
		IonosServices: svc,
		parent:        "NetworkLoadBalancer",
	}
}
