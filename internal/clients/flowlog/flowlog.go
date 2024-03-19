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

// ErrNotFound flow log was not found
var ErrNotFound = errors.New("flow log not found")

type listFunc = func() (sdkgo.FlowLogs, *sdkgo.APIResponse, error)
type byIdFunc = func() (sdkgo.FlowLog, *sdkgo.APIResponse, error)
type createOrPatchFunc = func() (sdkgo.FlowLog, *sdkgo.APIResponse, error)
type deleteFunc = func() (*sdkgo.APIResponse, error)

// flowLogClient common CRUD interface for flow log
type flowLogClient interface {
	checkDuplicateFlowLog(flowLogName string, listFn listFunc) (string, error)
	getFlowLogByID(byIdFn byIdFunc) (sdkgo.FlowLog, *sdkgo.APIResponse, error)
	createFlowLog(ctx context.Context, createFn createOrPatchFunc) (sdkgo.FlowLog, *sdkgo.APIResponse, error)
	updateFlowLog(ctx context.Context, updateFn createOrPatchFunc) (sdkgo.FlowLog, *sdkgo.APIResponse, error)
	deleteFlowLog(ctx context.Context, deleteFn deleteFunc) (*sdkgo.APIResponse, error)
}

// client implements common functionality for IONOS flow log
type client struct {
	*clients.IonosServices
	parent string
}

func (c *client) checkDuplicateFlowLog(flowLogName string, listFn listFunc) (string, error) {
	flowLogs, _, err := listFn()
	if err != nil {
		return "", fmt.Errorf(logListErr, c.parent, err)
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
		return "", fmt.Errorf("error: found multiple %s Flow Logs with the name %v", flowLogName, c.parent)
	}
	if matchedItems[0].Id == nil {
		return "", fmt.Errorf("error getting ID for %s Flow Log named: %v", flowLogName, c.parent)
	}
	return *matchedItems[0].Id, nil
}

func (c *client) getFlowLogByID(byIdFn byIdFunc) (sdkgo.FlowLog, *sdkgo.APIResponse, error) {
	flowLog, apiResponse, err := byIdFn()
	if err != nil {
		err = ErrNotFound
		if !apiResponse.HttpNotFound() {
			err = fmt.Errorf(logGetByIDErr, c.parent, err)
		}
	}
	return flowLog, apiResponse, err
}

func (c *client) createFlowLog(ctx context.Context, createFn createOrPatchFunc) (sdkgo.FlowLog, *sdkgo.APIResponse, error) {
	flowLog, apiResponse, err := createFn()
	if err != nil {
		return sdkgo.FlowLog{}, apiResponse, fmt.Errorf(logCreateErr, c.parent, err)
	}
	if err = compute.WaitForRequest(ctx, c.ComputeClient, apiResponse); err != nil {
		return sdkgo.FlowLog{}, apiResponse, fmt.Errorf(logCreateWaitErr, c.parent, err)
	}
	return flowLog, apiResponse, err
}

func (c *client) updateFlowLog(ctx context.Context, updateFn createOrPatchFunc) (sdkgo.FlowLog, *sdkgo.APIResponse, error) {
	flowLog, apiResponse, err := updateFn()
	if err != nil {
		return sdkgo.FlowLog{}, apiResponse, fmt.Errorf(logUpdateErr, c.parent, err)
	}
	if err = compute.WaitForRequest(ctx, c.ComputeClient, apiResponse); err != nil {
		return sdkgo.FlowLog{}, apiResponse, fmt.Errorf(logUpdateWaitErr, c.parent, err)
	}
	return flowLog, apiResponse, err
}

func (c *client) deleteFlowLog(ctx context.Context, deleteFn deleteFunc) (*sdkgo.APIResponse, error) {
	apiResponse, err := deleteFn()
	if err != nil {
		if apiResponse.HttpNotFound() {
			return apiResponse, ErrNotFound
		}
		return apiResponse, fmt.Errorf(logDeleteErr, c.parent, err)
	}
	if err = compute.WaitForRequest(ctx, c.ComputeClient, apiResponse); err != nil {
		return apiResponse, fmt.Errorf(logDeleteWaitErr, c.parent, err)
	}
	return apiResponse, nil
}
