package mocktest

import (
	"net/http"
	"sync"
	"time"

	sdkgo "github.com/ionos-cloud/sdk-go/v6"
)

const (
	MethodCreate = "create"
	MethodGet    = "get"
	MethodUpdate = "update"
	MethodDelete = "delete"
)

// FakeServiceBase provides common error injection, API client setup, and response helpers
// for all fake service implementations. Embed this struct in your resource-specific fake service.
type FakeServiceBase struct {
	Mu        sync.Mutex
	CreateErr error
	GetErr    error
	UpdateErr error
	DeleteErr error

	DeleteCalls []string

	APIClient *sdkgo.APIClient
	ServerURL string
}

// NewFakeServiceBase creates a FakeServiceBase with an SDK client configured for the test server.
func NewFakeServiceBase(serverURL string) FakeServiceBase {
	cfg := sdkgo.NewConfiguration("", "", "test-token", serverURL)
	cfg.PollInterval = 100 * time.Millisecond
	return FakeServiceBase{
		APIClient: sdkgo.NewAPIClient(cfg),
		ServerURL: serverURL,
	}
}

// SetError sets a per-method error that the fake service should return.
func (f *FakeServiceBase) SetError(method string, err error) {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	switch method {
	case MethodCreate:
		f.CreateErr = err
	case MethodGet:
		f.GetErr = err
	case MethodUpdate:
		f.UpdateErr = err
	case MethodDelete:
		f.DeleteErr = err
	}
}

// ClearErrors resets all per-method errors to nil.
func (f *FakeServiceBase) ClearErrors() {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	f.CreateErr = nil
	f.GetErr = nil
	f.UpdateErr = nil
	f.DeleteErr = nil
}

// GetDeleteCalls returns a copy of the recorded delete call IDs.
func (f *FakeServiceBase) GetDeleteCalls() []string {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	result := make([]string, len(f.DeleteCalls))
	copy(result, f.DeleteCalls)
	return result
}

// GetAPIClient returns the SDK API client.
func (f *FakeServiceBase) GetAPIClient() *sdkgo.APIClient {
	return f.APIClient
}

// AcceptedResponse creates an *sdkgo.APIResponse with HTTP 202 and a Location header
// pointing to the test server's request status endpoint.
func (f *FakeServiceBase) AcceptedResponse(reqType, resourceID string) *sdkgo.APIResponse {
	header := http.Header{}
	header.Set("Location", f.ServerURL+"/cloudapi/v6/requests/"+reqType+"-req-"+resourceID+"/status")
	return &sdkgo.APIResponse{
		Response: &http.Response{
			StatusCode: http.StatusAccepted,
			Header:     header,
		},
	}
}

// ErrorResponse creates an *sdkgo.APIResponse with the given HTTP status code.
func ErrorResponse(statusCode int) *sdkgo.APIResponse {
	return &sdkgo.APIResponse{
		Response: &http.Response{StatusCode: statusCode},
	}
}

// OKResponse creates an *sdkgo.APIResponse with HTTP 200.
func OKResponse() *sdkgo.APIResponse {
	return &sdkgo.APIResponse{
		Response: &http.Response{StatusCode: http.StatusOK},
	}
}
