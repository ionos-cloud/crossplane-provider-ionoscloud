package clients

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	ionos "github.com/ionos-cloud/sdk-go/v6"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	throttleTestDC      = "dc-1"
	throttleTestOtherDC = "dc-2"
	throttleTestDCPath  = "/cloudapi/v6/datacenters/" + throttleTestDC
	throttleTestWait    = 2 * time.Minute
	throttleTestMaxWait = 10 * time.Minute
)

var throttleTestOptions = ComputeThrottleOptions{Wait: throttleTestWait, MaxWait: throttleTestMaxWait}

// resetComputeThrottle drops all blocks and restores the real clock and the default options.
func resetComputeThrottle() {
	throttleMu.Lock()
	defer throttleMu.Unlock()
	clear(throttleDatacenterBlocks)
	clear(throttleRequestBlocks)
	throttleNow = time.Now
	computeThrottle = ComputeThrottleOptions{}
}

// freshComputeThrottle isolates a test and freezes the throttle clock. The returned function
// moves the clock forward.
func freshComputeThrottle(t *testing.T) func(time.Duration) {
	t.Helper()
	resetComputeThrottle()
	t.Cleanup(resetComputeThrottle)

	now := time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC)
	throttleNow = func() time.Time { return now }
	return func(d time.Duration) { now = now.Add(d) }
}

func throttleBlockCount() int {
	throttleMu.Lock()
	defer throttleMu.Unlock()
	return len(throttleDatacenterBlocks) + len(throttleRequestBlocks)
}

// throttleTestAPI is a fake Cloud API answering every request with the configured status and
// headers, and recording the requests it received.
type throttleTestAPI struct {
	mu     sync.Mutex
	status int
	header http.Header
	hits   []string
}

func newThrottleTestAPI(t *testing.T) (*throttleTestAPI, *httptest.Server) {
	t.Helper()
	api := &throttleTestAPI{status: http.StatusOK}
	srv := httptest.NewServer(api)
	t.Cleanup(srv.Close)
	return api, srv
}

func (a *throttleTestAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.hits = append(a.hits, r.Method+" "+r.URL.Path)
	for k, v := range a.header {
		w.Header()[k] = v
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(a.status)
	_, _ = io.WriteString(w, `{}`)
}

func (a *throttleTestAPI) respond(status int, header http.Header) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.status = status
	a.header = header
}

func (a *throttleTestAPI) hitCount() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return len(a.hits)
}

type throttleTestResult struct {
	status int
	header http.Header
	body   string
}

func (r throttleTestResult) throttled() bool {
	return r.header.Get(throttledHeader) == "true"
}

func throttleTestDo(t *testing.T, hc *http.Client, method, url string) throttleTestResult {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), method, url, nil)
	require.NoError(t, err)
	resp, err := hc.Do(req)
	require.NoError(t, err)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	return throttleTestResult{status: resp.StatusCode, header: resp.Header, body: string(body)}
}

