package clients

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/crossplane/crossplane-runtime/pkg/logging"
	ionos "github.com/ionos-cloud/sdk-go/v6"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute"
)

const (
	faultTestDC      = "dc-fault"
	faultTestOtherDC = "dc-other"
	faultTestHost    = "https://api.fault.test"
	faultTestAPIURL  = faultTestHost + "/cloudapi/v6"
	faultTestDCURL   = faultTestAPIURL + "/datacenters/" + faultTestDC
	faultTestUntil   = "until=2026-01-01T12:30:00Z"
	faultTestIDShape = `^[0-9a-f]{8}-fa17-4000-8000-[0-9a-f]{12}$`
)

var faultTestNow = time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC)

// faultTestLog records Info calls as maps of their key/value pairs plus "msg".
type faultTestLog struct{ lines []map[string]any }

func (l *faultTestLog) Info(msg string, keysAndValues ...any) {
	line := map[string]any{"msg": msg}
	for i := 0; i+1 < len(keysAndValues); i += 2 {
		line[fmt.Sprint(keysAndValues[i])] = keysAndValues[i+1]
	}
	l.lines = append(l.lines, line)
}
func (l *faultTestLog) Debug(_ string, _ ...any)           {}
func (l *faultTestLog) WithValues(_ ...any) logging.Logger { return l }

// answers returns the answer of every "cloudapi fault injected" line.
func (l *faultTestLog) answers() []string {
	out := make([]string, 0, len(l.lines))
	for _, line := range l.lines {
		if line["msg"] == "cloudapi fault injected" {
			out = append(out, fmt.Sprint(line["answer"]))
		}
	}
	return out
}

// faultTestNext stands in for the Cloud API behind the injector: it answers 200 {} and counts the
// calls. It fails the test for a fake request ID, and for a write to blockDC while that is set.
type faultTestNext struct {
	t       *testing.T
	blockDC string
	calls   int
}

func (n *faultTestNext) RoundTrip(req *http.Request) (*http.Response, error) {
	n.calls++
	if req.Body != nil {
		_ = req.Body.Close()
	}
	if strings.Contains(strings.ToLower(req.URL.Path), fakeRequestIDMiddle) {
		n.t.Errorf("fake request id forwarded: %s %s", req.Method, req.URL.Path)
	}
	if n.blockDC != "" && faultIsWrite(req.Method) && strings.Contains(req.URL.Path, "/datacenters/"+n.blockDC) {
		n.t.Errorf("write to the faulted datacenter forwarded: %s %s", req.Method, req.URL.Path)
	}
	return &http.Response{
		Status: "200 OK", StatusCode: http.StatusOK, Proto: "HTTP/1.1", ProtoMajor: 1, ProtoMinor: 1,
		Header:  http.Header{"Content-Type": []string{"application/json"}},
		Body:    io.NopCloser(strings.NewReader(`{}`)),
		Request: req,
	}, nil
}

type faultTest struct {
	t    *testing.T
	hc   *http.Client
	next *faultTestNext
	log  *faultTestLog
	now  time.Time
}

// newFaultTest builds an injector for values, with its clock at faultTestNow.
func newFaultTest(t *testing.T, values ...string) *faultTest {
	t.Helper()
	ft := &faultTest{t: t, next: &faultTestNext{t: t, blockDC: faultTestDC}, log: &faultTestLog{}, now: faultTestNow}
	faults, err := ParseCloudAPIFaults(values, faultTestNow, ft.log)
	require.NoError(t, err)
	ft.hc = withCloudAPIFaults(&http.Client{Transport: ft.next}, faults, ft.log)
	fi, ok := ft.hc.Transport.(*faultInjector)
	require.True(t, ok, "expected *faultInjector, got %T", ft.hc.Transport)
	fi.now = func() time.Time { return ft.now }
	return ft
}

func (ft *faultTest) advance(d time.Duration) { ft.now = ft.now.Add(d) }

type faultTestResult struct {
	status int
	header http.Header
	body   string
	err    error
}

func (ft *faultTest) do(method, url string, body io.Reader) faultTestResult {
	ft.t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), method, url, body)
	require.NoError(ft.t, err)
	resp, err := ft.hc.Do(req)
	if err != nil {
		return faultTestResult{err: err}
	}
	data, err := io.ReadAll(resp.Body)
	require.NoError(ft.t, err)
	require.NoError(ft.t, resp.Body.Close())
	return faultTestResult{status: resp.StatusCode, header: resp.Header, body: string(data)}
}

