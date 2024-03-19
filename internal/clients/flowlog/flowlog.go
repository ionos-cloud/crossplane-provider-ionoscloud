package flowlog

import (
	"context"
	"errors"
	"fmt"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute"
	sdkgo "github.com/ionos-cloud/sdk-go/v6"
)

const (
	logGetByIDErr    = "failed to get %s flow log by ID: %w"
	logListErr       = "failed to get %s flow logs list: %w"
	logCreateErr     = "failed to create %s flow log: %w"
	logCreateWaitErr = "error while waiting for %s flow log create request: %w"
	logUpdateErr     = "failed to update %s flow log: %w"
	logUpdateWaitErr = "error while waiting for %s flow log update request: %w"
	logDeleteErr     = "failed to delete %s flow log: %w"
	logDeleteWaitErr = "error while waiting for %s flow log delete request: %w"
)

var ErrNotFound = errors.New("flow log not found")

type listFn = func() (sdkgo.FlowLogs, *sdkgo.APIResponse, error)
type byIdFn = func() (sdkgo.FlowLog, *sdkgo.APIResponse, error)
type createOrPatchFn = func() (sdkgo.FlowLog, *sdkgo.APIResponse, error)
type deleteFn = func() (*sdkgo.APIResponse, error)

// Client defines CRUD operations which are applicable to all custom resource flow logs
type Client interface {
	CheckDuplicateFlowLog(flowLogName string, listFn listFn) (string, error)
	GetFlowLogByID(byIdFn byIdFn) (sdkgo.FlowLog, *sdkgo.APIResponse, error)
	CreateFlowLog(ctx context.Context, createFn createOrPatchFn) (sdkgo.FlowLog, *sdkgo.APIResponse, error)
	UpdateFlowLog(ctx context.Context, updateFn createOrPatchFn) (sdkgo.FlowLog, *sdkgo.APIResponse, error)
	DeleteFlowLog(ctx context.Context, deleteFn deleteFn) (*sdkgo.APIResponse, error)
}

type apiClient struct {
	*clients.IonosServices
	parent string
}

// CheckDuplicateFlowLog searches all flow logs with flowLogName using the listing function listFn
func (cp *apiClient) CheckDuplicateFlowLog(flowLogName string, listFn listFn) (string, error) {
	flowLogs, _, err := listFn()
	if err != nil {
		return "", fmt.Errorf(logListErr, cp.parent, err)
	}

	matchedItems := make([]sdkgo.FlowLog, 0)

	if flowLogs.Items != nil {
		for _, item := range *flowLogs.Items {
			if item.Properties != nil && item.Properties.Name != nil && *item.Properties.Name == flowLogName {
				matchedItems = append(matchedItems, item)
			}
		}
	}

	if len(matchedItems) == 0 {
		return "", nil
	}
	if len(matchedItems) > 1 {
		return "", fmt.Errorf("error: found multiple %s Flow Logs with the name %v", flowLogName, cp.parent)
	}
	if matchedItems[0].Id == nil {
		return "", fmt.Errorf("error getting ID for %s Flow Log named: %v", flowLogName, cp.parent)
	}
	return *matchedItems[0].Id, nil
}

// GetFlowLogByID retrieves a flow log using byIdFn
func (cp *apiClient) GetFlowLogByID(byIdFn byIdFn) (sdkgo.FlowLog, *sdkgo.APIResponse, error) {
	flowLog, apiResponse, err := byIdFn()
	if err != nil {
		err = ErrNotFound
		if !apiResponse.HttpNotFound() {
			err = fmt.Errorf(logGetByIDErr, cp.parent, err)
		}
	}
	return flowLog, apiResponse, err
}

// CreateFlowLog creates a flow log for a resource using createFn
func (cp *apiClient) CreateFlowLog(ctx context.Context, createFn createOrPatchFn) (sdkgo.FlowLog, *sdkgo.APIResponse, error) {
	flowLog, apiResponse, err := createFn()
	if err != nil {
		return sdkgo.FlowLog{}, apiResponse, fmt.Errorf(logCreateErr, cp.parent, err)
	}
	if err = compute.WaitForRequest(ctx, cp.ComputeClient, apiResponse); err != nil {
		return sdkgo.FlowLog{}, apiResponse, fmt.Errorf(logCreateWaitErr, cp.parent, err)
	}
	return flowLog, apiResponse, err
}

// UpdateFlowLog updates the flow log of a resource using updateFn
func (cp *apiClient) UpdateFlowLog(ctx context.Context, updateFn createOrPatchFn) (sdkgo.FlowLog, *sdkgo.APIResponse, error) {
	flowLog, apiResponse, err := updateFn()
	if err != nil {
		return sdkgo.FlowLog{}, apiResponse, fmt.Errorf(logUpdateErr, cp.parent, err)
	}
	if err = compute.WaitForRequest(ctx, cp.ComputeClient, apiResponse); err != nil {
		return sdkgo.FlowLog{}, apiResponse, fmt.Errorf(logUpdateWaitErr, cp.parent, err)
	}
	return flowLog, apiResponse, err
}

// DeleteFlowLog deletes the flow log of a resource using deleteFn
func (cp *apiClient) DeleteFlowLog(ctx context.Context, deleteFn deleteFn) (*sdkgo.APIResponse, error) {
	apiResponse, err := deleteFn()
	if err != nil {
		if apiResponse.HttpNotFound() {
			return apiResponse, ErrNotFound
		}
		return apiResponse, fmt.Errorf(logDeleteErr, cp.parent, err)
	}
	if err = compute.WaitForRequest(ctx, cp.ComputeClient, apiResponse); err != nil {
		return apiResponse, fmt.Errorf(logDeleteWaitErr, cp.parent, err)
	}
	return apiResponse, nil
}

// NewClient returns a new flow log Client
// parent is used to specify the resource type to which this flow log will belong
func NewClient(parent string, svc *clients.IonosServices) Client {
	return &apiClient{parent: parent, IonosServices: svc}
}
