package compute

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	sdkgo "github.com/ionos-cloud/sdk-go/v6"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

// DELETERequestIDAnnotationKey is the annotation that stores the ID of the last
// DELETE request sent for a resource. It is the Delete counterpart of
// POSTRequestIDAnnotationKey: while that request is still QUEUED or RUNNING,
// Delete waits for it instead of sending a new DELETE (see
// IsDeleteRequestPending and StoreDeleteRequestID).
const DELETERequestIDAnnotationKey = "ionos.cloud/delete-request-id"

const (
	// DeleteFailuresAnnotationKey stores the number of consecutive failed DELETE requests.
	DeleteFailuresAnnotationKey = "ionos.cloud/delete-failures"
	// DeleteFailedAtAnnotationKey stores the time (RFC3339, UTC) of the last failed DELETE request.
	DeleteFailedAtAnnotationKey = "ionos.cloud/delete-failed-at"
)

// DeleteHoldBase and DeleteHoldMax are the hold after the first failed DELETE and the longest hold.
var (
	DeleteHoldBase = 30 * time.Second
	DeleteHoldMax  = 5 * time.Minute
)

const deleteHoldJitter = 0.1

// DeleteAnnotationKeys returns the annotation keys the provider writes while deleting a resource.
func DeleteAnnotationKeys() []string {
	return []string{DELETERequestIDAnnotationKey, DeleteFailuresAnnotationKey, DeleteFailedAtAnnotationKey}
}

// RequestFailedError is returned when a Cloud API request ends with status FAILED.
type RequestFailedError struct {
	ID      string
	Message string
}

func (e *RequestFailedError) Error() string {
	return fmt.Sprintf("request %s failed: %s", e.ID, e.Message)
}

const (
	// fastPollWindowFactor and maxPollIntervalFactor are multiples of the base
	// poll interval: WaitForRequest polls at the base interval for the first
	// fastPollWindowFactor intervals, then backs off up to maxPollIntervalFactor.
	fastPollWindowFactor  = 60
	maxPollIntervalFactor = 10
	// defaultRequestPollInterval is the base interval in WaitForRequest when the
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
// status at a fixed PollInterval (1s by default). Here the first poll comes
// after one base interval (the client's PollInterval). Polls then stay at the
// base interval for the first 60 intervals of the wait; after that the wait
// between polls doubles, capped at 10 intervals (with the default
// PollInterval: every 1s for 60s, then 2s, 4s, 8s, 10s, 10s, ...).
// Long-running requests therefore no longer cause one status GET per second.
//
// Results:
//   - DONE: nil
//   - FAILED: a *RequestFailedError with the request message
//   - status GET answered 429, 502, 503 or 504, or got no HTTP response
//     because of a short network error (timeout, connection reset, refused
//     or closed, see isTransientPollError): keep polling. The SDK client
//     makes a single attempt per call, so the retry happens here, with the
//     backoff above, instead of failing the wait and sending the request
//     again on the next reconcile.
//   - status GET fails in any other way (404, 500, TLS/x509 error, ...) or
//     returns another non-200 code: error
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

	base := client.GetConfig().PollInterval
	if base <= 0 {
		base = defaultRequestPollInterval
	}
	start := time.Now()
	interval := base
	for {
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}

		done, err := pollRequest(ctx, client, reqID)
		if err != nil || done {
			return err
		}
		if time.Since(start) >= fastPollWindowFactor*base {
			interval = nextPollInterval(interval, maxPollIntervalFactor*base)
		}
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
		return false, newRequestFailedError(reqID, reqStatus)
	default:
		return false, nil
	}
}

func newRequestFailedError(reqID string, reqStatus sdkgo.RequestStatus) *RequestFailedError {
	e := &RequestFailedError{ID: reqID, Message: "<none>"}
	if reqStatus.Id != nil {
		e.ID = *reqStatus.Id
	}
	if reqStatus.Metadata != nil && reqStatus.Metadata.Message != nil {
		e.Message = *reqStatus.Metadata.Message
	}
	return e
}

