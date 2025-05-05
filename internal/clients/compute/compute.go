package compute

import (
	"context"
	"errors"
	"fmt"

	sdkgo "github.com/ionos-cloud/sdk-go/v6"
)

// States of Compute Resources
const (
	AVAILABLE  = "AVAILABLE"
	BUSY       = "BUSY"
	ACTIVE     = "ACTIVE"
	UPDATING   = "UPDATING"
	DESTROYING = "DESTROYING"
)

const (
	errAPIResponse    = "%w, API Response Status: %v"
	errAPIResponseNil = "error: APIResponse must not be nil"
	errAPIClientNil   = "error: APIClient must not be nil"

	// RequestHeader is related to the APIResponse Header
	requestHeader = "Location"
)

func IsRequestDone(ctx context.Context, client *sdkgo.APIClient, targetID, method string) (bool, error) {
	reqs, _, err := client.RequestsApi.RequestsGet(ctx).FilterRequestStatus(targetID).FilterMethod(method).Limit(1).Execute()
	if err != nil {
		return false, fmt.Errorf("failed to get %s request for resource %s. error: %w", method, targetID, err)
	}

	if len(*reqs.Items) == 0 {
		return false, fmt.Errorf("no %s request found for resource %s", method, targetID)
	}

	// we retrieve only the most recent request that matches the criteria
	for _, req := range *reqs.Items {
		status := req.Metadata.RequestStatus.Metadata.Status
		if *status == sdkgo.RequestStatusDone {
			return true, nil
		}
		if *status == sdkgo.RequestStatusFailed {
			return false, fmt.Errorf("%s request for resource %s failed", method, targetID)
		}
	}

	return false, nil
}

// WaitForRequest waits for the request to be DONE
func WaitForRequest(ctx context.Context, client *sdkgo.APIClient, apiResponse *sdkgo.APIResponse) error {
	if client != nil {
		if apiResponse != nil && apiResponse.Response != nil {
			if _, err := client.WaitForRequest(ctx, apiResponse.Response.Header.Get(requestHeader)); err != nil {
				return err
			}
			return nil
		}
		return errors.New(errAPIResponseNil)
	}
	return errors.New(errAPIClientNil)
}

// ErrorUnlessNotFound returns an error with status code info, unless the status code is 404
func ErrorUnlessNotFound(apiResponse *sdkgo.APIResponse, retErr error) error {
	if apiResponse != nil && apiResponse.Response != nil && apiResponse.StatusCode >= 300 {
		retErr = fmt.Errorf(errAPIResponse, retErr, apiResponse.Status)
		if apiResponse.HttpNotFound() {
			retErr = nil
		}
	}
	return retErr
}

// AddAPIResponseInfo adds APIResponse status info to an existing error
func AddAPIResponseInfo(apiResponse *sdkgo.APIResponse, retErr error) error {
	if apiResponse != nil && apiResponse.Response != nil {
		retErr = fmt.Errorf(errAPIResponse, retErr, apiResponse.Response.Status)
	}
	return retErr
}
