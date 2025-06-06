package flowlog

import (
	"context"

	sdkgo "github.com/ionos-cloud/sdk-go/v6"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"
)

// NLBFlowLog wrapper for IONOS NetworkLoadBalancer flow log methods
type NLBFlowLog interface {
	CheckDuplicateFlowLog(ctx context.Context, datacenterID, nlbID, flowLogName string) (string, error)
	GetFlowLogByID(ctx context.Context, datacenterID, nlbID, flowLogID string) (sdkgo.FlowLog, error)
	CreateFlowLog(ctx context.Context, datacenterID, nlbID string, flowLog sdkgo.FlowLog) (sdkgo.FlowLog, error)
	UpdateFlowLog(ctx context.Context, datacenterID, nlbID, flowLogID string, flowLog sdkgo.FlowLogProperties) (sdkgo.FlowLog, error)
	DeleteFlowLog(ctx context.Context, datacenterID, nlbID, flowLogID string) error
}

// nlbClient implements NetworkLoadBalancer specific functionality for IONOS flow log
type nlbClient struct {
	flowLogClient
	*clients.IonosServices
}

// CheckDuplicateFlowLog returns the ID of the duplicate Flow Log if any,
// or an error if multiple Flow Logs with the same name are found
func (nc *nlbClient) CheckDuplicateFlowLog(ctx context.Context, datacenterID, nlbID, flowLogName string) (string, error) {
	listFn := nc.IonosServices.ComputeClient.NetworkLoadBalancersApi.DatacentersNetworkloadbalancersFlowlogsGet(ctx, datacenterID, nlbID).Depth(utils.DepthQueryParam).Execute
	return nc.flowLogClient.checkDuplicateFlowLog(flowLogName, listFn)
}

// GetFlowLogByID based on Datacenter ID, NetworkLoadBalancer ID and FlowLog ID
func (nc *nlbClient) GetFlowLogByID(ctx context.Context, datacenterID, nlbID, flowLogID string) (sdkgo.FlowLog, error) {
	byIDFn := nc.IonosServices.ComputeClient.NetworkLoadBalancersApi.DatacentersNetworkloadbalancersFlowlogsFindByFlowLogId(ctx, datacenterID, nlbID, flowLogID).Depth(utils.DepthQueryParam).Execute
	return nc.flowLogClient.getFlowLogByID(byIDFn)
}

// CreateFlowLog based on Datacenter ID, NetworkLoadBalancer ID and FlowLog
func (nc *nlbClient) CreateFlowLog(ctx context.Context, datacenterID, nlbID string, flowLog sdkgo.FlowLog) (sdkgo.FlowLog, error) {
	createFn := nc.IonosServices.ComputeClient.NetworkLoadBalancersApi.DatacentersNetworkloadbalancersFlowlogsPost(ctx, datacenterID, nlbID).NetworkLoadBalancerFlowLog(flowLog).Execute
	return nc.flowLogClient.createFlowLog(ctx, createFn)
}

// UpdateFlowLog based on Datacenter ID, NetworkLoadBalancer ID, FlowLog ID, and FlowLog
func (nc *nlbClient) UpdateFlowLog(ctx context.Context, datacenterID, nlbID, flowLogID string, flowLog sdkgo.FlowLogProperties) (sdkgo.FlowLog, error) {
	updateFn := nc.IonosServices.ComputeClient.NetworkLoadBalancersApi.DatacentersNetworkloadbalancersFlowlogsPatch(ctx, datacenterID, nlbID, flowLogID).NetworkLoadBalancerFlowLogProperties(flowLog).Execute
	return nc.flowLogClient.updateFlowLog(ctx, updateFn)
}

// DeleteFlowLog based on Datacenter ID, NetworkLoadBalancer ID and FlowLog ID
func (nc *nlbClient) DeleteFlowLog(ctx context.Context, datacenterID, nlbID, flowLogID string) error {
	deleteFn := nc.IonosServices.ComputeClient.NetworkLoadBalancersApi.DatacentersNetworkloadbalancersFlowlogsDelete(ctx, datacenterID, nlbID, flowLogID).Execute
	return nc.flowLogClient.deleteFlowLog(ctx, deleteFn)
}

// NLBClient returns a new NetworkLoadBalancer flow log client
func NLBClient(svc *clients.IonosServices) NLBFlowLog {
	return &nlbClient{
		IonosServices: svc,
		flowLogClient: &client{
			parent:        "NetworkLoadBalancer",
			IonosServices: svc,
		},
	}
}
