package flowlog

import (
	"context"
	"fmt"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/flowlog"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/compare"
	sdkgo "github.com/ionos-cloud/sdk-go/v6"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/nlb/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"
)

var ErrNotFound = fmt.Errorf("network load balancer: %w", flowlog.ErrNotFound)

type apiClient struct {
	*clients.IonosServices
	fc flowlog.Client
}

// Client wrapper
type Client interface {
	CheckDuplicateFlowLog(ctx context.Context, datacenterID, nlbID, flowLogName string) (string, error)
	GetFlowLogByID(ctx context.Context, datacenterID, nlbID, flowLogID string) (sdkgo.FlowLog, *sdkgo.APIResponse, error)
	CreateFlowLog(ctx context.Context, datacenterID, nlbID string, flowLog sdkgo.FlowLog) (sdkgo.FlowLog, *sdkgo.APIResponse, error)
	UpdateFlowLog(ctx context.Context, datacenterID, nlbID, flowLogID string, flowLog sdkgo.FlowLogProperties) (sdkgo.FlowLog, *sdkgo.APIResponse, error)
	DeleteFlowLog(ctx context.Context, datacenterID, nlbID, flowLogID string) (*sdkgo.APIResponse, error)
}

// CheckDuplicateFlowLog returns the ID of the duplicate Flow Log if any,
// or an error if multiple Flow Logs with the same name are found
func (cp *apiClient) CheckDuplicateFlowLog(ctx context.Context, datacenterID, nlbID, flowLogName string) (string, error) {
	listFn := cp.ComputeClient.NetworkLoadBalancersApi.DatacentersNetworkloadbalancersFlowlogsGet(ctx, datacenterID, nlbID).Depth(utils.DepthQueryParam).Execute
	return cp.fc.CheckDuplicateFlowLog(flowLogName, listFn)
}

// GetFlowLogByID based on Datacenter ID, NetworkLoadBalancer ID and FlowLog ID
func (cp *apiClient) GetFlowLogByID(ctx context.Context, datacenterID, nlbID, flowLogID string) (sdkgo.FlowLog, *sdkgo.APIResponse, error) {
	byIdFn := cp.ComputeClient.NetworkLoadBalancersApi.DatacentersNetworkloadbalancersFlowlogsFindByFlowLogId(ctx, datacenterID, nlbID, flowLogID).Depth(utils.DepthQueryParam).Execute
	return cp.fc.GetFlowLogByID(byIdFn)
}

// CreateFlowLog based on Datacenter ID, NetworkLoadBalancer ID and FlowLog
func (cp *apiClient) CreateFlowLog(ctx context.Context, datacenterID, nlbID string, flowLog sdkgo.FlowLog) (sdkgo.FlowLog, *sdkgo.APIResponse, error) {
	createFn := cp.ComputeClient.NetworkLoadBalancersApi.DatacentersNetworkloadbalancersFlowlogsPost(ctx, datacenterID, nlbID).NetworkLoadBalancerFlowLog(flowLog).Execute
	return cp.fc.CreateFlowLog(ctx, createFn)
}

// UpdateFlowLog based on Datacenter ID, NetworkLoadBalancer ID, FlowLog ID, and FlowLog
func (cp *apiClient) UpdateFlowLog(ctx context.Context, datacenterID, nlbID, flowLogID string, flowLog sdkgo.FlowLogProperties) (sdkgo.FlowLog, *sdkgo.APIResponse, error) {
	updateFn := cp.ComputeClient.NetworkLoadBalancersApi.DatacentersNetworkloadbalancersFlowlogsPatch(ctx, datacenterID, nlbID, flowLogID).NetworkLoadBalancerFlowLogProperties(flowLog).Execute
	return cp.fc.UpdateFlowLog(ctx, updateFn)
}

// DeleteFlowLog based on Datacenter ID, NetworkLoadBalancer ID and FlowLog ID
func (cp *apiClient) DeleteFlowLog(ctx context.Context, datacenterID, nlbID, flowLogID string) (*sdkgo.APIResponse, error) {
	deleteFn := cp.ComputeClient.NetworkLoadBalancersApi.DatacentersNetworkloadbalancersFlowlogsDelete(ctx, datacenterID, nlbID, flowLogID).Execute
	return cp.fc.DeleteFlowLog(ctx, deleteFn)
}

// SetStatus sets fields of the FlowLogObservation based on sdkgo.FlowLog
func SetStatus(in *v1alpha1.FlowLogObservation, flowLog sdkgo.FlowLog) {
	if flowLog.Metadata != nil && flowLog.Metadata.State != nil {
		in.State = *flowLog.Metadata.State
	}
}

// GenerateCreateInput returns sdkgo.FlowLog for Create requests based on CR spec
func GenerateCreateInput(cr *v1alpha1.FlowLog) sdkgo.FlowLog {
	flowLogProperties := GenerateUpdateInput(cr)
	return sdkgo.FlowLog{Properties: &flowLogProperties}
}

// GenerateUpdateInput returns sdkgo.FlowLogProperties for Update requests based on CR spec
func GenerateUpdateInput(cr *v1alpha1.FlowLog) sdkgo.FlowLogProperties {
	return sdkgo.FlowLogProperties{
		Name:      &cr.Spec.ForProvider.Name,
		Action:    &cr.Spec.ForProvider.Action,
		Direction: &cr.Spec.ForProvider.Direction,
		Bucket:    &cr.Spec.ForProvider.Bucket,
	}
}

// IsUpToDate returns true if the FlowLog is up-to-date or false otherwise
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
	case !compare.EqualFlowLogProperties(cr, observed.Properties):
		return false
	}
	return true
}

// NewClient returns a new NetworkLoadBalancer flow log client
func NewClient(svc *clients.IonosServices) Client {
	return &apiClient{
		IonosServices: svc,
		fc:            flowlog.NewClient("NetworkLoadBalancer", svc),
	}
}
