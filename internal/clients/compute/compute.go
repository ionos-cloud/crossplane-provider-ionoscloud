package compute

import (
	"context"
	"errors"
	"fmt"
	"strings"

	sdkgo "github.com/ionos-cloud/sdk-go/v6"
)

// States of Compute Resources
const (
	AVAILABLE  = "AVAILABLE"
	BUSY       = "BUSY"
	ACTIVE     = "ACTIVE"
	INACTIVE   = "INACTIVE"
	UPDATING   = "UPDATING"
	DESTROYING = "DESTROYING"

	// Server Power States
	RUNNING   = "RUNNING"
	SUSPENDED = "SUSPENDED"
	SHUTOFF   = "SHUTOFF"
)

const (
	errAPIResponse    = "%w, API Response Status: %v"
	errAPIResponseNil = "error: APIResponse must not be nil"
	errAPIClientNil   = "error: APIClient must not be nil"

	// RequestHeader is related to the APIResponse Header
	requestHeader = "Location"
)

const POSTRequestIDAnnotationKey = "ionos.cloud/post-request-id"

// ExtractRequestID extracts the IONOS CLOUD request ID from the Location header
// of an API response, which has the following format:
//
//	https://api.ionos.com/cloudapi/v6/requests/{requestID}/status
//
// This function parses out the {requestID} segment from that URL.
//
// It is called after create operations (e.g. CreateLan) to capture the request
// ID, which is then stored as a POSTRequestIDAnnotationKey annotation on the CR
// for later polling via IsRequestDone.
//
// Returns (requestID, nil) on success, or ("", error) if the Location header is
// missing, malformed, or contains an empty request ID.
func ExtractRequestID(apiResponse *sdkgo.APIResponse) (string, error) {
	if apiResponse == nil || apiResponse.Response == nil {
		return "", errors.New(errAPIResponseNil)
	}
	requestStatusURL := apiResponse.Header.Get(requestHeader)
	if requestStatusURL == "" {
		return "", fmt.Errorf("request status URL is empty")
	}

	var requestID string
	_, requestID, found := strings.Cut(requestStatusURL, "/requests/")
	if !found {
		return "", fmt.Errorf("request status URL is malformed, expected path '/requests/', got: %s", requestStatusURL)
	}
	if requestID == "" {
		return "", fmt.Errorf("request ID is empty in the request status URL: %s", requestStatusURL)
	}

	requestID = strings.TrimSuffix(requestID, "/status")

	return requestID, nil
}

// IsRequestDone checks whether a previously initiated IONOS
// Cloud API request has completed by polling its status endpoint.
//
// Parameters:
//   - ctx: context for cancellation
//   - client: the IONOS SDK client used to call the Requests API
//   - reqID: the request ID
//
// Return value semantics:
//   - (true, nil)  — request completed successfully (status DONE).
//   - (false, nil)  — request is still in progress (QUEUED/RUNNING), or the
//     request was not found (404). Callers should retry later.
//   - (false, error) — request failed (status FAILED, with message) or an
//     unexpected API/metadata error occurred. Callers should surface this error.
//
// A 404 response is treated as "not done, no error" to cover the case where the
// IONOS API has lost track of the request. The caller will keep retrying; manual
// intervention (removing the POSTRequestIDAnnotationKey annotation from the CR)
// may be needed to unstick the resource.
//
// This function is called during the Create reconciliation path when a
// POSTRequestIDAnnotationKey annotation exists on the CR, indicating a previous
// Create issued a request that has not yet been confirmed as complete.
func IsRequestDone(ctx context.Context, client *sdkgo.APIClient, reqID string) (bool, error) {
	reqStatus, apiResponse, err := client.RequestsApi.RequestsStatusGet(ctx, reqID).Execute()
	if err != nil {
		if apiResponse != nil && apiResponse.HttpNotFound() {
			return false, nil
		}

		return false, fmt.Errorf("failed to retrieve request (%s) status: %w", reqID, err)
	}

	if reqStatus.Metadata == nil || reqStatus.Metadata.Status == nil {
		return false, fmt.Errorf("failed to retrieve request (%s) status from metadata", reqID)
	}

	status := *reqStatus.Metadata.Status
	switch status {
	case sdkgo.RequestStatusDone:
		return true, nil
	case sdkgo.RequestStatusFailed:
		errMsg := fmt.Sprintf("request (%s) status is failed", reqID)

		msg := reqStatus.Metadata.Message
		if msg != nil {
			errMsg = fmt.Sprintf("%s (%s)", errMsg, *msg)
		}
		return false, errors.New(errMsg)
	case sdkgo.RequestStatusQueued, sdkgo.RequestStatusRunning:
		return false, nil
	default:
		return false, fmt.Errorf("request (%s) status is unknown", reqID)
	}
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
