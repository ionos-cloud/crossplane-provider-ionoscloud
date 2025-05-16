package flowlog

import (
	"context"
	"errors"
	"fmt"

	sdkgo "github.com/ionos-cloud/sdk-go/v6"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute"
)

var (
	// ErrNotFound flow log was not found
	ErrNotFound = errors.New("flow log not found")

	errGetByID    = errors.New("failed to get flow log by ID")
	errList       = errors.New("failed to get flow logs list")
	errCreate     = errors.New("failed to create flow log")
	errCreateWait = errors.New("error while waiting for flow log create request")
	errUpdate     = errors.New("failed to update flow log")
	errUpdateWait = errors.New("error while waiting for flow log update request")
	errDelete     = errors.New("failed to delete flow log")
	errDeleteWait = errors.New("error while waiting for flow log delete request")
)

type listFunc = func() (sdkgo.FlowLogs, *sdkgo.APIResponse, error)
type byIDFunc = func() (sdkgo.FlowLog, *sdkgo.APIResponse, error)
type createOrPatchFunc = func() (sdkgo.FlowLog, *sdkgo.APIResponse, error)
type deleteFunc = func() (*sdkgo.APIResponse, error)

// flowLogClient common CRUD interface for flow log
type flowLogClient interface {
	checkDuplicateFlowLog(flowLogName string, listFn listFunc) (string, error)
	getFlowLogByID(byIDFn byIDFunc) (sdkgo.FlowLog, error)
	createFlowLog(ctx context.Context, createFn createOrPatchFunc) (sdkgo.FlowLog, error)
	updateFlowLog(ctx context.Context, updateFn createOrPatchFunc) (sdkgo.FlowLog, error)
	deleteFlowLog(ctx context.Context, deleteFn deleteFunc) error
}

// client implements common functionality for IONOS flow log
type client struct {
	*clients.IonosServices
	parent string
}

func (c *client) checkDuplicateFlowLog(flowLogName string, listFn listFunc) (string, error) {
	flowLogs, _, err := listFn()
	if err != nil {
		return "", fmt.Errorf("%w: %w", errList, err)
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

func (c *client) getFlowLogByID(byIDFn byIDFunc) (sdkgo.FlowLog, error) {
	flowLog, apiResponse, err := byIDFn()
	if err != nil {
		if apiResponse.HttpNotFound() {
			return flowLog, ErrNotFound
		}
		return flowLog, fmt.Errorf("%w: %w", errGetByID, err)
	}
	return flowLog, nil
}

func (c *client) createFlowLog(ctx context.Context, createFn createOrPatchFunc) (sdkgo.FlowLog, error) {
	flowLog, apiResponse, err := createFn()
	if err != nil {
		return sdkgo.FlowLog{}, fmt.Errorf("%w: %w", errCreate, err)
	}
	if err = compute.WaitForRequest(ctx, c.IonosServices.ComputeClient, apiResponse); err != nil {
		return sdkgo.FlowLog{}, fmt.Errorf("%w: %w", errCreateWait, err)
	}
	return flowLog, nil
}

func (c *client) updateFlowLog(ctx context.Context, updateFn createOrPatchFunc) (sdkgo.FlowLog, error) {
	flowLog, apiResponse, err := updateFn()
	if err != nil {
		return sdkgo.FlowLog{}, fmt.Errorf("%w: %w", errUpdate, err)
	}
	if err = compute.WaitForRequest(ctx, c.IonosServices.ComputeClient, apiResponse); err != nil {
		return sdkgo.FlowLog{}, fmt.Errorf("%w: %w", errUpdateWait, err)
	}
	return flowLog, nil
}

func (c *client) deleteFlowLog(ctx context.Context, deleteFn deleteFunc) error {
	apiResponse, err := deleteFn()
	if err != nil {
		if apiResponse.HttpNotFound() {
			return ErrNotFound
		}
		return fmt.Errorf("%w: %w", errDelete, err)
	}
	if err = compute.WaitForRequest(ctx, c.IonosServices.ComputeClient, apiResponse); err != nil {
		return fmt.Errorf("%w: %w", errDeleteWait, err)
	}
	return nil
}
