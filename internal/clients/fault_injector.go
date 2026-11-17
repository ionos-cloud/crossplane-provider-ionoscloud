package clients

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/crossplane/crossplane-runtime/pkg/logging"
)

// TEST ONLY, never in production (ICNAS-928 dev test, to be reverted). --test-cloudapi-fault makes
// the compute client fake a stuck or failing Cloud API for one datacenter, so the DELETE, status
// poll and throttle handling can be watched without a real outage. While a fault is active, no
// write to its datacenter and no status poll of a fake request ID reaches the Cloud API. Reads
// (GET, HEAD, OPTIONS) of resources always pass, so Observe sees the real state.

// Fault modes, see the --test-cloudapi-fault help.
const (
	faultModeDeleteStuck = "delete-stuck"
	faultModeStatus503   = "status-503"
	faultModeStatusConn  = "status-conn"
	faultModeWrite429    = "write-429"
	faultModeDelete422   = "delete-422"
)

const (
	// fakeRequestIDMiddle is the fixed middle of a fake request ID:
	// <first 8 hex of sha256(dc)>-fa17-4000-8000-<12 random hex>.
	fakeRequestIDMiddle = "-fa17-4000-8000-"
	// faultErrorCode is the errorCode in the body of a fake error response.
	faultErrorCode = "PROVIDER_TEST_FAULT"
	// faultAnswerConnReset is the logged answer for a fake connection reset.
	faultAnswerConnReset = "ECONNRESET"
)

// CloudAPIFaults is the parsed --test-cloudapi-fault config. nil means the flag is not set and the
// injector is not installed. Not nil with no faults (all expired) still answers fake request IDs.
type CloudAPIFaults struct {
	faults []cloudAPIFault
}

// cloudAPIFault is one --test-cloudapi-fault value.
type cloudAPIFault struct {
	dc   string
	mode string
	// dcHash is the first 8 hex of sha256(dc), the start of every fake request ID for dc.
	dcHash string
	// from is zero when not set: active right away.
	from  time.Time
	until time.Time
	// retryAfter is the Retry-After in seconds for write-429, 0 for none.
	retryAfter int
}

// activeAt reports whether now is in [from, until).
func (f cloudAPIFault) activeAt(now time.Time) bool {
	return (f.from.IsZero() || !now.Before(f.from)) && now.Before(f.until)
}

// ParseCloudAPIFaults parses the --test-cloudapi-fault values, each
// dc=<datacenterID>,mode=<mode>,until=<RFC3339>[,from=<RFC3339>][,retryAfter=<seconds>].
// It returns nil for no values. A value whose until is not after now is logged and skipped, so a
// restart after the test window does not fail. Bad syntax, an unknown key or mode, a missing dc,
// mode or until, from not before until, and overlapping windows for one dc are errors.
func ParseCloudAPIFaults(values []string, now time.Time, log logging.Logger) (*CloudAPIFaults, error) {
	if len(values) == 0 {
		return nil, nil
	}
	if log == nil {
		log = logging.NewNopLogger()
	}
	parsed := &CloudAPIFaults{}
	for _, value := range values {
		fault, err := parseCloudAPIFault(value)
		if err != nil {
			return nil, fmt.Errorf("%q: %w", value, err)
		}
		if !now.Before(fault.until) {
			log.Info("TEST ONLY cloudapi fault ignored, until is past", "dc", fault.dc, "mode", fault.mode, "until", faultTimeString(fault.until))
			continue
		}
		for _, other := range parsed.faults {
			if other.dc == fault.dc && faultWindowsOverlap(other, fault) {
				return nil, fmt.Errorf("%q: window overlaps another fault for datacenter %s", value, fault.dc)
			}
		}
		parsed.faults = append(parsed.faults, fault)
	}
	return parsed, nil
}

func parseCloudAPIFault(value string) (cloudAPIFault, error) {
	var fault cloudAPIFault
	seen := map[string]bool{}
	for _, pair := range strings.Split(value, ",") {
		key, val, ok := strings.Cut(pair, "=")
		key, val = strings.TrimSpace(key), strings.TrimSpace(val)
		if !ok || key == "" || val == "" {
			return cloudAPIFault{}, fmt.Errorf("%q is not key=value", pair)
		}
		if seen[key] {
			return cloudAPIFault{}, fmt.Errorf("key %s is set twice", key)
		}
		seen[key] = true
		if err := setCloudAPIFaultField(&fault, key, val); err != nil {
			return cloudAPIFault{}, err
		}
	}
	if err := validateCloudAPIFault(fault); err != nil {
		return cloudAPIFault{}, err
	}
	fault.dcHash = faultDCHash(fault.dc)
	return fault, nil
}

