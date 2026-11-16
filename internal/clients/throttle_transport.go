package clients

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ICNAS-928: when the Cloud API answers 429 for a datacenter, more writes to that datacenter only
// add load and fail again. The compute client can stop sending them for a while and answer them
// locally instead. Off by default; enabled with --api-throttle-wait.

const (
	// retryAfterHeader is read from Cloud API responses and set on synthetic ones.
	retryAfterHeader = "Retry-After"
	// throttledHeader marks a response made by computeThrottleRoundTripper, not by the Cloud API.
	throttledHeader = "X-Ionos-Provider-Throttled"
	// throttledErrorCode is the errorCode in the body of such a response.
	throttledErrorCode = "PROVIDER_THROTTLED"
	// datacentersPathSegment is the path segment followed by the datacenter ID in Cloud API paths.
	datacentersPathSegment = "datacenters"
	// maxRetryAfterSeconds is the largest Retry-After in seconds that fits in a time.Duration.
	maxRetryAfterSeconds = int64(math.MaxInt64 / time.Second)
)

// ComputeThrottleOptions configures how long writes to the compute (Cloud API v6) API are held
// back after the API answered 429, or 422 for a DELETE, PUT or PATCH.
type ComputeThrottleOptions struct {
	// Wait is used when the response has no usable Retry-After header. 0 disables the throttle.
	Wait time.Duration
	// MaxWait caps both Retry-After and Wait.
	MaxWait time.Duration
}

func (o ComputeThrottleOptions) enabled() bool {
	return o.Wait > 0
}

// ValidateComputeThrottleOptions rejects negative durations, and a MaxWait below Wait while the
// throttle is enabled.
func ValidateComputeThrottleOptions(o ComputeThrottleOptions) error {
	if o.Wait < 0 {
		return fmt.Errorf("api throttle wait %s must not be negative", o.Wait)
	}
	if o.MaxWait < 0 {
		return fmt.Errorf("api throttle max wait %s must not be negative", o.MaxWait)
	}
	if o.enabled() && o.MaxWait < o.Wait {
		return fmt.Errorf("api throttle max wait %s must not be less than the wait %s", o.MaxWait, o.Wait)
	}
	return nil
}

// computeThrottle is set once by ConfigureComputeThrottle and read by newIonosClients.
var computeThrottle ComputeThrottleOptions

// ConfigureComputeThrottle sets the throttle for all compute clients created from now on. Call it
// once from main, before controller.Setup: like ionosAPIEndpoint it is read without a lock, and it
// is not part of the client cache key.
func ConfigureComputeThrottle(o ComputeThrottleOptions) {
	computeThrottle = o
}

// throttleNow is time.Now, indirected so expiry can be tested without sleeping.
var throttleNow = time.Now

// throttleBlock holds back writes until a point in time.
type throttleBlock struct {
	until time.Time
	// reason says why, for the synthetic error message.
	reason string
}

// The blocks are process wide, not per client: datacenter IDs are globally unique, and a block
// must survive client cache eviction and apply to every ProviderConfig using the datacenter.
var (
	throttleMu sync.Mutex
	// throttleDatacenterBlocks is keyed by datacenter ID (after a 429).
	throttleDatacenterBlocks = map[string]throttleBlock{}
	// throttleRequestBlocks is keyed by "METHOD path" (after a 422 on a DELETE, PUT or PATCH).
	// POST keys are never stored, see computeThrottleRoundTripper.
	throttleRequestBlocks = map[string]throttleBlock{}
)

// computeThrottleRoundTripper holds back writes to the Cloud API:
//   - after a 429 on any request to /datacenters/{id}/..., all writes to that datacenter;
//   - after a 422 on a DELETE, PUT or PATCH of a resource, the same method and path.
//
// A 422 on POST does not block: POST goes to a collection (e.g. /datacenters/{id}/volumes), so
// its 422 is about that one request body, and blocking the path would hold back every create of
// that kind in the datacenter. Callers that react to a 422 on POST (e.g. the ServerSet boot
// volume controller) still get the real response every time.
//
// While blocked, it answers with a synthetic 429 instead of calling the API. Reads always pass.
// Blocks expire by time only; a successful write does not clear them.
type computeThrottleRoundTripper struct {
	next http.RoundTripper
	opts ComputeThrottleOptions
	// nextShared is true when next is shared with http.DefaultClient. Closing its idle connections
	// would affect every other client using it, so CloseIdleConnections is not forwarded then.
	nextShared bool
}