// faultTestRequestID returns the request ID of a fake 202, read like the controllers do.
func faultTestRequestID(t *testing.T, res faultTestResult) string {
	t.Helper()
	require.Equal(t, http.StatusAccepted, res.status)
	id, err := compute.ExtractRequestID(&ionos.APIResponse{Response: &http.Response{Header: res.header}})
	require.NoError(t, err)
	return id
}

func faultTestHash(dc string) string {
	sum := sha256.Sum256([]byte(dc))
	return hex.EncodeToString(sum[:])[:8]
}

func freshCloudAPIFaults(t *testing.T) {
	t.Helper()
	reset := func() { cloudAPIFaultConfig, cloudAPIFaultLog = nil, logging.NewNopLogger() }
	reset()
	t.Cleanup(reset)
}

func TestFaultInjector_DeleteStuck(t *testing.T) {
	ft := newFaultTest(t, "dc="+faultTestDC+",mode=delete-stuck,"+faultTestUntil)

	id := faultTestRequestID(t, ft.do(http.MethodDelete, faultTestDCURL+"/servers/s1", nil))
	assert.Regexp(t, faultTestIDShape, id)
	assert.Equal(t, faultTestHash(faultTestDC), id[:8])
	assert.NotEqual(t, id, faultTestRequestID(t, ft.do(http.MethodDelete, faultTestDCURL+"/servers/s1", nil)), "random part")

	res := ft.do(http.MethodDelete, faultTestDCURL+"/lans/1", nil)
	assert.Equal(t, faultTestAPIURL+"/requests/"+faultTestRequestID(t, res)+"/status", res.header.Get("Location"))
	assert.Empty(t, res.body)

	for _, path := range []string{"/requests/" + id + "/status", "/requests/" + id} {
		res = ft.do(http.MethodGet, faultTestAPIURL+path, nil)
		assert.Equal(t, http.StatusOK, res.status, path)
		assert.JSONEq(t, `{"metadata":{"status":"RUNNING"}}`, res.body, path)
	}
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch} {
		require.ErrorIs(t, ft.do(method, faultTestDCURL+"/servers/s1", nil).err, syscall.ECONNRESET, method)
	}

	assert.Equal(t, 0, ft.next.calls)
	answers := ft.log.answers()
	require.Len(t, answers, 8, "one line per intercepted call")
	assert.Equal(t, "202 fake request "+id, answers[0])
	assert.Equal(t, []string{"200 RUNNING", "200 RUNNING", "ECONNRESET", "ECONNRESET", "ECONNRESET"}, answers[3:])
}

func TestFaultInjector_StatusModes(t *testing.T) {
	for _, mode := range []string{faultModeStatus503, faultModeStatusConn} {
		t.Run(mode, func(t *testing.T) {
			ft := newFaultTest(t, "dc="+faultTestDC+",mode="+mode+","+faultTestUntil)
			id := faultTestRequestID(t, ft.do(http.MethodDelete, faultTestDCURL+"/volumes/v1", nil))
			res := ft.do(http.MethodGet, faultTestAPIURL+"/requests/"+id+"/status", nil)
			if mode == faultModeStatus503 {
				require.NoError(t, res.err)
				assert.Equal(t, http.StatusServiceUnavailable, res.status)
				assert.Contains(t, res.body, faultErrorCode)
			} else {
				require.ErrorIs(t, res.err, syscall.ECONNRESET)
			}
			require.ErrorIs(t, ft.do(http.MethodPost, faultTestDCURL+"/volumes", nil).err, syscall.ECONNRESET)
			assert.Equal(t, 0, ft.next.calls)
			assert.Len(t, ft.log.answers(), 3)
		})
	}
}

func TestFaultInjector_Write429(t *testing.T) {
	ft := newFaultTest(t, "dc="+faultTestDC+",mode=write-429,"+faultTestUntil+",retryAfter=30")
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		res := ft.do(method, faultTestDCURL+"/nics/n1", nil)
		assert.Equal(t, http.StatusTooManyRequests, res.status, method)
		assert.Equal(t, "30", res.header.Get("Retry-After"), method)
		assert.Equal(t, "application/json", res.header.Get("Content-Type"), method)
		assert.Contains(t, res.body, faultErrorCode, method)
	}
	assert.Equal(t, "429 retry-after=30", ft.log.answers()[0])

	ft = newFaultTest(t, "dc="+faultTestDC+",mode=write-429,"+faultTestUntil)
	res := ft.do(http.MethodDelete, faultTestDCURL+"/nics/n1", nil)
	assert.Equal(t, http.StatusTooManyRequests, res.status)
	assert.Empty(t, res.header.Get("Retry-After"))
	assert.Equal(t, []string{"429"}, ft.log.answers())
	assert.Equal(t, 0, ft.next.calls)
}

