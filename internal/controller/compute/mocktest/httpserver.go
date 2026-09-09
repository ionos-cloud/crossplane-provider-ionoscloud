package mocktest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
)

// RequestStatusMode controls the HTTP response for /cloudapi/v6/requests/ endpoints.
type RequestStatusMode int32

const (
	StatusModeDone    RequestStatusMode = iota // return DONE
	StatusModeRunning                          // return RUNNING
	StatusModeFailed                           // return FAILED with message
	StatusModeError                            // return HTTP 500
	StatusMode404                              // return HTTP 404
)

// TestHTTPServer wraps an httptest.Server with a configurable request status mode.
type TestHTTPServer struct {
	Server *httptest.Server
	mode   atomic.Int32
}

// NewTestHTTPServer creates a test HTTP server handling /cloudapi/v6/requests/ endpoints.
func NewTestHTTPServer() *TestHTTPServer {
	ts := &TestHTTPServer{}
	mux := http.NewServeMux()

	mux.HandleFunc("/cloudapi/v6/requests/", func(w http.ResponseWriter, r *http.Request) {
		mode := RequestStatusMode(ts.mode.Load())
		w.Header().Set("Content-Type", "application/json")
		switch mode {
		case StatusModeDone:
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":   "test-request-id",
				"type": "request-status",
				"metadata": map[string]any{
					"status": "DONE",
				},
			})
		case StatusModeRunning:
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":   "test-request-id",
				"type": "request-status",
				"metadata": map[string]any{
					"status": "RUNNING",
				},
			})
		case StatusModeFailed:
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":   "test-request-id",
				"type": "request-status",
				"metadata": map[string]any{
					"status":  "FAILED",
					"message": "simulated failure",
				},
			})
		case StatusModeError:
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error": "internal server error",
			})
		case StatusMode404:
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error": "not found",
			})
		}
	})

	ts.Server = httptest.NewServer(mux)
	return ts
}

// SetMode changes the request status response mode.
func (ts *TestHTTPServer) SetMode(mode RequestStatusMode) {
	ts.mode.Store(int32(mode))
}

// Stop shuts down the test HTTP server.
func (ts *TestHTTPServer) Stop() {
	ts.Server.Close()
}

// URL returns the base URL of the test HTTP server.
func (ts *TestHTTPServer) URL() string {
	return ts.Server.URL
}
