package compute

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/crossplane/crossplane-runtime/pkg/logging"
	sdkgo "github.com/ionos-cloud/sdk-go/v6"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	testRequestID         = "req-1"
	testRequestStatusPath = "/cloudapi/v6/requests/" + testRequestID + "/status"
	testPollInterval      = 10 * time.Millisecond
	simulatedFailure      = "simulated failure"
)

// statusResponse is one answer of the fake request status endpoint. An empty
// status means the response has no metadata. Code 0 means the server closes
// the connection without an answer (a network error for the client).
type statusResponse struct {
	code    int
	status  string
	message string
}

// requestStatusServer answers request status GETs with the given responses in
// order. The last response is repeated.
type requestStatusServer struct {
	*httptest.Server

	t         *testing.T
	mu        sync.Mutex
	responses []statusResponse
	calls     []time.Time
	paths     []string
}

func newRequestStatusServer(t *testing.T, responses ...statusResponse) *requestStatusServer {
	t.Helper()
	s := &requestStatusServer{t: t, responses: responses}
	s.Server = httptest.NewServer(http.HandlerFunc(s.handle))
	t.Cleanup(s.Close)
	return s
}

func (s *requestStatusServer) handle(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	resp := s.responses[min(len(s.calls), len(s.responses)-1)]
	s.calls = append(s.calls, time.Now())
	s.paths = append(s.paths, r.URL.Path)
	s.mu.Unlock()

	if resp.code == 0 {
		hj, ok := w.(http.Hijacker)
		if !ok {
			s.t.Errorf("response writer does not support hijacking")
			return
		}
		conn, _, err := hj.Hijack()
		if err != nil {
			s.t.Errorf("hijack connection: %v", err)
			return
		}
		if err := conn.Close(); err != nil {
			s.t.Errorf("close connection: %v", err)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.code)
	if resp.code != http.StatusOK {
		if err := json.NewEncoder(w).Encode(map[string]any{
			"httpStatus": resp.code,
			"messages":   []map[string]any{{"errorCode": "000", "message": "simulated error"}},
		}); err != nil {
			s.t.Errorf("encode error response: %v", err)
		}
		return
	}
	body := map[string]any{"id": testRequestID}
	if resp.status != "" {
		metadata := map[string]any{"status": resp.status}
		if resp.message != "" {
			metadata["message"] = resp.message
		}
		body["metadata"] = metadata
	}
	if err := json.NewEncoder(w).Encode(body); err != nil {
		s.t.Errorf("encode status response: %v", err)
	}
}

func (s *requestStatusServer) callTimes() []time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]time.Time(nil), s.calls...)
}

func (s *requestStatusServer) callPaths() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.paths...)
}

func newTestClient(serverURL string) *sdkgo.APIClient {
	cfg := sdkgo.NewConfiguration("", "", "test-token", serverURL)
	cfg.PollInterval = testPollInterval
	// Like the provider's compute client: one attempt per call, no SDK retries.
	cfg.MaxRetries = 1
	return sdkgo.NewAPIClient(cfg)
}

func acceptedResponse(location string) *sdkgo.APIResponse {
	header := http.Header{}
	if location != "" {
		header.Set(requestHeader, location)
	}
	return &sdkgo.APIResponse{
		Response: &http.Response{StatusCode: http.StatusAccepted, Header: header},
	}
}

func TestWaitForRequestBacksOffUntilDone(t *testing.T) {
	responses := []statusResponse{
		{code: http.StatusOK, status: sdkgo.RequestStatusQueued},
		{code: http.StatusOK},
	}
	for len(responses) < fastPollWindowFactor+4 {
		responses = append(responses, statusResponse{code: http.StatusOK, status: sdkgo.RequestStatusRunning})
	}
	responses = append(responses, statusResponse{code: http.StatusOK, status: sdkgo.RequestStatusDone})
	s := newRequestStatusServer(t, responses...)

	err := WaitForRequest(context.Background(), newTestClient(s.URL), acceptedResponse(s.URL+testRequestStatusPath))
	require.NoError(t, err)

	calls := s.callTimes()
	require.Len(t, calls, len(responses))
	for _, p := range s.callPaths() {
		assert.Equal(t, testRequestStatusPath, p)
	}
	for i := 1; i < len(calls); i++ {
		assert.GreaterOrEqual(t, calls[i].Sub(calls[i-1]), testPollInterval, "wait before poll %d", i+1)
	}
	// The window ends by poll fastPollWindowFactor; then the waits double up to the cap.
	want := []time.Duration{2 * testPollInterval, 4 * testPollInterval, 8 * testPollInterval, maxPollIntervalFactor * testPollInterval}
	for i, w := range want {
		n := fastPollWindowFactor + i
		assert.GreaterOrEqual(t, calls[n].Sub(calls[n-1]), w, "wait before poll %d", n+1)
	}
}