func TestComputeThrottle_429RetryAfterSecondsBlocksDatacenterWrites(t *testing.T) {
	advance := freshComputeThrottle(t)
	api, srv := newThrottleTestAPI(t)
	hc := withComputeThrottle(&http.Client{}, throttleTestOptions)

	api.respond(http.StatusTooManyRequests, http.Header{retryAfterHeader: []string{"30"}})
	got := throttleTestDo(t, hc, http.MethodDelete, srv.URL+throttleTestDCPath+"/servers/s1")
	require.Equal(t, http.StatusTooManyRequests, got.status)
	require.False(t, got.throttled(), "the first 429 comes from the API")
	api.respond(http.StatusOK, nil)

	hits := api.hitCount()
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		got = throttleTestDo(t, hc, method, srv.URL+throttleTestDCPath+"/lans/1")
		assert.Equal(t, http.StatusTooManyRequests, got.status, method)
		assert.True(t, got.throttled(), method)
		assert.Equal(t, "30", got.header.Get(retryAfterHeader), method)
		assert.Equal(t, "application/json", got.header.Get("Content-Type"), method)
		assert.Contains(t, got.body, "throttled locally by crossplane-provider-ionoscloud until 2026-01-01T12:00:30Z after Cloud API 429 for datacenter dc-1", method)
	}
	assert.Equal(t, hits, api.hitCount(), "blocked writes must not reach the API")

	assert.Equal(t, http.StatusOK, throttleTestDo(t, hc, http.MethodGet, srv.URL+throttleTestDCPath+"/servers").status, "reads pass")
	assert.Equal(t, http.StatusOK, throttleTestDo(t, hc, http.MethodPost, srv.URL+"/cloudapi/v6/datacenters/"+throttleTestOtherDC+"/servers").status, "other datacenters pass")
	assert.Equal(t, http.StatusOK, throttleTestDo(t, hc, http.MethodPost, srv.URL+"/cloudapi/v6/datacenters").status, "requests without a datacenter ID pass")
	assert.Equal(t, hits+3, api.hitCount())

	advance(29*time.Second + 500*time.Millisecond)
	got = throttleTestDo(t, hc, http.MethodDelete, srv.URL+throttleTestDCPath+"/lans/1")
	assert.True(t, got.throttled(), "still blocked before Retry-After passed")
	assert.Equal(t, "1", got.header.Get(retryAfterHeader), "remaining time is rounded up")

	advance(500 * time.Millisecond)
	got = throttleTestDo(t, hc, http.MethodDelete, srv.URL+throttleTestDCPath+"/lans/1")
	assert.Equal(t, http.StatusOK, got.status, "unblocked once Retry-After passed")
	assert.False(t, got.throttled())
	assert.Equal(t, 0, throttleBlockCount(), "expired blocks are dropped")
}

func TestComputeThrottle_429OnReadWithoutRetryAfterUsesWait(t *testing.T) {
	advance := freshComputeThrottle(t)
	api, srv := newThrottleTestAPI(t)
	hc := withComputeThrottle(&http.Client{}, throttleTestOptions)

	// No /cloudapi/v6 prefix: the datacenter ID is found either way.
	api.respond(http.StatusTooManyRequests, nil)
	throttleTestDo(t, hc, http.MethodGet, srv.URL+"/datacenters/"+throttleTestDC+"/servers")
	api.respond(http.StatusOK, nil)

	advance(throttleTestWait - time.Second)
	got := throttleTestDo(t, hc, http.MethodPost, srv.URL+throttleTestDCPath+"/servers")
	assert.True(t, got.throttled())
	assert.Equal(t, "1", got.header.Get(retryAfterHeader))

	advance(time.Second)
	assert.False(t, throttleTestDo(t, hc, http.MethodPost, srv.URL+throttleTestDCPath+"/servers").throttled())
}

func TestComputeThrottle_422BlocksOnlySameMethodAndPath(t *testing.T) {
	advance := freshComputeThrottle(t)
	api, srv := newThrottleTestAPI(t)
	hc := withComputeThrottle(&http.Client{}, throttleTestOptions)
	lanURL := srv.URL + throttleTestDCPath + "/lans/lan-1"

	api.respond(http.StatusUnprocessableEntity, nil)
	throttleTestDo(t, hc, http.MethodDelete, lanURL)
	api.respond(http.StatusOK, nil)

	hits := api.hitCount()
	got := throttleTestDo(t, hc, http.MethodDelete, lanURL)
	assert.True(t, got.throttled())
	assert.Contains(t, got.body, "after Cloud API 422 for DELETE "+throttleTestDCPath+"/lans/lan-1")
	assert.Equal(t, hits, api.hitCount(), "the blocked DELETE must not reach the API")

	assert.False(t, throttleTestDo(t, hc, http.MethodPut, lanURL).throttled(), "other method")
	assert.False(t, throttleTestDo(t, hc, http.MethodDelete, srv.URL+throttleTestDCPath+"/lans/lan-2").throttled(), "other path")
	assert.False(t, throttleTestDo(t, hc, http.MethodGet, lanURL).throttled(), "read")

	advance(throttleTestWait)
	assert.False(t, throttleTestDo(t, hc, http.MethodDelete, lanURL).throttled(), "expired")
}