func setCloudAPIFaultField(fault *cloudAPIFault, key, val string) error {
	var err error
	switch key {
	case "dc":
		fault.dc = val
	case "mode":
		fault.mode = val
	case "from":
		fault.from, err = time.Parse(time.RFC3339, val)
	case "until":
		fault.until, err = time.Parse(time.RFC3339, val)
	case "retryAfter":
		fault.retryAfter, err = strconv.Atoi(val)
		if err == nil && fault.retryAfter < 1 {
			err = errors.New("must be at least 1")
		}
	default:
		return fmt.Errorf("unknown key %q", key)
	}
	if err != nil {
		return fmt.Errorf("%s: %w", key, err)
	}
	return nil
}

func validateCloudAPIFault(fault cloudAPIFault) error {
	switch {
	case !isCloudAPIFaultMode(fault.mode):
		return fmt.Errorf("unknown or missing mode %q", fault.mode)
	case fault.dc == "" || strings.Contains(fault.dc, "/"):
		return fmt.Errorf("dc %q is missing or not a datacenter ID", fault.dc)
	case fault.until.IsZero():
		return errors.New("until is missing")
	case fault.retryAfter > 0 && fault.mode != faultModeWrite429:
		return fmt.Errorf("retryAfter is only valid with mode %s", faultModeWrite429)
	case !fault.from.IsZero() && !fault.from.Before(fault.until):
		return errors.New("from must be before until")
	}
	return nil
}

func isCloudAPIFaultMode(mode string) bool {
	switch mode {
	case faultModeDeleteStuck, faultModeStatus503, faultModeStatusConn, faultModeWrite429, faultModeDelete422:
		return true
	default:
		return false
	}
}

// faultWindowsOverlap reports whether [a.from, a.until) and [b.from, b.until) overlap. A zero from
// is open to the past.
func faultWindowsOverlap(a, b cloudAPIFault) bool {
	return (a.from.IsZero() || a.from.Before(b.until)) && (b.from.IsZero() || b.from.Before(a.until))
}

func faultTimeString(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.UTC().Format(time.RFC3339)
}

// cloudAPIFaultConfig and cloudAPIFaultLog are set once by ConfigureCloudAPIFaults and read by
// newIonosClients.
var (
	cloudAPIFaultConfig *CloudAPIFaults
	cloudAPIFaultLog    = logging.NewNopLogger()
)

// ConfigureCloudAPIFaults installs the fault injector in all compute clients created from now on,
// unless faults is nil, and logs every armed fault. Call it once from main, before
// controller.Setup: like computeThrottle it is read without a lock and is not part of the client
// cache key.
func ConfigureCloudAPIFaults(faults *CloudAPIFaults, log logging.Logger) {
	if log == nil {
		log = logging.NewNopLogger()
	}
	cloudAPIFaultConfig, cloudAPIFaultLog = faults, log
	if faults == nil {
		return
	}
	for _, f := range faults.faults {
		log.Info("TEST ONLY cloudapi fault armed", "dc", f.dc, "mode", f.mode, "from", faultTimeString(f.from), "until", faultTimeString(f.until), "retryAfter", f.retryAfter)
	}
}

// faultInjector is the RoundTripper behind --test-cloudapi-fault. It is placed below the throttle
// (computeThrottleRoundTripper), so the throttle sees its fake 429 and 422 like real ones.
type faultInjector struct {
	next   http.RoundTripper
	faults []cloudAPIFault
	log    logging.Logger
	// now is time.Now, replaced in tests.
	now func() time.Time
	// nextShared is true when next is shared with http.DefaultClient, so CloseIdleConnections is
	// not forwarded.
	nextShared bool
}

// withCloudAPIFaults returns a new *http.Client like hc, with its Transport wrapped in the fault
// injector. hc itself is not changed: it is often http.DefaultClient, which the SDKs share.
func withCloudAPIFaults(hc *http.Client, faults *CloudAPIFaults, log logging.Logger) *http.Client {
	var wrapped http.Client
	if hc != nil {
		wrapped = *hc
	}
	next := wrapped.Transport
	shared := hc == nil || hc == http.DefaultClient || next == nil || next == http.DefaultTransport
	if next == nil {
		next = http.DefaultTransport
	}
	if log == nil {
		log = logging.NewNopLogger()
	}
	injector := &faultInjector{next: next, log: log, now: time.Now, nextShared: shared}
	if faults != nil {
		injector.faults = faults.faults
	}
	wrapped.Transport = injector
	return &wrapped
}