func TestWaitForRequestNoImmediatePoll(t *testing.T) {
	s := newRequestStatusServer(t, statusResponse{code: http.StatusOK, status: sdkgo.RequestStatusDone})

	start := time.Now()
	err := WaitForRequest(context.Background(), newTestClient(s.URL), acceptedResponse(s.URL+testRequestStatusPath))
	require.NoError(t, err)

	calls := s.callTimes()
	require.Len(t, calls, 1)
	assert.GreaterOrEqual(t, calls[0].Sub(start), testPollInterval)
}

func TestWaitForRequestFastWindow(t *testing.T) {
	responses := make([]statusResponse, 0, 11)
	for range 10 {
		responses = append(responses, statusResponse{code: http.StatusOK, status: sdkgo.RequestStatusRunning})
	}
	responses = append(responses, statusResponse{code: http.StatusOK, status: sdkgo.RequestStatusDone})
	s := newRequestStatusServer(t, responses...)

	start := time.Now()
	err := WaitForRequest(context.Background(), newTestClient(s.URL), acceptedResponse(s.URL+testRequestStatusPath))
	require.NoError(t, err)

	require.Len(t, s.callTimes(), len(responses))
	assert.Less(t, time.Since(start), fastPollWindowFactor*testPollInterval)
}

func TestWaitForRequestFailed(t *testing.T) {
	s := newRequestStatusServer(t,
		statusResponse{code: http.StatusOK, status: sdkgo.RequestStatusRunning},
		statusResponse{code: http.StatusOK, status: sdkgo.RequestStatusFailed, message: simulatedFailure},
	)

	err := WaitForRequest(context.Background(), newTestClient(s.URL), acceptedResponse(s.URL+testRequestStatusPath))
	require.EqualError(t, err, "request "+testRequestID+" failed: "+simulatedFailure)
	var failed *RequestFailedError
	require.ErrorAs(t, err, &failed)
	assert.Len(t, s.callTimes(), 2)
}

func TestWaitForRequestStatusGetError(t *testing.T) {
	for _, code := range []int{http.StatusNotFound, http.StatusInternalServerError} {
		t.Run(http.StatusText(code), func(t *testing.T) {
			s := newRequestStatusServer(t, statusResponse{code: code})

			err := WaitForRequest(context.Background(), newTestClient(s.URL), acceptedResponse(s.URL+testRequestStatusPath))
			require.ErrorContains(t, err, "failed to retrieve request ("+testRequestID+") status")
			assert.Len(t, s.callTimes(), 1)
		})
	}
}

func TestWaitForRequestTransientStatusGetError(t *testing.T) {
	for _, tc := range []struct {
		name  string
		first statusResponse
	}{
		{name: "TooManyRequests", first: statusResponse{code: http.StatusTooManyRequests}},
		{name: "BadGateway", first: statusResponse{code: http.StatusBadGateway}},
		{name: "ServiceUnavailable", first: statusResponse{code: http.StatusServiceUnavailable}},
		{name: "GatewayTimeout", first: statusResponse{code: http.StatusGatewayTimeout}},
		{name: "ConnectionClosed", first: statusResponse{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := newRequestStatusServer(t,
				tc.first,
				statusResponse{code: http.StatusOK, status: sdkgo.RequestStatusRunning},
				statusResponse{code: http.StatusOK, status: sdkgo.RequestStatusDone},
			)

			err := WaitForRequest(context.Background(), newTestClient(s.URL), acceptedResponse(s.URL+testRequestStatusPath))
			require.NoError(t, err)
			assert.Len(t, s.callTimes(), 3)
		})
	}
}

func TestWaitForRequestTransientThenDone(t *testing.T) {
	s := newRequestStatusServer(t,
		statusResponse{code: http.StatusServiceUnavailable},
		statusResponse{code: http.StatusOK, status: sdkgo.RequestStatusDone},
	)

	err := WaitForRequest(context.Background(), newTestClient(s.URL), acceptedResponse(s.URL+testRequestStatusPath))
	require.NoError(t, err)
	assert.Len(t, s.callTimes(), 2)
}