// isTransientPollError reports whether a failed request status GET is worth
// another poll. If the API answered, that depends on the status code only (see
// isTransientPollStatus). If there is no HTTP response, it is true only for
// short network failures that a later poll can fix: a timeout (including a ctx
// deadline during the call), a connection reset or refused, or the server
// closing the connection (EOF). Other transport errors (TLS/x509, proxy auth,
// bad URL, DNS, ...) are not transient: every *url.Error is a net.Error, so
// checking for net.Error alone would match them too. If ctx is done, the
// caller's wait loop returns ctx.Err() right away.
func isTransientPollError(apiResponse *sdkgo.APIResponse, err error) bool {
	if apiResponse != nil && apiResponse.Response != nil {
		return isTransientPollStatus(apiResponse.StatusCode)
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	return errors.Is(err, syscall.ECONNRESET) || errors.Is(err, syscall.ECONNREFUSED) ||
		errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF)
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
// maxInterval. A wait that is already above the cap is kept as is.
func nextPollInterval(interval, maxInterval time.Duration) time.Duration {
	if interval >= maxInterval {
		return interval
	}
	return min(2*interval, maxInterval)
}

// IsDeleteRequestPending reports whether the DELETE request stored in the
// DELETERequestIDAnnotationKey annotation of obj is still being processed by
// the Cloud API. Delete calls it before sending a DELETE, so that a reconcile
// that runs while an earlier DELETE is QUEUED or RUNNING (for example because
// WaitForRequest ran out of time) waits for that request instead of sending a
// new one.
//
// It makes at most one request status GET. Results:
//   - no annotation: false, no GET
//   - QUEUED, RUNNING, another unfinished status or no status yet: true
//   - status GET answered 429, 502, 503 or 504, or got no HTTP response
//     because of a short network error (see isTransientPollError): true, the
//     status is checked again on the next reconcile
//   - DONE: false, the caller sends a new DELETE. The resource is gone, so
//     that DELETE gets a 404, which ErrorUnlessNotFound treats as success.
//   - FAILED: false and a *RequestFailedError, which HandleDeleteError
//     records as a failed DELETE.
//   - status GET answered 404 (the Cloud API does not know the request): false
//   - status GET failed in any other way (401, 403, 500, TLS/x509 error, ...)
//     or returned another non-200 code: error. The state of the request is
//     unknown, so no new DELETE is sent; the error shows in the Synced
//     condition and the reconcile is retried with backoff. Removing the
//     annotation makes the next reconcile send a new DELETE.
func IsDeleteRequestPending(ctx context.Context, client *sdkgo.APIClient, obj metav1.Object) (bool, error) {
	reqID := obj.GetAnnotations()[DELETERequestIDAnnotationKey]
	if reqID == "" {
		return false, nil
	}
	if client == nil {
		return false, errors.New(errAPIClientNil)
	}

	reqStatus, apiResponse, err := client.RequestsApi.RequestsStatusGet(ctx, reqID).Execute()
	if err != nil {
		switch {
		case isTransientPollError(apiResponse, err):
			return true, nil
		case apiResponse.HttpNotFound():
			return false, nil
		default:
			return false, fmt.Errorf("failed to retrieve delete request (%s) status: %w", reqID, err)
		}
	}
	if apiResponse == nil || apiResponse.Response == nil {
		return false, errors.New(errAPIResponseNil)
	}
	if apiResponse.StatusCode != http.StatusOK {
		return false, fmt.Errorf("failed to retrieve delete request (%s) status: received status code %d from API", reqID, apiResponse.StatusCode)
	}
	if reqStatus.Metadata == nil || reqStatus.Metadata.Status == nil {
		return true, nil
	}

	switch *reqStatus.Metadata.Status {
	case sdkgo.RequestStatusDone:
		return false, nil
	case sdkgo.RequestStatusFailed:
		return false, newRequestFailedError(reqID, reqStatus)
	default:
		return true, nil
	}
}

// StoreDeleteRequestID sets the ID of the DELETE request answered by
// apiResponse as the DELETERequestIDAnnotationKey annotation of cr and
// persists it right away with kube, before the caller waits for the request.
// The next reconcile then finds the request even if the wait times out (see
// IsDeleteRequestPending).
//
// It uses the managed reconciler's RetryingCriticalAnnotationUpdater, which
// retries on API server errors. The update reloads cr from the API server, so
// unsaved changes to its status are replaced by the stored status.
//
// Errors are logged, not returned: without the annotation, the next reconcile
// sends a new DELETE, as it did before. With a nil kube (unit tests) nothing
// is stored.
//
// It returns true only if the annotation was persisted. The annotation stays
// on cr in memory even if the update failed, so callers must use this result,
// not the annotation, to know whether the request ID is stored (see
// WaitForDeleteRequest).
func StoreDeleteRequestID(ctx context.Context, kube client.Client, cr client.Object, apiResponse *sdkgo.APIResponse, log logging.Logger) bool {
	reqID, err := ExtractRequestID(apiResponse)
	if err != nil {
		log.Info("Cannot store the DELETE request ID", "name", cr.GetName(), "error", err)
		return false
	}
	if kube == nil {
		log.Debug("No kube client, not storing the DELETE request ID", "name", cr.GetName(), "request-id", reqID)
		return false
	}
	meta.AddAnnotations(cr, map[string]string{DELETERequestIDAnnotationKey: reqID})
	if err := managed.NewRetryingCriticalAnnotationUpdater(kube).UpdateCriticalAnnotations(ctx, cr); err != nil {
		log.Info("Cannot store the DELETE request ID", "name", cr.GetName(), "request-id", reqID, "error", err)
		return false
	}
	return true
}

// deleteRequestWait is set once by ConfigureDeleteRequestWait and read by
// WaitForDeleteRequest. 0 means no limit of its own.
var deleteRequestWait time.Duration

// ConfigureDeleteRequestWait sets how long WaitForDeleteRequest waits for a
// stored DELETE request (--delete-request-wait). 0 turns the limit off. Call
// it once from main, before controller.Setup: it is read without a lock.
func ConfigureDeleteRequestWait(wait time.Duration) error {
	if wait < 0 {
		return fmt.Errorf("delete request wait %s must not be negative", wait)
	}
	deleteRequestWait = wait
	return nil
}

// WaitForDeleteRequest is WaitForRequest for the DELETE request of a Delete,
// with a limit of its own (see ConfigureDeleteRequestWait), so that a request
// that stays QUEUED or RUNNING does not hold a reconcile worker until the
// reconcile timeout.
//
// stored is the result of StoreDeleteRequestID for this request. The limit is
// only used when stored is true and the limit is above 0; otherwise it is the
// same as WaitForRequest.
//
// When the limit is hit while ctx is still alive, it returns nil: the Cloud
// API accepted the DELETE and its ID is in the DELETERequestIDAnnotationKey
// annotation, so the reconcile after the requeue checks that request with
// IsDeleteRequestPending instead of sending a new DELETE. Cancellation or the
// deadline of ctx itself (the reconcile timeout), FAILED and every other
// error are returned as by WaitForRequest.
//
// Use it only after a DELETE: an Update has no stored request ID, so ending
// its wait early would make the next reconcile send the PUT or PATCH again.
func WaitForDeleteRequest(ctx context.Context, client *sdkgo.APIClient, apiResponse *sdkgo.APIResponse, stored bool) error {
	if !stored || deleteRequestWait <= 0 {
		return WaitForRequest(ctx, client, apiResponse)
	}
	waitCtx, cancel := context.WithTimeout(ctx, deleteRequestWait)
	defer cancel()
	err := WaitForRequest(waitCtx, client, apiResponse)
	if errors.Is(err, context.DeadlineExceeded) && ctx.Err() == nil {
		return nil
	}
	return err
}

// CheckDeleteHold returns an error while obj waits after a failed DELETE; missing or invalid annotations mean no wait.
func CheckDeleteHold(obj metav1.Object, now time.Time) error {
	n, failedAt, ok := deleteFailureState(obj)
	if !ok {
		return nil
	}
	next := failedAt.Add(deleteHold(obj.GetUID(), n))
	if !now.Before(next) {
		return nil
	}
	return fmt.Errorf("%d DELETE request(s) failed, last at %s; next DELETE not before %s",
		n, failedAt.UTC().Format(time.RFC3339), next.UTC().Format(time.RFC3339))
}

// deleteFailureState reads the failed DELETE annotations of obj; ok is false if they are missing or invalid.
func deleteFailureState(obj metav1.Object) (n int, failedAt time.Time, ok bool) {
	n, nErr := strconv.Atoi(obj.GetAnnotations()[DeleteFailuresAnnotationKey])
	failedAt, tErr := time.Parse(time.RFC3339, obj.GetAnnotations()[DeleteFailedAtAnnotationKey])
	return n, failedAt, nErr == nil && n >= 1 && tErr == nil
}

// HandleDeleteError returns err; if err is a failed DELETE, it first records the failure on cr and adds when the next DELETE may be sent.
func HandleDeleteError(ctx context.Context, kube client.Client, cr client.Object, apiResponse *sdkgo.APIResponse, err error, log logging.Logger) error {
	if err == nil || !isDeleteFailure(apiResponse, err) {
		return err
	}
	n, convErr := strconv.Atoi(cr.GetAnnotations()[DeleteFailuresAnnotationKey])
	if convErr != nil || n < 0 {
		n = 0
	}
	n++
	failedAt := time.Now().UTC().Truncate(time.Second)
	annotations := map[string]string{
		DeleteFailuresAnnotationKey: strconv.Itoa(n),
		DeleteFailedAtAnnotationKey: failedAt.Format(time.RFC3339),
	}
	// The annotation updater cannot remove an annotation, so the request ID is emptied.
	if _, ok := cr.GetAnnotations()[DELETERequestIDAnnotationKey]; ok {
		annotations[DELETERequestIDAnnotationKey] = ""
	}
	meta.AddAnnotations(cr, annotations)
	if kube != nil {
		if uErr := managed.NewRetryingCriticalAnnotationUpdater(kube).UpdateCriticalAnnotations(ctx, cr); uErr != nil {
			log.Info("Cannot store the failed DELETE request", "name", cr.GetName(), "failures", n, "error", uErr)
		}
	}
	next := failedAt.Add(deleteHold(cr.GetUID(), n))
	return fmt.Errorf("%w; next DELETE not before %s", err, next.Format(time.RFC3339))
}

// isDeleteFailure is true for a FAILED request and for a 4xx answer other than 404, 408 and 429.
func isDeleteFailure(apiResponse *sdkgo.APIResponse, err error) bool {
	var failed *RequestFailedError
	if errors.As(err, &failed) {
		return true
	}
	if apiResponse == nil || apiResponse.Response == nil {
		return false
	}
	switch code := apiResponse.StatusCode; code {
	case http.StatusNotFound, http.StatusRequestTimeout, http.StatusTooManyRequests:
		return false
	default:
		return code >= 400 && code < 500
	}
}

// deleteHold doubles DeleteHoldBase per failure up to DeleteHoldMax, with a jitter fixed per uid and n.
func deleteHold(uid types.UID, n int) time.Duration {
	d := DeleteHoldBase
	for i := 1; i < n && d < DeleteHoldMax; i++ {
		d *= 2
	}
	d = min(d, DeleteHoldMax)
	h := fnv.New32a()
	_, _ = h.Write([]byte(string(uid) + "/" + strconv.Itoa(n)))
	f := float64(h.Sum32()) / float64(^uint32(0))
	return time.Duration(float64(d) * (1 - deleteHoldJitter + 2*deleteHoldJitter*f))
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
