package compute

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	sdkgo "github.com/ionos-cloud/sdk-go/v6"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	s := newRequestStatusServer(t,
		statusResponse{code: http.StatusOK, status: sdkgo.RequestStatusQueued},
		statusResponse{code: http.StatusOK, status: sdkgo.RequestStatusRunning},
		statusResponse{code: http.StatusOK},
		statusResponse{code: http.StatusOK, status: sdkgo.RequestStatusRunning},
		statusResponse{code: http.StatusOK, status: sdkgo.RequestStatusDone},
	)

	err := WaitForRequest(context.Background(), newTestClient(s.URL), acceptedResponse(s.URL+testRequestStatusPath))
	require.NoError(t, err)

	calls := s.callTimes()
	require.Len(t, calls, 5)
	for _, p := range s.callPaths() {
		assert.Equal(t, testRequestStatusPath, p)
	}
	// The wait between polls doubles: 10ms, 20ms, 40ms, 80ms.
	want := testPollInterval
	for i := 1; i < len(calls); i++ {
		assert.GreaterOrEqual(t, calls[i].Sub(calls[i-1]), want, "wait before poll %d", i+1)
		want *= 2
	}
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
	got := make([]time.Duration, 0, 8)
	interval := time.Second
	for range 8 {
		got = append(got, interval)
		interval = nextPollInterval(interval)
	}
	want := []time.Duration{
		1 * time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second,
		16 * time.Second, 30 * time.Second, 30 * time.Second, 30 * time.Second,
	}
	assert.Equal(t, want, got)

	// A configured interval above the cap is kept.
	assert.Equal(t, time.Minute, nextPollInterval(time.Minute))
}