func TestWaitForRequestTransientUntilDeadline(t *testing.T) {
	s := newRequestStatusServer(t, statusResponse{code: http.StatusTooManyRequests})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	start := time.Now()
	err := WaitForRequest(ctx, newTestClient(s.URL), acceptedResponse(s.URL+testRequestStatusPath))
	require.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Less(t, time.Since(start), 5*time.Second)
	assert.GreaterOrEqual(t, len(s.callTimes()), 2)
}

func TestIsTransientPollStatus(t *testing.T) {
	for _, code := range []int{http.StatusTooManyRequests, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout} {
		assert.True(t, isTransientPollStatus(code), code)
	}
	for _, code := range []int{http.StatusOK, http.StatusBadRequest, http.StatusUnauthorized, http.StatusNotFound, http.StatusInternalServerError} {
		assert.False(t, isTransientPollStatus(code), code)
	}
}

func TestIsTransientPollError(t *testing.T) {
	urlErr := func(err error) error { return &url.Error{Op: "Get", URL: "https://api.example.com", Err: err} }
	for _, tc := range []struct {
		name      string
		err       error
		transient bool
	}{
		{name: "PlainError", err: errors.New("simulated error")},
		{name: "UnsupportedScheme", err: urlErr(errors.New(`unsupported protocol scheme ""`))},
		{name: "Timeout", err: urlErr(context.DeadlineExceeded), transient: true},
		{name: "ConnectionReset", err: fmt.Errorf("read: %w", syscall.ECONNRESET), transient: true},
		{name: "ConnectionRefused", err: urlErr(&net.OpError{Op: "dial", Net: "tcp", Err: syscall.ECONNREFUSED}), transient: true},
		{name: "EOF", err: urlErr(io.EOF), transient: true},
		{name: "UnexpectedEOF", err: urlErr(io.ErrUnexpectedEOF), transient: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.transient, isTransientPollError(&sdkgo.APIResponse{}, tc.err))
		})
	}
}

func TestRequestStatusTLSErrorIsNotTransient(t *testing.T) {
	// The SDK's default HTTP client does not trust the test server's
	// certificate, so every status GET fails in the TLS handshake.
	s := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Error("unexpected request: the TLS handshake must fail")
	}))
	t.Cleanup(s.Close)
	client := newTestClient(s.URL)

	err := WaitForRequest(context.Background(), client, acceptedResponse(s.URL+testRequestStatusPath))
	require.ErrorContains(t, err, "failed to retrieve request ("+testRequestID+") status")

	pending, err := IsDeleteRequestPending(context.Background(), client, deleteRequestObject(testRequestID))
	require.ErrorContains(t, err, "failed to retrieve delete request ("+testRequestID+") status")
	assert.False(t, pending)
}

func TestWaitForRequestContextDeadline(t *testing.T) {
	s := newRequestStatusServer(t, statusResponse{code: http.StatusOK, status: sdkgo.RequestStatusRunning})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	start := time.Now()
	err := WaitForRequest(ctx, newTestClient(s.URL), acceptedResponse(s.URL+testRequestStatusPath))
	require.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Less(t, time.Since(start), 5*time.Second)
}

func TestWaitForRequestInvalidInput(t *testing.T) {
	s := newRequestStatusServer(t, statusResponse{code: http.StatusOK, status: sdkgo.RequestStatusDone})
	client := newTestClient(s.URL)

	require.EqualError(t, WaitForRequest(context.Background(), nil, acceptedResponse(s.URL+testRequestStatusPath)), errAPIClientNil)
	require.EqualError(t, WaitForRequest(context.Background(), client, nil), errAPIResponseNil)
	require.EqualError(t, WaitForRequest(context.Background(), client, &sdkgo.APIResponse{}), errAPIResponseNil)
	require.Error(t, WaitForRequest(context.Background(), client, acceptedResponse("")))
	assert.Empty(t, s.callTimes())
}

func TestNextPollInterval(t *testing.T) {
	got := make([]time.Duration, 0, 7)
	interval := time.Second
	for range 7 {
		got = append(got, interval)
		interval = nextPollInterval(interval, maxPollIntervalFactor*time.Second)
	}
	want := []time.Duration{
		1 * time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second,
		10 * time.Second, 10 * time.Second, 10 * time.Second,
	}
	assert.Equal(t, want, got)

	// An interval above the cap is kept.
	assert.Equal(t, time.Minute, nextPollInterval(time.Minute, 10*time.Second))
}