func TestFaultInjector_Delete422(t *testing.T) {
	ft := newFaultTest(t, "dc="+faultTestDC+",mode=delete-422,"+faultTestUntil)
	res := ft.do(http.MethodDelete, faultTestDCURL+"/lans/1", nil)
	assert.Equal(t, http.StatusUnprocessableEntity, res.status)
	assert.Contains(t, res.body, faultErrorCode)
	require.ErrorIs(t, ft.do(http.MethodPatch, faultTestDCURL+"/lans/1", nil).err, syscall.ECONNRESET)
	assert.Equal(t, []string{"422", "ECONNRESET"}, ft.log.answers())
	assert.Equal(t, 0, ft.next.calls)
}

func TestFaultInjector_PassThrough(t *testing.T) {
	ft := newFaultTest(t, "dc="+faultTestDC+",mode=delete-stuck,"+faultTestUntil)
	for _, req := range []struct{ method, url string }{
		{http.MethodGet, faultTestDCURL + "/servers/s1"},
		{http.MethodHead, faultTestDCURL + "/servers/s1"},
		{http.MethodDelete, faultTestAPIURL + "/datacenters/" + faultTestOtherDC + "/servers/s1"},
		{http.MethodPost, faultTestAPIURL + "/datacenters"},
		{http.MethodGet, faultTestAPIURL + "/requests/5d1a3d5e-29a4-4c63-9c6f-8b2e4d9a0f11/status"},
		{http.MethodDelete, faultTestHost + "/v6/datacenters/" + faultTestOtherDC},
	} {
		calls := ft.next.calls
		name := req.method + " " + req.url
		assert.Equal(t, http.StatusOK, ft.do(req.method, req.url, nil).status, name)
		assert.Equal(t, calls+1, ft.next.calls, "exactly once: "+name)
	}
	assert.Empty(t, ft.log.lines, "pass-through calls are not logged")
}

func TestFaultInjector_NoPrefixAndBodyClosed(t *testing.T) {
	ft := newFaultTest(t, "dc="+faultTestDC+",mode=delete-stuck,"+faultTestUntil)
	body := &faultTestBody{Reader: strings.NewReader(`{}`)}
	require.ErrorIs(t, ft.do(http.MethodPost, faultTestHost+"/datacenters/"+faultTestDC+"/servers", body).err, syscall.ECONNRESET)
	assert.True(t, body.closed, "POST body")

	body = &faultTestBody{Reader: strings.NewReader(`{}`)}
	res := ft.do(http.MethodDelete, faultTestHost+"/datacenters/"+faultTestDC+"/servers/s1", body)
	assert.True(t, body.closed, "DELETE body")
	assert.Equal(t, faultTestHost+"/requests/"+faultTestRequestID(t, res)+"/status", res.header.Get("Location"))
}

// faultTestBody records whether the request body was closed.
type faultTestBody struct {
	io.Reader
	closed bool
}

func (b *faultTestBody) Close() error {
	b.closed = true
	return nil
}

func TestFaultInjector_WindowEdges(t *testing.T) {
	ft := newFaultTest(t, "dc="+faultTestDC+",mode=delete-422,from=2026-01-01T12:10:00Z,until=2026-01-01T12:20:00Z")
	url := faultTestDCURL + "/lans/1"

	ft.next.blockDC = ""
	assert.Equal(t, http.StatusOK, ft.do(http.MethodDelete, url, nil).status, "before from")
	ft.next.blockDC = faultTestDC
	ft.advance(10 * time.Minute)
	assert.Equal(t, http.StatusUnprocessableEntity, ft.do(http.MethodDelete, url, nil).status, "at from")
	ft.advance(10*time.Minute - time.Nanosecond)
	assert.Equal(t, http.StatusUnprocessableEntity, ft.do(http.MethodDelete, url, nil).status, "just before until")
	ft.next.blockDC = ""
	ft.advance(time.Nanosecond)
	assert.Equal(t, http.StatusOK, ft.do(http.MethodDelete, url, nil).status, "at until")
	assert.Equal(t, 2, ft.next.calls)
}