func TestComputeThrottle_422OnPostDoesNotBlock(t *testing.T) {
	freshComputeThrottle(t)
	api, srv := newThrottleTestAPI(t)
	hc := withComputeThrottle(&http.Client{}, throttleTestOptions)
	volumesURL := srv.URL + throttleTestDCPath + "/volumes"

	api.respond(http.StatusUnprocessableEntity, nil)
	throttleTestDo(t, hc, http.MethodPost, volumesURL)
	assert.Equal(t, 0, throttleBlockCount())

	hits := api.hitCount()
	got := throttleTestDo(t, hc, http.MethodPost, volumesURL)
	assert.Equal(t, http.StatusUnprocessableEntity, got.status, "the next POST gets the real 422")
	assert.False(t, got.throttled())
	assert.Equal(t, hits+1, api.hitCount(), "the next POST reaches the API")
}

func TestComputeThrottle_422OnReadDoesNotBlock(t *testing.T) {
	freshComputeThrottle(t)
	api, srv := newThrottleTestAPI(t)
	hc := withComputeThrottle(&http.Client{}, throttleTestOptions)

	api.respond(http.StatusUnprocessableEntity, nil)
	throttleTestDo(t, hc, http.MethodGet, srv.URL+throttleTestDCPath+"/servers")
	assert.Equal(t, 0, throttleBlockCount())
}

func TestComputeThrottle_429WithoutDatacenterDoesNotBlock(t *testing.T) {
	freshComputeThrottle(t)
	api, srv := newThrottleTestAPI(t)
	hc := withComputeThrottle(&http.Client{}, throttleTestOptions)

	api.respond(http.StatusTooManyRequests, nil)
	throttleTestDo(t, hc, http.MethodPost, srv.URL+"/cloudapi/v6/datacenters")
	throttleTestDo(t, hc, http.MethodGet, srv.URL+"/cloudapi/v6/requests/r1/status")
	assert.Equal(t, 0, throttleBlockCount())

	hits := api.hitCount()
	assert.False(t, throttleTestDo(t, hc, http.MethodPost, srv.URL+"/cloudapi/v6/datacenters").throttled())
	assert.False(t, throttleTestDo(t, hc, http.MethodPost, srv.URL+throttleTestDCPath+"/servers").throttled())
	assert.Equal(t, hits+2, api.hitCount())
}

// readCloserSpy records whether the transport closed the request body.
type readCloserSpy struct {
	io.Reader
	closed bool
}

func (r *readCloserSpy) Close() error {
	r.closed = true
	return nil
}

func TestComputeThrottle_BlockedRequestBodyIsClosed(t *testing.T) {
	freshComputeThrottle(t)
	addThrottleBlock(throttleDatacenterBlocks, throttleTestDC, throttleNow().Add(time.Minute), "test")

	rt := &computeThrottleRoundTripper{next: http.DefaultTransport, opts: throttleTestOptions, nextShared: true}
	body := &readCloserSpy{Reader: strings.NewReader(`{}`)}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "http://127.0.0.1:1"+throttleTestDCPath+"/servers", body)
	require.NoError(t, err)

	resp, err := rt.RoundTrip(req)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	assert.Equal(t, http.StatusTooManyRequests, resp.StatusCode)
	assert.Same(t, req, resp.Request)
	assert.True(t, body.closed)
}