// withComputeThrottle returns a new *http.Client like hc, with its Transport wrapped in the
// throttle. hc itself is not changed: it is often http.DefaultClient, which the SDKs share.
func withComputeThrottle(hc *http.Client, opts ComputeThrottleOptions) *http.Client {
	var wrapped http.Client
	if hc != nil {
		wrapped = *hc
	}
	next := wrapped.Transport
	shared := hc == nil || hc == http.DefaultClient || next == nil || next == http.DefaultTransport
	if next == nil {
		next = http.DefaultTransport
	}
	wrapped.Transport = &computeThrottleRoundTripper{next: next, opts: opts, nextShared: shared}
	return &wrapped
}

func (t *computeThrottleRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	datacenterID := datacenterIDFromPath(req.URL.Path)
	requestKey := req.Method + " " + req.URL.Path
	write := isWriteMethod(req.Method)

	if write {
		now := throttleNow()
		if block, ok := activeThrottleBlock(now, datacenterID, requestKey); ok {
			// A RoundTripper must close the request body, also when it does not send it.
			if req.Body != nil {
				_ = req.Body.Close()
			}
			return throttledResponse(req, now, block)
		}
	}

	resp, err := t.next.RoundTrip(req)
	if err != nil {
		return resp, err
	}

	switch {
	case resp.StatusCode == http.StatusTooManyRequests && datacenterID != "":
		addThrottleBlock(throttleDatacenterBlocks, datacenterID, t.untilFor(resp),
			fmt.Sprintf("Cloud API %d for datacenter %s", resp.StatusCode, datacenterID))
	case resp.StatusCode == http.StatusUnprocessableEntity && isResourceWriteMethod(req.Method):
		addThrottleBlock(throttleRequestBlocks, requestKey, t.untilFor(resp),
			fmt.Sprintf("Cloud API %d for %s", resp.StatusCode, requestKey))
	}
	return resp, nil
}

// CloseIdleConnections forwards to the wrapped transport, unless it is shared with
// http.DefaultClient. http.Client.CloseIdleConnections type-asserts for this method.
func (t *computeThrottleRoundTripper) CloseIdleConnections() {
	if t.nextShared {
		return
	}
	if next, ok := t.next.(interface{ CloseIdleConnections() }); ok {
		next.CloseIdleConnections()
	}
}

// untilFor returns the end of the block for resp: now + min(Retry-After or Wait, MaxWait).
func (t *computeThrottleRoundTripper) untilFor(resp *http.Response) time.Time {
	now := throttleNow()
	return now.Add(throttleWait(resp.Header.Get(retryAfterHeader), now, t.opts))
}

// throttleWait returns the Retry-After value if it is usable, else opts.Wait, capped at opts.MaxWait.
func throttleWait(retryAfter string, now time.Time, opts ComputeThrottleOptions) time.Duration {
	wait := opts.Wait
	if d, ok := parseRetryAfter(retryAfter, now); ok {
		wait = d
	}
	if wait > opts.MaxWait {
		wait = opts.MaxWait
	}
	return wait
}

// parseRetryAfter parses a Retry-After header as seconds or as an HTTP-date. Empty, invalid, zero,
// negative and too large values, and dates not in the future, are not usable.
func parseRetryAfter(value string, now time.Time) (time.Duration, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}
	if seconds, err := strconv.ParseInt(value, 10, 64); err == nil {
		if seconds <= 0 || seconds > maxRetryAfterSeconds {
			return 0, false
		}
		return time.Duration(seconds) * time.Second, true
	}
	if date, err := http.ParseTime(value); err == nil {
		if d := date.Sub(now); d > 0 {
			return d, true
		}
	}
	return 0, false
}