// deleteRequestObject returns object metadata with reqID as the stored DELETE
// request ID.
func deleteRequestObject(reqID string) *metav1.ObjectMeta {
	return &metav1.ObjectMeta{Annotations: map[string]string{DELETERequestIDAnnotationKey: reqID}}
}

func TestIsDeleteRequestPending(t *testing.T) {
	for _, tc := range []struct {
		name    string
		resp    statusResponse
		pending bool
		wantErr string
	}{
		{name: "Queued", resp: statusResponse{code: http.StatusOK, status: sdkgo.RequestStatusQueued}, pending: true},
		{name: "Running", resp: statusResponse{code: http.StatusOK, status: sdkgo.RequestStatusRunning}, pending: true},
		{name: "NoStatus", resp: statusResponse{code: http.StatusOK}, pending: true},
		{name: "Done", resp: statusResponse{code: http.StatusOK, status: sdkgo.RequestStatusDone}},
		{name: "Failed", resp: statusResponse{code: http.StatusOK, status: sdkgo.RequestStatusFailed, message: simulatedFailure},
			wantErr: "request " + testRequestID + " failed: " + simulatedFailure},
		{name: "NotFound", resp: statusResponse{code: http.StatusNotFound}},
		{name: "TooManyRequests", resp: statusResponse{code: http.StatusTooManyRequests}, pending: true},
		{name: "ServiceUnavailable", resp: statusResponse{code: http.StatusServiceUnavailable}, pending: true},
		{name: "ConnectionClosed", resp: statusResponse{}, pending: true},
		{name: "Unauthorized", resp: statusResponse{code: http.StatusUnauthorized},
			wantErr: "failed to retrieve delete request (" + testRequestID + ") status"},
		{name: "InternalServerError", resp: statusResponse{code: http.StatusInternalServerError},
			wantErr: "failed to retrieve delete request (" + testRequestID + ") status"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := newRequestStatusServer(t, tc.resp)

			pending, err := IsDeleteRequestPending(context.Background(), newTestClient(s.URL), deleteRequestObject(testRequestID))
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
			} else {
				require.NoError(t, err)
			}
			var failed *RequestFailedError
			assert.Equal(t, tc.resp.status == sdkgo.RequestStatusFailed, errors.As(err, &failed))
			assert.Equal(t, tc.pending, pending)
			assert.Equal(t, []string{testRequestStatusPath}, s.callPaths())
		})
	}
}

func TestIsDeleteRequestPendingWithoutAnnotation(t *testing.T) {
	s := newRequestStatusServer(t, statusResponse{code: http.StatusOK, status: sdkgo.RequestStatusRunning})
	apiClient := newTestClient(s.URL)

	for _, obj := range []*metav1.ObjectMeta{
		{},
		{Annotations: map[string]string{POSTRequestIDAnnotationKey: testRequestID}},
		deleteRequestObject(""),
	} {
		pending, err := IsDeleteRequestPending(context.Background(), apiClient, obj)
		require.NoError(t, err)
		assert.False(t, pending)
	}
	assert.Empty(t, s.callTimes())

	// Without a client, only a stored request ID is an error.
	pending, err := IsDeleteRequestPending(context.Background(), nil, &metav1.ObjectMeta{})
	require.NoError(t, err)
	assert.False(t, pending)
	_, err = IsDeleteRequestPending(context.Background(), nil, deleteRequestObject(testRequestID))
	require.EqualError(t, err, errAPIClientNil)
}

func TestStoreDeleteRequestID(t *testing.T) {
	ctx := context.Background()
	key := client.ObjectKey{Namespace: "default", Name: "cr"}
	kube := fake.NewClientBuilder().
		WithObjects(&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Namespace: key.Namespace, Name: key.Name}}).
		Build()
	cr := &corev1.ConfigMap{}
	require.NoError(t, kube.Get(ctx, key, cr))

	assert.True(t, StoreDeleteRequestID(ctx, kube, cr, acceptedResponse("https://api.ionos.com/cloudapi/v6/requests/"+testRequestID+"/status"), logging.NewNopLogger()))
	assert.Equal(t, testRequestID, cr.GetAnnotations()[DELETERequestIDAnnotationKey])

	stored := &corev1.ConfigMap{}
	require.NoError(t, kube.Get(ctx, key, stored))
	assert.Equal(t, testRequestID, stored.GetAnnotations()[DELETERequestIDAnnotationKey])
}