func Test_throttleWait(t *testing.T) {
	now := time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name       string
		retryAfter string
		want       time.Duration
	}{
		{name: "seconds", retryAfter: "30", want: 30 * time.Second},
		{name: "seconds with spaces", retryAfter: " 30 ", want: 30 * time.Second},
		{name: "http date", retryAfter: now.Add(90 * time.Second).Format(http.TimeFormat), want: 90 * time.Second},
		{name: "missing", retryAfter: "", want: throttleTestWait},
		{name: "invalid", retryAfter: "soon", want: throttleTestWait},
		{name: "negative", retryAfter: "-5", want: throttleTestWait},
		{name: "zero", retryAfter: "0", want: throttleTestWait},
		{name: "date in the past", retryAfter: now.Add(-time.Minute).Format(http.TimeFormat), want: throttleTestWait},
		{name: "above max", retryAfter: "3600", want: throttleTestMaxWait},
		{name: "too large for a duration", retryAfter: "99999999999999999", want: throttleTestWait},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, throttleWait(tt.retryAfter, now, throttleTestOptions))
		})
	}
}

func Test_datacenterIDFromPath(t *testing.T) {
	tests := map[string]string{
		"/cloudapi/v6/datacenters/dc-1/servers/s1": "dc-1",
		"/v6/datacenters/dc-1/lans":                "dc-1",
		"/datacenters/dc-1":                        "dc-1",
		"/cloudapi/v6/datacenters":                 "",
		"/cloudapi/v6/datacenters/":                "",
		"/cloudapi/v6/requests/r1/status":          "",
		"/cloudapi/v6/um/groups":                   "",
	}
	for path, want := range tests {
		assert.Equal(t, want, datacenterIDFromPath(path), path)
	}
}