func TestFaultInjector_FakeIDNotFound(t *testing.T) {
	ft := newFaultTest(t, "dc="+faultTestDC+",mode=write-429,"+faultTestUntil)
	for _, id := range []string{
		faultTestHash(faultTestDC) + fakeRequestIDMiddle + "0123456789ab", // mode without fake requests
		"00000000" + fakeRequestIDMiddle + "0123456789ab",                 // unknown hash
		strings.ToUpper(faultTestHash(faultTestDC) + fakeRequestIDMiddle + "0123456789ab"),
	} {
		res := ft.do(http.MethodGet, faultTestAPIURL+"/requests/"+id+"/status", nil)
		assert.Equal(t, http.StatusNotFound, res.status, id)
		assert.Contains(t, res.body, faultErrorCode, id)
	}
	assert.Equal(t, 0, ft.next.calls)
	assert.Equal(t, []string{"404 fake request id", "404 fake request id", "404 fake request id"}, ft.log.answers())
}

// TestFaultInjector_SDK runs the injector under a real sdk-go v6 client and the compute helpers.
func TestFaultInjector_SDK(t *testing.T) {
	for _, mode := range []string{faultModeDeleteStuck, faultModeStatus503, faultModeStatusConn} {
		t.Run(mode, func(t *testing.T) {
			ft := newFaultTest(t, "dc="+faultTestDC+",mode="+mode+","+faultTestUntil)
			cfg := ionos.NewConfiguration("fault-user", "fault-secret", "", faultTestHost)
			cfg.MaxRetries = computeMaxRetries
			cfg.HTTPClient = ft.hc
			client := ionos.NewAPIClient(cfg)
			ctx := context.Background()

			apiResponse, err := client.ServersApi.DatacentersServersDelete(ctx, faultTestDC, "server-1").Execute()
			require.NoError(t, err)
			require.Equal(t, http.StatusAccepted, apiResponse.StatusCode)
			id, err := compute.ExtractRequestID(apiResponse)
			require.NoError(t, err)
			assert.Regexp(t, faultTestIDShape, id)

			obj := &metav1.ObjectMeta{Annotations: map[string]string{compute.DELETERequestIDAnnotationKey: id}}
			pending, err := compute.IsDeleteRequestPending(ctx, client, obj)
			require.NoError(t, err)
			assert.True(t, pending, "RUNNING, 503 and connection reset all count as pending")
			if mode == faultModeDeleteStuck {
				status, _, err := client.RequestsApi.RequestsStatusGet(ctx, id).Execute()
				require.NoError(t, err)
				require.NotNil(t, status.Metadata)
				require.NotNil(t, status.Metadata.Status)
				assert.Equal(t, ionos.RequestStatusRunning, *status.Metadata.Status)
			}

			ft.advance(30 * time.Minute)
			_, apiResponse, err = client.RequestsApi.RequestsStatusGet(ctx, id).Execute()
			require.Error(t, err)
			assert.True(t, apiResponse.HttpNotFound(), "after the window the fake ID is answered 404 locally")
			pending, err = compute.IsDeleteRequestPending(ctx, client, obj)
			require.NoError(t, err)
			assert.False(t, pending)

			ft.next.blockDC = ""
			_, err = client.ServersApi.DatacentersServersDelete(ctx, faultTestDC, "server-1").Execute()
			require.NoError(t, err)
			assert.Equal(t, 1, ft.next.calls, "only the DELETE after the window reaches the Cloud API")
		})
	}
}