func TestStoreDeleteRequestIDSkipped(t *testing.T) {
	ctx := context.Background()
	key := client.ObjectKey{Namespace: "default", Name: "cr"}
	kube := fake.NewClientBuilder().
		WithObjects(&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Namespace: key.Namespace, Name: key.Name}}).
		Build()

	// No request ID in the response: nothing is stored.
	cr := &corev1.ConfigMap{}
	require.NoError(t, kube.Get(ctx, key, cr))
	assert.False(t, StoreDeleteRequestID(ctx, kube, cr, acceptedResponse(""), logging.NewNopLogger()))
	stored := &corev1.ConfigMap{}
	require.NoError(t, kube.Get(ctx, key, stored))
	assert.NotContains(t, stored.GetAnnotations(), DELETERequestIDAnnotationKey)

	// No kube client: nothing is set, and no panic.
	cr = &corev1.ConfigMap{}
	assert.False(t, StoreDeleteRequestID(ctx, nil, cr, acceptedResponse("https://api.ionos.com/cloudapi/v6/requests/"+testRequestID+"/status"), logging.NewNopLogger()))
	assert.NotContains(t, cr.GetAnnotations(), DELETERequestIDAnnotationKey)
}

func TestStoreDeleteRequestIDUpdateFails(t *testing.T) {
	ctx := context.Background()
	// The object is not in the fake API server, so the update fails.
	kube := fake.NewClientBuilder().Build()
	cr := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "cr"}}

	assert.False(t, StoreDeleteRequestID(ctx, kube, cr, acceptedResponse("https://api.ionos.com/cloudapi/v6/requests/"+testRequestID+"/status"), logging.NewNopLogger()))
}

// setDeleteRequestWait sets the package wide delete request wait for one test.
func setDeleteRequestWait(t *testing.T, wait time.Duration) {
	t.Helper()
	old := deleteRequestWait
	require.NoError(t, ConfigureDeleteRequestWait(wait))
	t.Cleanup(func() { deleteRequestWait = old })
}

func TestConfigureDeleteRequestWait(t *testing.T) {
	setDeleteRequestWait(t, 0)
	require.Error(t, ConfigureDeleteRequestWait(-time.Second))
	assert.Zero(t, deleteRequestWait)
	require.NoError(t, ConfigureDeleteRequestWait(time.Minute))
	assert.Equal(t, time.Minute, deleteRequestWait)
}

func TestWaitForDeleteRequestStopsAfterWait(t *testing.T) {
	setDeleteRequestWait(t, 100*time.Millisecond)
	s := newRequestStatusServer(t, statusResponse{code: http.StatusOK, status: sdkgo.RequestStatusRunning})

	start := time.Now()
	err := WaitForDeleteRequest(context.Background(), newTestClient(s.URL), acceptedResponse(s.URL+testRequestStatusPath), true)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, time.Since(start), 100*time.Millisecond)
	assert.Less(t, time.Since(start), 5*time.Second)
	assert.GreaterOrEqual(t, len(s.callTimes()), 2)
}

func TestWaitForDeleteRequestParentDeadline(t *testing.T) {
	setDeleteRequestWait(t, time.Minute)
	s := newRequestStatusServer(t, statusResponse{code: http.StatusOK, status: sdkgo.RequestStatusRunning})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	start := time.Now()
	err := WaitForDeleteRequest(ctx, newTestClient(s.URL), acceptedResponse(s.URL+testRequestStatusPath), true)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Less(t, time.Since(start), 5*time.Second)
}

func TestWaitForDeleteRequestNoShortWait(t *testing.T) {
	for name, tc := range map[string]struct {
		wait   time.Duration
		stored bool
	}{
		"wait 0":     {wait: 0, stored: true},
		"not stored": {wait: 10 * time.Millisecond, stored: false},
	} {
		t.Run(name, func(t *testing.T) {
			setDeleteRequestWait(t, tc.wait)
			s := newRequestStatusServer(t, statusResponse{code: http.StatusOK, status: sdkgo.RequestStatusRunning})

			// Like WaitForRequest: the wait lasts until the deadline of ctx.
			ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
			defer cancel()
			err := WaitForDeleteRequest(ctx, newTestClient(s.URL), acceptedResponse(s.URL+testRequestStatusPath), tc.stored)
			require.ErrorIs(t, err, context.DeadlineExceeded)
		})
	}
}