func (fi *faultInjector) RoundTrip(req *http.Request) (*http.Response, error) {
	segments := strings.Split(strings.Trim(req.URL.Path, "/"), "/")
	now := fi.now()
	if hash, ok := fakeRequestIDHash(segments); ok {
		closeFaultRequestBody(req)
		return fi.answerFakeRequestID(req, now, hash)
	}
	if !faultIsWrite(req.Method) {
		return fi.next.RoundTrip(req)
	}
	dcIndex := faultDatacenterIndex(segments)
	if dcIndex < 0 {
		return fi.next.RoundTrip(req)
	}
	dc := segments[dcIndex+1]
	fault, ok := fi.activeFault(now, func(f cloudAPIFault) bool { return f.dc == dc })
	if !ok {
		return fi.next.RoundTrip(req)
	}
	closeFaultRequestBody(req)
	return fi.answerWrite(req, fault, segments[:dcIndex])
}

// CloseIdleConnections forwards to the wrapped transport, unless it is shared with
// http.DefaultClient. http.Client.CloseIdleConnections type-asserts for this method.
func (fi *faultInjector) CloseIdleConnections() {
	if fi.nextShared {
		return
	}
	if next, ok := fi.next.(interface{ CloseIdleConnections() }); ok {
		next.CloseIdleConnections()
	}
}

// answerWrite answers a POST, PUT, PATCH or DELETE to the datacenter of an active fault. prefix is
// the path before the datacenters segment, e.g. [cloudapi v6].
func (fi *faultInjector) answerWrite(req *http.Request, fault cloudAPIFault, prefix []string) (*http.Response, error) {
	switch {
	case fault.mode == faultModeWrite429:
		header := http.Header{}
		answer := "429"
		if fault.retryAfter > 0 {
			header.Set("Retry-After", strconv.Itoa(fault.retryAfter))
			answer = fmt.Sprintf("429 retry-after=%d", fault.retryAfter)
		}
		fi.logInjected(req, fault, answer)
		return faultErrorResponse(req, http.StatusTooManyRequests, header, fault)
	case req.Method != http.MethodDelete:
		// No real write; the caller sees a network error.
		fi.logInjected(req, fault, faultAnswerConnReset)
		return nil, faultConnReset()
	case fault.mode == faultModeDelete422:
		fi.logInjected(req, fault, "422")
		return faultErrorResponse(req, http.StatusUnprocessableEntity, http.Header{}, fault)
	default:
		// delete-stuck, status-503, status-conn: accept the DELETE with a fake request.
		id, err := newFakeRequestID(fault.dcHash)
		if err != nil {
			return nil, err
		}
		statusPath := "/requests/" + id + "/status"
		if len(prefix) > 0 {
			statusPath = "/" + strings.Join(prefix, "/") + statusPath
		}
		header := http.Header{}
		header.Set("Location", req.URL.Scheme+"://"+req.URL.Host+statusPath)
		fi.logInjected(req, fault, "202 fake request "+id)
		return faultResponse(req, http.StatusAccepted, header, nil), nil
	}
}

// answerFakeRequestID answers any request for a fake request ID. It is never forwarded.
func (fi *faultInjector) answerFakeRequestID(req *http.Request, now time.Time, hash string) (*http.Response, error) {
	fault, ok := fi.activeFault(now, func(f cloudAPIFault) bool { return f.dcHash == hash })
	if !ok {
		fault = cloudAPIFault{dc: fi.dcForHash(hash), mode: "none"}
	}
	switch fault.mode {
	case faultModeDeleteStuck:
		fi.logInjected(req, fault, "200 RUNNING")
		return faultResponse(req, http.StatusOK, http.Header{}, []byte(`{"metadata":{"status":"RUNNING"}}`)), nil
	case faultModeStatus503:
		fi.logInjected(req, fault, "503")
		return faultErrorResponse(req, http.StatusServiceUnavailable, http.Header{}, fault)
	case faultModeStatusConn:
		fi.logInjected(req, fault, faultAnswerConnReset)
		return nil, faultConnReset()
	default:
		// Window over, unknown hash or a mode without fake requests: the Cloud API would not
		// know the request, so IsDeleteRequestPending returns false and one real DELETE follows.
		fi.logInjected(req, fault, "404 fake request id")
		return faultErrorResponse(req, http.StatusNotFound, http.Header{}, fault)
	}
}

