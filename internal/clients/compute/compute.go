package compute

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

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

const (
	// maxRequestPollInterval caps the backoff between two request status polls
	// in WaitForRequest.
	maxRequestPollInterval = 30 * time.Second
	// defaultRequestPollInterval is the first wait in WaitForRequest when the
	// client has no positive PollInterval configured (same as the SDK default).
	defaultRequestPollInterval = time.Second
)

// ExtractRequestID extracts the IONOS Cloud request ID from the Location header
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

// WaitForRequest waits for the request referenced by the Location header of
// apiResponse to be DONE.
//
// It replaces the SDK's APIClient.WaitForRequest, which polls the request
// status at a fixed PollInterval (1s by default). Here the first poll is
// immediate, then the wait between polls starts at the client's PollInterval
// and doubles after each poll, capped at maxRequestPollInterval
// (1s, 2s, 4s, 8s, 16s, 30s, 30s, ... with the default PollInterval).
// Long-running requests therefore no longer cause one status GET per second.
//
// Results:
//   - DONE: nil
//   - FAILED: error with the request message
//   - status GET answered 429, 502, 503 or 504, or got no HTTP response
//     because of a network error: keep polling. The SDK client makes a
//     single attempt per call, so the retry happens here, with the backoff
//     above, instead of failing the wait and sending the request again on
//     the next reconcile.
//   - status GET fails in any other way (404, 500, ...) or returns another
//     non-200 code: error
//   - QUEUED, RUNNING or no status yet: keep polling
//   - ctx cancelled or past its deadline: error
func WaitForRequest(ctx context.Context, client *sdkgo.APIClient, apiResponse *sdkgo.APIResponse) error {
	if client == nil {
		return errors.New(errAPIClientNil)
	}
	if apiResponse == nil || apiResponse.Response == nil {
		return errors.New(errAPIResponseNil)
	}
	reqID, err := ExtractRequestID(apiResponse)
	if err != nil {
		return err
	}

	interval := client.GetConfig().PollInterval
	if interval <= 0 {
		interval = defaultRequestPollInterval
	}
	for {
		done, err := pollRequest(ctx, client, reqID)
		if err != nil || done {
			return err
		}

		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
		interval = nextPollInterval(interval)
	}
}

// pollRequest gets the status of a request once. It returns true if the
// request is DONE, an error if the request FAILED or its status could not be
// read, and false otherwise. A transient failure of the status GET (see
// isTransientPollError) is not an error: it returns false, so the caller polls
// again.
func pollRequest(ctx context.Context, client *sdkgo.APIClient, reqID string) (bool, error) {
	reqStatus, apiResponse, err := client.RequestsApi.RequestsStatusGet(ctx, reqID).Execute()
	if err != nil {
		if isTransientPollError(apiResponse, err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to retrieve request (%s) status: %w", reqID, err)
	}
	if apiResponse == nil || apiResponse.Response == nil {
		return false, errors.New(errAPIResponseNil)
	}
	if apiResponse.StatusCode != http.StatusOK {
		return false, fmt.Errorf("failed to retrieve request (%s) status: received status code %d from API", reqID, apiResponse.StatusCode)
	}
	if reqStatus.Metadata == nil || reqStatus.Metadata.Status == nil {
		return false, nil
	}

	switch *reqStatus.Metadata.Status {
	case sdkgo.RequestStatusDone:
		return true, nil
	case sdkgo.RequestStatusFailed:
		id, message := reqID, "<none>"
		if reqStatus.Id != nil {
			id = *reqStatus.Id
		}
		if reqStatus.Metadata.Message != nil {
			message = *reqStatus.Metadata.Message
		}
		return false, fmt.Errorf("request %s failed: %s", id, message)
	default:
		return false, nil
	}
}

// isTransientPollError reports whether a failed request status GET is worth
// another poll. If the API answered, that depends on the status code only (see
// isTransientPollStatus). If there is no HTTP response, it is true for network
// errors (connection reset, timeout, ...): the SDK returns the *url.Error of
// the HTTP client, which is a net.Error. If ctx is done, the caller's wait
// loop returns ctx.Err() right away.
func isTransientPollError(apiResponse *sdkgo.APIResponse, err error) bool {
	if apiResponse != nil && apiResponse.Response != nil {
		return isTransientPollStatus(apiResponse.StatusCode)
	}
	var netErr net.Error
	return errors.As(err, &netErr)
}

// isTransientPollStatus reports whether a request status GET that returned
// code should be retried: rate limited (429) or a gateway / availability
// error (502, 503, 504). These are the codes the SDK itself retries.
func isTransientPollStatus(code int) bool {
	switch code {
	case http.StatusTooManyRequests,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

// nextPollInterval doubles the wait between two request status polls, up to
// maxRequestPollInterval. A wait that is already above the cap (a large
// configured PollInterval) is kept as is.
func nextPollInterval(interval time.Duration) time.Duration {
	if interval >= maxRequestPollInterval {
		return interval
	}
	return min(2*interval, maxRequestPollInterval)
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
