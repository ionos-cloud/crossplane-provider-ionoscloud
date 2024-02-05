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

// WaitForRequest waits for the request to be DONE.
var WaitForRequest requestWaiter = requestWait

// requestWaiter defines a type to wait for requests.
type requestWaiter func(ctx context.Context, client *sdkgo.APIClient, apiResponse *sdkgo.APIResponse) error

// requestWait is the default requestWaiter.
func requestWait(ctx context.Context, client *sdkgo.APIClient, apiResponse *sdkgo.APIResponse) error {
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

	if apiResponse != nil && apiResponse.Response != nil {
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
