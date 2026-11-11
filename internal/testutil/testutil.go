// Package testutil provides helpers for tests that exercise a service client
// against a local stub server, without a live platform or cloud credentials.
//
// It lives under internal/ so it never forms part of the SDK's public API.
package testutil

import (
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/yellowdogai/yellowdog-sdk-go/internal"
)

// NewHTTPClient returns an HTTPClient whose requests are served by handler.
// The stub server is shut down when the test finishes.
//
// Service packages wrap this to build their own client, for example:
//
//	func newTestComputeClient(t *testing.T, h http.HandlerFunc) *ComputeClient {
//		return NewComputeClient(testutil.NewHTTPClient(t, h))
//	}
func NewHTTPClient(t *testing.T, handler http.HandlerFunc) *internal.HTTPClient {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	return internal.NewHTTPClient(&internal.HTTPConfig{
		BaseURL:   server.URL,
		APIKey:    "test-key",
		APISecret: "test-secret",
		Timeout:   5 * time.Second,
	})
}

// Request is a single request observed by a [Recorder].
type Request struct {
	Method string
	Path   string
	Query  string
	Body   string
}

// Recorder is a stub handler that records every request it receives and replies
// with a fixed status and body. It suits assertions about the request the client
// produced — method, path, query string, body — where the response content does
// not matter.
//
// A Recorder is written to from the stub server's goroutine and read from the
// test's, so its accessors are mutex-guarded; do not read the Requests field
// directly.
type Recorder struct {
	status   int
	body     string
	mu       sync.Mutex
	requests []Request
}

// NewRecorder returns a Recorder that replies to every request with the given
// status and body. Pass an empty body for responses the client ignores.
func NewRecorder(status int, body string) *Recorder {
	return &Recorder{status: status, body: body}
}

// Handler returns the http.HandlerFunc to hand to [NewHTTPClient].
func (r *Recorder) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		body, _ := io.ReadAll(req.Body)

		r.mu.Lock()
		r.requests = append(r.requests, Request{
			Method: req.Method,
			Path:   req.URL.Path,
			Query:  req.URL.RawQuery,
			Body:   string(body),
		})
		r.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(r.status)
		if r.body != "" {
			_, _ = w.Write([]byte(r.body))
		}
	}
}

// Count returns the number of requests received.
func (r *Recorder) Count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.requests)
}

// Last returns the most recent request, failing the test if there was none.
func (r *Recorder) Last(t *testing.T) Request {
	t.Helper()
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.requests) == 0 {
		t.Fatal("no request reached the stub server")
	}
	return r.requests[len(r.requests)-1]
}

// All returns every request received, in order.
func (r *Recorder) All() []Request {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]Request(nil), r.requests...)
}