func TestParseCloudAPIFaults(t *testing.T) {
	dc := "dc=" + faultTestDC + ","
	t.Run("no values", func(t *testing.T) {
		faults, err := ParseCloudAPIFaults(nil, faultTestNow, &faultTestLog{})
		require.NoError(t, err)
		assert.Nil(t, faults)
	})
	t.Run("valid", func(t *testing.T) {
		faults, err := ParseCloudAPIFaults([]string{
			" dc = " + faultTestDC + " , mode=write-429,from=2026-01-01T13:00:00+01:00," + faultTestUntil + ",retryAfter=7",
			dc + "mode=delete-stuck,from=2026-01-01T12:30:00Z,until=2026-01-01T13:00:00Z", // adjacent, no overlap
			"dc=" + faultTestOtherDC + ",mode=status-conn," + faultTestUntil,
		}, faultTestNow, &faultTestLog{})
		require.NoError(t, err)
		require.Len(t, faults.faults, 3)
		f := faults.faults[0]
		assert.Equal(t, faultTestDC, f.dc)
		assert.Equal(t, faultModeWrite429, f.mode)
		assert.True(t, f.from.Equal(faultTestNow))
		assert.True(t, f.until.Equal(faultTestNow.Add(30*time.Minute)))
		assert.Equal(t, 7, f.retryAfter)
		assert.Equal(t, faultTestHash(faultTestDC), f.dcHash)
	})
	t.Run("until past is ignored", func(t *testing.T) {
		log := &faultTestLog{}
		faults, err := ParseCloudAPIFaults([]string{
			dc + "mode=delete-stuck,until=2026-01-01T12:00:00Z",
			dc + "mode=delete-422,until=2026-01-01T12:30:00Z,from=2025-12-31T00:00:00Z",
			dc + "mode=status-503,until=2026-01-01T11:00:00Z", // overlaps the one above, but is past
		}, faultTestNow, log)
		require.NoError(t, err)
		require.NotNil(t, faults, "the injector stays installed for fake request IDs")
		require.Len(t, faults.faults, 1)
		assert.Equal(t, faultModeDelete422, faults.faults[0].mode)
		assert.Len(t, log.lines, 2)
	})
	for name, value := range map[string]string{
		"missing until":         dc + "mode=delete-stuck",
		"missing dc":            "mode=delete-stuck," + faultTestUntil,
		"missing mode":          dc + faultTestUntil,
		"unknown mode":          dc + "mode=slow," + faultTestUntil,
		"unknown key":           dc + "mode=delete-stuck,color=red," + faultTestUntil,
		"bad until":             dc + "mode=delete-stuck,until=tomorrow",
		"bad from":              dc + "mode=delete-stuck,from=2026-01-01," + faultTestUntil,
		"retryAfter wrong mode": dc + "mode=delete-stuck,retryAfter=5," + faultTestUntil,
		"retryAfter zero":       dc + "mode=write-429,retryAfter=0," + faultTestUntil,
		"retryAfter not int":    dc + "mode=write-429,retryAfter=5s," + faultTestUntil,
		"from not before until": dc + "mode=delete-stuck,from=2026-01-01T12:30:00Z," + faultTestUntil,
		"key twice":             dc + "mode=delete-stuck," + faultTestUntil + "," + faultTestUntil,
		"not key=value":         dc + "mode=delete-stuck,oops," + faultTestUntil,
		"empty value":           dc + "mode=," + faultTestUntil,
		"dc with slash":         "dc=a/b,mode=delete-stuck," + faultTestUntil,
	} {
		t.Run(name, func(t *testing.T) {
			_, err := ParseCloudAPIFaults([]string{value}, faultTestNow, &faultTestLog{})
			require.Error(t, err)
		})
	}
	t.Run("overlapping windows", func(t *testing.T) {
		_, err := ParseCloudAPIFaults([]string{
			dc + "mode=delete-stuck," + faultTestUntil,
			dc + "mode=write-429,from=2026-01-01T12:29:59Z,until=2026-01-01T13:00:00Z",
		}, faultTestNow, &faultTestLog{})
		require.ErrorContains(t, err, "overlaps")
	})
}

func TestConfigureCloudAPIFaults_LogsArmedFaults(t *testing.T) {
	freshCloudAPIFaults(t)
	log := &faultTestLog{}
	faults, err := ParseCloudAPIFaults([]string{
		"dc=" + faultTestDC + ",mode=write-429," + faultTestUntil + ",retryAfter=9",
		"dc=" + faultTestOtherDC + ",mode=delete-stuck,from=2026-01-01T12:10:00Z," + faultTestUntil,
	}, faultTestNow, log)
	require.NoError(t, err)
	ConfigureCloudAPIFaults(faults, log)
	assert.Same(t, faults, cloudAPIFaultConfig)
	require.Len(t, log.lines, 2)
	assert.Equal(t, map[string]any{"msg": "TEST ONLY cloudapi fault armed", "dc": faultTestDC, "mode": faultModeWrite429,
		"from": "-", "until": "2026-01-01T12:30:00Z", "retryAfter": 9}, log.lines[0])
	assert.Equal(t, "2026-01-01T12:10:00Z", log.lines[1]["from"])
}