func TestWaitForDeleteRequestResult(t *testing.T) {
	for _, wait := range []time.Duration{0, time.Minute} {
		t.Run("wait "+wait.String(), func(t *testing.T) {
			setDeleteRequestWait(t, wait)

			done := newRequestStatusServer(t,
				statusResponse{code: http.StatusOK, status: sdkgo.RequestStatusRunning},
				statusResponse{code: http.StatusOK, status: sdkgo.RequestStatusDone},
			)
			require.NoError(t, WaitForDeleteRequest(context.Background(), newTestClient(done.URL), acceptedResponse(done.URL+testRequestStatusPath), true))

			failed := newRequestStatusServer(t,
				statusResponse{code: http.StatusOK, status: sdkgo.RequestStatusFailed, message: simulatedFailure},
			)
			err := WaitForDeleteRequest(context.Background(), newTestClient(failed.URL), acceptedResponse(failed.URL+testRequestStatusPath), true)
			require.ErrorContains(t, err, "request "+testRequestID+" failed: "+simulatedFailure)
		})
	}
}

// deleteFailedObject returns object metadata with the given failure annotations.
func deleteFailedObject(uid, failures, failedAt string) *metav1.ObjectMeta {
	return &metav1.ObjectMeta{UID: types.UID(uid), Annotations: map[string]string{
		DeleteFailuresAnnotationKey: failures,
		DeleteFailedAtAnnotationKey: failedAt,
	}}
}

func TestDeleteHold(t *testing.T) {
	for n, want := range map[int]time.Duration{
		1: 30 * time.Second, 2: time.Minute, 3: 2 * time.Minute, 4: 4 * time.Minute,
		5: 5 * time.Minute, 6: 5 * time.Minute, 100: 5 * time.Minute,
	} {
		t.Run(strconv.Itoa(n), func(t *testing.T) {
			seen := map[time.Duration]bool{}
			for i := range 20 {
				uid := types.UID(fmt.Sprintf("uid-%d", i))
				got := deleteHold(uid, n)
				assert.GreaterOrEqual(t, got, want*9/10)
				assert.LessOrEqual(t, got, want*11/10)
				assert.Equal(t, got, deleteHold(uid, n), "same uid and n give the same hold")
				seen[got] = true
			}
			assert.Greater(t, len(seen), 1, "holds differ between objects")
		})
	}
}

func TestCheckDeleteHold(t *testing.T) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	failedAt := now.Format(time.RFC3339)
	for _, tc := range []struct {
		name    string
		obj     *metav1.ObjectMeta
		at      time.Time
		wantErr string
	}{
		{name: "NoAnnotations", obj: &metav1.ObjectMeta{}, at: now},
		{name: "GarbageCount", obj: deleteFailedObject("uid-1", "x", failedAt), at: now},
		{name: "ZeroCount", obj: deleteFailedObject("uid-1", "0", failedAt), at: now},
		{name: "GarbageTime", obj: deleteFailedObject("uid-1", "1", "yesterday"), at: now},
		{name: "Active", obj: deleteFailedObject("uid-1", "2", failedAt), at: now.Add(50 * time.Second),
			wantErr: "2 DELETE request(s) failed, last at 2026-01-02T03:04:05Z; next DELETE not before 2026-01-02T03:0"},
		{name: "Expired", obj: deleteFailedObject("uid-1", "2", failedAt), at: now.Add(70 * time.Second)},
		{name: "ExpiredAtMax", obj: deleteFailedObject("uid-1", "9", failedAt), at: now.Add(6 * time.Minute)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := CheckDeleteHold(tc.obj, tc.at)
			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tc.wantErr)
		})
	}
}