// datacenterIDFromPath returns the segment after the first "datacenters" segment of p, or "" if
// there is none. Works with and without the /cloudapi/v6 prefix.
func datacenterIDFromPath(p string) string {
	segments := strings.Split(strings.Trim(p, "/"), "/")
	for i := 0; i+1 < len(segments); i++ {
		if segments[i] == datacentersPathSegment {
			return segments[i+1]
		}
	}
	return ""
}

func isWriteMethod(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

// isResourceWriteMethod reports whether method writes a specific resource path, so that a 422
// for it can block the same method and path. POST is excluded, see computeThrottleRoundTripper.
func isResourceWriteMethod(method string) bool {
	switch method {
	case http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

// addThrottleBlock blocks key until the given time. An existing block that lasts longer is kept.
func addThrottleBlock(blocks map[string]throttleBlock, key string, until time.Time, reason string) {
	throttleMu.Lock()
	defer throttleMu.Unlock()
	if current, ok := blocks[key]; ok && !current.until.Before(until) {
		return
	}
	blocks[key] = throttleBlock{until: until, reason: reason}
}

// activeThrottleBlock drops expired blocks, then returns the one that applies to a write with the
// given datacenter ID and request key. If both apply, the one that lasts longer is returned.
func activeThrottleBlock(now time.Time, datacenterID, requestKey string) (throttleBlock, bool) {
	throttleMu.Lock()
	defer throttleMu.Unlock()
	dropExpiredThrottleBlocksLocked(throttleDatacenterBlocks, now)
	dropExpiredThrottleBlocksLocked(throttleRequestBlocks, now)

	block, found := throttleRequestBlocks[requestKey]
	if datacenterID != "" {
		if dcBlock, ok := throttleDatacenterBlocks[datacenterID]; ok && (!found || dcBlock.until.After(block.until)) {
			block, found = dcBlock, true
		}
	}
	return block, found
}

func dropExpiredThrottleBlocksLocked(blocks map[string]throttleBlock, now time.Time) {
	for key, block := range blocks {
		if !now.Before(block.until) {
			delete(blocks, key)
		}
	}
}

// throttledErrorBody and throttledErrorMessage follow the Cloud API error shape (sdk-go v6
// Error / ErrorMessage), so the SDK decodes the synthetic response like a real one.
type throttledErrorBody struct {
	HTTPStatus int                     `json:"httpStatus"`
	Messages   []throttledErrorMessage `json:"messages"`
}

type throttledErrorMessage struct {
	ErrorCode string `json:"errorCode"`
	Message   string `json:"message"`
}

// throttledResponse builds the synthetic 429 returned instead of calling the API.
func throttledResponse(req *http.Request, now time.Time, block throttleBlock) (*http.Response, error) {
	body, err := json.Marshal(throttledErrorBody{
		HTTPStatus: http.StatusTooManyRequests,
		Messages: []throttledErrorMessage{{
			ErrorCode: throttledErrorCode,
			Message: fmt.Sprintf("throttled locally by %s until %s after %s",
				UserAgent, block.until.UTC().Format(time.RFC3339), block.reason),
		}},
	})
	if err != nil {
		return nil, fmt.Errorf("cannot build throttled response: %w", err)
	}

	// Remaining time in whole seconds, rounded up, and at least 1.
	remaining := block.until.Sub(now)
	retryAfter := int64(remaining / time.Second)
	if remaining%time.Second != 0 || retryAfter < 1 {
		retryAfter++
	}

	header := http.Header{}
	header.Set("Content-Type", "application/json")
	header.Set(retryAfterHeader, strconv.FormatInt(retryAfter, 10))
	header.Set(throttledHeader, "true")

	return &http.Response{
		Status:        fmt.Sprintf("%d %s", http.StatusTooManyRequests, http.StatusText(http.StatusTooManyRequests)),
		StatusCode:    http.StatusTooManyRequests,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        header,
		Body:          io.NopCloser(bytes.NewReader(body)),
		ContentLength: int64(len(body)),
		Request:       req,
	}, nil
}
