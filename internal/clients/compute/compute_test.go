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
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	testRequestID         = "req-1"
	testRequestStatusPath = "/cloudapi/v6/requests/" + testRequestID + "/status"
	testPollInterval      = 10 * time.Millisecond
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
		statusResponse{code: http.StatusOK, status: sdkgo.RequestStatusFailed, message: "simulated failure"},
	)

	err := WaitForRequest(context.Background(), newTestClient(s.URL), acceptedResponse(s.URL+testRequestStatusPath))
	require.ErrorContains(t, err, "request "+testRequestID+" failed: simulated failure")
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
		wantErr bool
	}{
		{name: "Queued", resp: statusResponse{code: http.StatusOK, status: sdkgo.RequestStatusQueued}, pending: true},
		{name: "Running", resp: statusResponse{code: http.StatusOK, status: sdkgo.RequestStatusRunning}, pending: true},
		{name: "NoStatus", resp: statusResponse{code: http.StatusOK}, pending: true},
		{name: "Done", resp: statusResponse{code: http.StatusOK, status: sdkgo.RequestStatusDone}},
		{name: "Failed", resp: statusResponse{code: http.StatusOK, status: sdkgo.RequestStatusFailed, message: "simulated failure"}},
		{name: "NotFound", resp: statusResponse{code: http.StatusNotFound}},
		{name: "TooManyRequests", resp: statusResponse{code: http.StatusTooManyRequests}, pending: true},
		{name: "ServiceUnavailable", resp: statusResponse{code: http.StatusServiceUnavailable}, pending: true},
		{name: "ConnectionClosed", resp: statusResponse{}, pending: true},
		{name: "Unauthorized", resp: statusResponse{code: http.StatusUnauthorized}, wantErr: true},
		{name: "InternalServerError", resp: statusResponse{code: http.StatusInternalServerError}, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := newRequestStatusServer(t, tc.resp)

			pending, err := IsDeleteRequestPending(context.Background(), newTestClient(s.URL), deleteRequestObject(testRequestID))
			if tc.wantErr {
				require.ErrorContains(t, err, "failed to retrieve delete request ("+testRequestID+") status")
			} else {
				require.NoError(t, err)
			}
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
				statusResponse{code: http.StatusOK, status: sdkgo.RequestStatusFailed, message: "simulated failure"},
			)
			err := WaitForDeleteRequest(context.Background(), newTestClient(failed.URL), acceptedResponse(failed.URL+testRequestStatusPath), true)
			require.ErrorContains(t, err, "request "+testRequestID+" failed: simulated failure")
		})
	}
}