// activeFault returns the first fault that matches and is active at now.
func (fi *faultInjector) activeFault(now time.Time, match func(cloudAPIFault) bool) (cloudAPIFault, bool) {
	for _, f := range fi.faults {
		if match(f) && f.activeAt(now) {
			return f, true
		}
	}
	return cloudAPIFault{}, false
}

func (fi *faultInjector) dcForHash(hash string) string {
	for _, f := range fi.faults {
		if f.dcHash == hash {
			return f.dc
		}
	}
	return "unknown"
}

// logInjected writes the one log line of an intercepted call.
func (fi *faultInjector) logInjected(req *http.Request, fault cloudAPIFault, answer string) {
	fi.log.Info("cloudapi fault injected", "dc", fault.dc, "mode", fault.mode, "method", req.Method, "path", req.URL.Path, "answer", answer)
}

// fakeRequestIDHash returns the dc hash of the first path segment holding a fake request ID. Any
// segment with the fixed middle counts, so a fake ID is never sent to the Cloud API.
func fakeRequestIDHash(segments []string) (string, bool) {
	for _, segment := range segments {
		if hash, _, found := strings.Cut(strings.ToLower(segment), fakeRequestIDMiddle); found {
			return hash, true
		}
	}
	return "", false
}

// faultDatacenterIndex returns the index of the first "datacenters" segment followed by an ID, or
// -1. Works with and without the /cloudapi/v6 prefix.
func faultDatacenterIndex(segments []string) int {
	for i := 0; i+1 < len(segments); i++ {
		if segments[i] == "datacenters" && segments[i+1] != "" {
			return i
		}
	}
	return -1
}

func faultIsWrite(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

// closeFaultRequestBody closes the request body: a RoundTripper must, also when it does not send it.
func closeFaultRequestBody(req *http.Request) {
	if req.Body != nil {
		_ = req.Body.Close()
	}
}

func faultDCHash(dc string) string {
	sum := sha256.Sum256([]byte(dc))
	return hex.EncodeToString(sum[:4])
}

func newFakeRequestID(dcHash string) (string, error) {
	random := make([]byte, 6)
	if _, err := rand.Read(random); err != nil {
		return "", fmt.Errorf("cannot make a fake request id: %w", err)
	}
	return dcHash + fakeRequestIDMiddle + hex.EncodeToString(random), nil
}

// faultConnReset is a connection reset as net/http reports it, so errors.Is(err,
// syscall.ECONNRESET) is true also through the *url.Error of http.Client.
func faultConnReset() error {
	return &net.OpError{Op: "read", Net: "tcp", Err: os.NewSyscallError("read", syscall.ECONNRESET)}
}

// faultErrorBody and faultErrorMessage follow the Cloud API error shape (sdk-go v6 Error), so the
// SDK decodes a fake error like a real one.
type faultErrorBody struct {
	HTTPStatus int                 `json:"httpStatus"`
	Messages   []faultErrorMessage `json:"messages"`
}

type faultErrorMessage struct {
	ErrorCode string `json:"errorCode"`
	Message   string `json:"message"`
}

func faultErrorResponse(req *http.Request, status int, header http.Header, fault cloudAPIFault) (*http.Response, error) {
	body, err := json.Marshal(faultErrorBody{
		HTTPStatus: status,
		Messages: []faultErrorMessage{{
			ErrorCode: faultErrorCode,
			Message:   fmt.Sprintf("TEST ONLY: fake %d from --test-cloudapi-fault, mode %s, datacenter %s", status, fault.mode, fault.dc),
		}},
	})
	if err != nil {
		return nil, fmt.Errorf("cannot build fake Cloud API error: %w", err)
	}
	return faultResponse(req, status, header, body), nil
}

// faultResponse builds a fake response. header must not be nil. An empty body is http.NoBody.
func faultResponse(req *http.Request, status int, header http.Header, body []byte) *http.Response {
	resp := &http.Response{
		Status:     fmt.Sprintf("%d %s", status, http.StatusText(status)),
		StatusCode: status,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     header,
		Body:       http.NoBody,
		Request:    req,
	}
	if len(body) > 0 {
		header.Set("Content-Type", "application/json")
		resp.Body = io.NopCloser(bytes.NewReader(body))
		resp.ContentLength = int64(len(body))
	}
	return resp
}