func TestIsDeleteFailure(t *testing.T) {
	failed := fmt.Errorf("wrapped: %w", &RequestFailedError{ID: testRequestID, Message: simulatedFailure})
	for _, tc := range []struct {
		name string
		code int
		err  error
		want bool
	}{
		{name: "RequestFailed", err: failed, want: true},
		{name: "UnprocessableEntity", code: http.StatusUnprocessableEntity, want: true},
		{name: "Conflict", code: http.StatusConflict, want: true},
		{name: "BadRequest", code: http.StatusBadRequest, want: true},
		{name: "NotFound", code: http.StatusNotFound},
		{name: "RequestTimeout", code: http.StatusRequestTimeout},
		{name: "TooManyRequests", code: http.StatusTooManyRequests},
		{name: "InternalServerError", code: http.StatusInternalServerError},
		{name: "ServiceUnavailable", code: http.StatusServiceUnavailable},
		{name: "ContextDeadline", err: context.DeadlineExceeded},
		{name: "NetworkError", err: syscall.ECONNRESET},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var apiResponse *sdkgo.APIResponse
			if tc.code != 0 {
				apiResponse = &sdkgo.APIResponse{Response: &http.Response{StatusCode: tc.code}}
			}
			err := tc.err
			if err == nil {
				err = errors.New("simulated error")
			}
			assert.Equal(t, tc.want, isDeleteFailure(apiResponse, err))
		})
	}
}

func TestHandleDeleteError(t *testing.T) {
	ctx := context.Background()
	key := client.ObjectKey{Namespace: "default", Name: "cr"}
	failed := &RequestFailedError{ID: testRequestID, Message: simulatedFailure}

	t.Run("NotAFailure", func(t *testing.T) {
		cr := &corev1.ConfigMap{}
		serverErr := errors.New("simulated error")
		require.NoError(t, HandleDeleteError(ctx, nil, cr, nil, nil, logging.NewNopLogger()))
		resp := &sdkgo.APIResponse{Response: &http.Response{StatusCode: http.StatusInternalServerError}}
		assert.Same(t, serverErr, HandleDeleteError(ctx, nil, cr, resp, serverErr, logging.NewNopLogger()))
		assert.Empty(t, cr.GetAnnotations())
	})

	t.Run("Recorded", func(t *testing.T) {
		kube := fake.NewClientBuilder().
			WithObjects(&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Namespace: key.Namespace, Name: key.Name, Annotations: map[string]string{
				DELETERequestIDAnnotationKey: testRequestID,
				DeleteFailuresAnnotationKey:  "2",
			}}}).
			Build()
		cr := &corev1.ConfigMap{}
		require.NoError(t, kube.Get(ctx, key, cr))

		before := time.Now().UTC().Truncate(time.Second)
		err := HandleDeleteError(ctx, kube, cr, nil, failed, logging.NewNopLogger())
		require.ErrorIs(t, err, failed)
		assert.Contains(t, err.Error(), "request "+testRequestID+" failed: "+simulatedFailure+"; next DELETE not before ")

		stored := &corev1.ConfigMap{}
		require.NoError(t, kube.Get(ctx, key, stored))
		assert.Equal(t, "3", stored.GetAnnotations()[DeleteFailuresAnnotationKey])
		assert.Contains(t, stored.GetAnnotations(), DELETERequestIDAnnotationKey)
		assert.Empty(t, stored.GetAnnotations()[DELETERequestIDAnnotationKey])
		at, perr := time.Parse(time.RFC3339, stored.GetAnnotations()[DeleteFailedAtAnnotationKey])
		require.NoError(t, perr)
		assert.False(t, at.Before(before))
		assert.False(t, at.After(time.Now()))
		require.Error(t, CheckDeleteHold(stored, time.Now()))
	})

	t.Run("FirstFailureWithoutRequestID", func(t *testing.T) {
		cr := &corev1.ConfigMap{}
		resp := &sdkgo.APIResponse{Response: &http.Response{StatusCode: http.StatusUnprocessableEntity}}
		err := HandleDeleteError(ctx, nil, cr, resp, errors.New("simulated 422"), logging.NewNopLogger())
		require.ErrorContains(t, err, "simulated 422; next DELETE not before ")
		assert.Equal(t, "1", cr.GetAnnotations()[DeleteFailuresAnnotationKey])
		assert.NotContains(t, cr.GetAnnotations(), DELETERequestIDAnnotationKey)
	})

	t.Run("UpdateFails", func(t *testing.T) {
		// The object is not in the fake API server, so the update fails.
		kube := fake.NewClientBuilder().Build()
		cr := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Namespace: key.Namespace, Name: key.Name}}
		err := HandleDeleteError(ctx, kube, cr, nil, failed, logging.NewNopLogger())
		require.ErrorIs(t, err, failed)
		assert.Contains(t, err.Error(), "; next DELETE not before ")
	})
}