func TestValidateComputeThrottleOptions(t *testing.T) {
	tests := []struct {
		name    string
		opts    ComputeThrottleOptions
		wantErr bool
	}{
		{name: "disabled", opts: ComputeThrottleOptions{MaxWait: throttleTestMaxWait}},
		{name: "disabled with zero max", opts: ComputeThrottleOptions{}},
		{name: "enabled", opts: throttleTestOptions},
		{name: "max equals wait", opts: ComputeThrottleOptions{Wait: time.Minute, MaxWait: time.Minute}},
		{name: "negative wait", opts: ComputeThrottleOptions{Wait: -time.Second, MaxWait: throttleTestMaxWait}, wantErr: true},
		{name: "negative max", opts: ComputeThrottleOptions{MaxWait: -time.Second}, wantErr: true},
		{name: "max below wait", opts: ComputeThrottleOptions{Wait: time.Hour, MaxWait: time.Minute}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateComputeThrottleOptions(tt.opts)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestNewIonosClient_ComputeThrottle(t *testing.T) {
	creds := []byte(`{"user": "username","password": "cGFzc3dvcmQ="}`)

	t.Run("disabled by default", func(t *testing.T) {
		freshComputeThrottle(t)
		freshIonosClientCache(t)
		svc, err := NewIonosClients(creds)
		require.NoError(t, err)
		assert.Same(t, http.DefaultClient, svc.ComputeClient.GetConfig().HTTPClient)
	})

	t.Run("enabled", func(t *testing.T) {
		freshComputeThrottle(t)
		freshIonosClientCache(t)
		ConfigureComputeThrottle(throttleTestOptions)
		defaultTransport := http.DefaultClient.Transport

		svc, err := NewIonosClients(creds)
		require.NoError(t, err)
		hc := svc.ComputeClient.GetConfig().HTTPClient
		require.NotSame(t, http.DefaultClient, hc)
		rt, ok := hc.Transport.(*computeThrottleRoundTripper)
		require.True(t, ok, "expected *computeThrottleRoundTripper, got %T", hc.Transport)
		assert.Equal(t, throttleTestOptions, rt.opts)
		assert.True(t, rt.nextShared)
		assert.Equal(t, defaultTransport, http.DefaultClient.Transport, "http.DefaultClient must not be changed")
		assert.Same(t, http.DefaultClient, svc.DBaaSPostgresClient.GetConfig().HTTPClient, "only the compute client is wrapped")
	})
}

// closeIdleSpy is a RoundTripper recording CloseIdleConnections calls.
type closeIdleSpy struct {
	closed int
}

func (s *closeIdleSpy) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("not used")
}

func (s *closeIdleSpy) CloseIdleConnections() {
	s.closed++
}

func Test_withComputeThrottle(t *testing.T) {
	t.Run("keeps the client fields and forwards CloseIdleConnections", func(t *testing.T) {
		spy := &closeIdleSpy{}
		base := &http.Client{Transport: spy, Timeout: 5 * time.Second}

		hc := withComputeThrottle(base, throttleTestOptions)
		require.NotSame(t, base, hc)
		assert.Same(t, spy, base.Transport, "the original client is not changed")
		assert.Equal(t, 5*time.Second, hc.Timeout)
		rt, ok := hc.Transport.(*computeThrottleRoundTripper)
		require.True(t, ok)
		assert.Same(t, spy, rt.next)

		hc.CloseIdleConnections()
		assert.Equal(t, 1, spy.closed)
	})

	t.Run("does not forward CloseIdleConnections to a shared transport", func(t *testing.T) {
		hc := withComputeThrottle(http.DefaultClient, throttleTestOptions)
		require.NotSame(t, http.DefaultClient, hc)
		rt, ok := hc.Transport.(*computeThrottleRoundTripper)
		require.True(t, ok)
		assert.True(t, rt.nextShared)
	})
}

// TestComputeThrottle_SDKDecodesSyntheticResponse checks that sdk-go v6 turns the synthetic
// response into a normal 429 API error that carries the message.
func TestComputeThrottle_SDKDecodesSyntheticResponse(t *testing.T) {
	freshComputeThrottle(t)
	api, srv := newThrottleTestAPI(t)

	cfg := ionos.NewConfiguration("throttle-user", "throttle-secret", "", srv.URL)
	cfg.MaxRetries = computeMaxRetries
	cfg.HTTPClient = withComputeThrottle(&http.Client{}, throttleTestOptions)
	client := ionos.NewAPIClient(cfg)

	api.respond(http.StatusTooManyRequests, http.Header{retryAfterHeader: []string{"30"}})
	_, err := client.DataCentersApi.DatacentersDelete(context.Background(), throttleTestDC).Execute()
	require.Error(t, err)
	require.Equal(t, 1, api.hitCount())
	api.respond(http.StatusOK, nil)

	apiResponse, err := client.DataCentersApi.DatacentersDelete(context.Background(), throttleTestDC).Execute()
	require.Error(t, err)
	assert.Equal(t, 1, api.hitCount(), "the blocked delete must not reach the API")
	require.NotNil(t, apiResponse)
	require.NotNil(t, apiResponse.Response)
	assert.Equal(t, http.StatusTooManyRequests, apiResponse.StatusCode)
	require.ErrorContains(t, err, "429 Too Many Requests")
	require.ErrorContains(t, err, "throttled locally by crossplane-provider-ionoscloud")

	var apiErr ionos.GenericOpenAPIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusTooManyRequests, apiErr.StatusCode())
	model, ok := apiErr.Model().(ionos.Error)
	require.True(t, ok, "expected ionos.Error model, got %T", apiErr.Model())
	require.NotNil(t, model.Messages)
	require.Len(t, *model.Messages, 1)
	msg := (*model.Messages)[0]
	require.NotNil(t, msg.ErrorCode)
	require.NotNil(t, msg.Message)
	assert.Equal(t, throttledErrorCode, *msg.ErrorCode)
	assert.Contains(t, *msg.Message, "after Cloud API 429 for datacenter "+throttleTestDC)
	require.NotNil(t, model.HttpStatus)
	assert.Equal(t, int32(http.StatusTooManyRequests), *model.HttpStatus)
}
