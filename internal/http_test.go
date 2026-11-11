package internal

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func newTestHTTPClient(t *testing.T, handler http.HandlerFunc) *HTTPClient {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	return NewHTTPClient(&HTTPConfig{
		BaseURL:   server.URL,
		APIKey:    "test-key",
		APISecret: "test-secret",
		UserAgent: "test-agent",
		Timeout:   5 * time.Second,
	})
}

func TestDo_SetsAuthAndUserAgentHeaders(t *testing.T) {
	var gotAuth, gotUA string
	client := newTestHTTPClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotUA = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
	})

	req, err := client.NewRequest(context.Background(), http.MethodGet, "/anything", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := client.Do(req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotAuth != "yd-key test-key:test-secret" {
		t.Errorf("Authorization header = %q, want %q", gotAuth, "yd-key test-key:test-secret")
	}
	if gotUA != "test-agent" {
		t.Errorf("User-Agent header = %q, want %q", gotUA, "test-agent")
	}
}

func TestGet_DecodesJSONResult(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
	}
	client := newTestHTTPClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/things/1" {
			t.Errorf("path = %s, want /things/1", r.URL.Path)
		}
		if err := json.NewEncoder(w).Encode(payload{Name: "widget"}); err != nil {
			t.Errorf("could not encode payload: %s", err.Error())
		}
	})

	var result payload
	if err := client.Get(context.Background(), "/things/1", &result); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != "widget" {
		t.Errorf("got %q, want widget", result.Name)
	}
}

func TestPost_MarshalsBodyAndSetsContentType(t *testing.T) {
	type request struct {
		Value int `json:"value"`
	}
	type response struct {
		Doubled int `json:"doubled"`
	}

	var gotContentType string
	var gotBody request
	client := newTestHTTPClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotContentType = r.Header.Get("Content-Type")
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Errorf("failed to decode request body: %v", err)
		}
		if err := json.NewEncoder(w).Encode(response{Doubled: gotBody.Value * 2}); err != nil {
			t.Errorf("failed to encode response: %v", err)
		}
	})

	var result response
	if err := client.Post(context.Background(), "/double", request{Value: 21}, &result); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotContentType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", gotContentType)
	}
	if gotBody.Value != 21 {
		t.Errorf("request body value = %d, want 21", gotBody.Value)
	}
	if result.Doubled != 42 {
		t.Errorf("response doubled = %d, want 42", result.Doubled)
	}
}

func TestPost_NilBodyOmitsContentType(t *testing.T) {
	var gotContentType string
	client := newTestHTTPClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotContentType = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusNoContent)
	})

	if err := client.Post(context.Background(), "/ping", nil, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotContentType != "" {
		t.Errorf("Content-Type = %q, want empty for a nil body", gotContentType)
	}
}

func TestPut_RoundTrips(t *testing.T) {
	client := newTestHTTPClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %s, want PUT", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	if err := client.Put(context.Background(), "/things/1", map[string]string{"a": "b"}, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDelete_SendsNoBody(t *testing.T) {
	client := newTestHTTPClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	if err := client.Delete(context.Background(), "/things/1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteWithResult_DecodesBody(t *testing.T) {
	client := newTestHTTPClient(t, func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewEncoder(w).Encode(map[string]string{"status": "deleted"}); err != nil {
			t.Errorf("failed to encode response: %v", err)
		}
	})

	var result map[string]string
	if err := client.DeleteWithResult(context.Background(), "/things/1", &result); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["status"] != "deleted" {
		t.Errorf("got %v, want status=deleted", result)
	}
}

func TestDoJSON_NoContentStatusSkipsDecode(t *testing.T) {
	client := newTestHTTPClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
		// The server refuses a body on a 204 (http.ErrBodyNotAllowed), so this write always fails.
		_, _ = w.Write([]byte("this should never be decoded"))
	})

	var result map[string]string
	if err := client.Get(context.Background(), "/things/1", &result); err != nil {
		t.Fatalf("unexpected error decoding 204 response: %v", err)
	}
	if result != nil {
		t.Errorf("expected result to remain nil for a 204 response, got %v", result)
	}
}

func TestDoJSON_BareStringResultFallsBackToRawBody(t *testing.T) {
	// The API sometimes returns an unquoted bare string; doJSON should use it
	// verbatim rather than failing to unmarshal as JSON.
	client := newTestHTTPClient(t, func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write([]byte("not-valid-json-string")); err != nil {
			t.Errorf("failed to write response: %v", err)
		}
	})

	var result string
	if err := client.Get(context.Background(), "/things/1", &result); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "not-valid-json-string" {
		t.Errorf("got %q, want not-valid-json-string", result)
	}
}

func TestDoJSON_ErrorStatusReturnsAPIError(t *testing.T) {
	client := newTestHTTPClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Request-ID", "req-123")
		w.WriteHeader(http.StatusBadRequest)
		if err := json.NewEncoder(w).Encode(map[string]string{"message": "invalid namespace"}); err != nil {
			t.Errorf("failed to encode response: %v", err)
		}
	})

	err := client.Get(context.Background(), "/things/1", nil)
	if err == nil {
		t.Fatal("expected an error for a 400 response")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != http.StatusBadRequest {
		t.Errorf("StatusCode = %d, want 400", apiErr.StatusCode)
	}
	if apiErr.Message != "invalid namespace" {
		t.Errorf("Message = %q, want %q", apiErr.Message, "invalid namespace")
	}
	if apiErr.RequestId != "req-123" {
		t.Errorf("RequestId = %q, want req-123", apiErr.RequestId)
	}
}

func TestDoJSON_ErrorStatusFallsBackToErrorField(t *testing.T) {
	client := newTestHTTPClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		if err := json.NewEncoder(w).Encode(map[string]string{"error": "boom"}); err != nil {
			t.Errorf("failed to encode response: %v", err)
		}
	})

	err := client.Get(context.Background(), "/things/1", nil)
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T: %v", err, err)
	}
	if apiErr.Message != "boom" {
		t.Errorf("Message = %q, want boom", apiErr.Message)
	}
}

func TestDoJSON_ErrorStatusWithNonJSONBody(t *testing.T) {
	client := newTestHTTPClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		if _, err := w.Write([]byte("<html>gateway down</html>")); err != nil {
			t.Errorf("failed to write response: %v", err)
		}
	})

	err := client.Get(context.Background(), "/things/1", nil)
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T: %v", err, err)
	}
	if apiErr.Message != "" {
		t.Errorf("Message should be empty for a non-JSON error body, got %q", apiErr.Message)
	}
	if string(apiErr.Body) != "<html>gateway down</html>" {
		t.Errorf("Body = %q, want the raw response body", apiErr.Body)
	}
}

func TestAPIError_ErrorString(t *testing.T) {
	withMessage := &APIError{StatusCode: 404, Status: "404 Not Found", Message: "namespace not found"}
	if got := withMessage.Error(); got != "API error 404: namespace not found" {
		t.Errorf("got %q", got)
	}

	withoutMessage := &APIError{StatusCode: 500, Status: "500 Internal Server Error"}
	if got := withoutMessage.Error(); got != "API error 500: 500 Internal Server Error" {
		t.Errorf("got %q", got)
	}
}

func TestBaseURLAndAuthorizationHeader(t *testing.T) {
	client := NewHTTPClient(&HTTPConfig{
		BaseURL:   "https://api.example.com",
		APIKey:    "k",
		APISecret: "s",
	})

	if client.BaseURL() != "https://api.example.com" {
		t.Errorf("BaseURL() = %q", client.BaseURL())
	}
	if client.AuthorizationHeader() != "yd-key k:s" {
		t.Errorf("AuthorizationHeader() = %q", client.AuthorizationHeader())
	}
}

// Asserted over the wire rather than against the client's fields, so it holds
// whichever SSE library is underneath.
func TestNewSSEClient_SendsAuthenticatedRequestToThePath(t *testing.T) {
	requests := make(chan *http.Request, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests <- r.Clone(context.Background())
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	client := NewHTTPClient(&HTTPConfig{
		BaseURL:   server.URL,
		APIKey:    "k",
		APISecret: "s",
		UserAgent: "yellowdog-sdk-go-test",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// The stream closes immediately; only the request it sent matters here.
	_ = client.NewSSEClient("/workerPools/123/updates").Stream(ctx, func(StreamEvent) {})

	select {
	case got := <-requests:
		if got.URL.Path != "/workerPools/123/updates" {
			t.Errorf("path = %q, want /workerPools/123/updates", got.URL.Path)
		}
		if h := got.Header.Get("Authorization"); h != "yd-key k:s" {
			t.Errorf("Authorization = %q, want yd-key k:s", h)
		}
		if h := got.Header.Get("User-Agent"); h != "yellowdog-sdk-go-test" {
			t.Errorf("User-Agent = %q, want yellowdog-sdk-go-test", h)
		}
		if h := got.Header.Get("Accept"); h != "text/event-stream" {
			t.Errorf("Accept = %q, want text/event-stream", h)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("no request reached the server")
	}
}

// A stream is long-lived, so it must not inherit the shared client's timeout.
func TestNewSSEClient_DoesNotInheritTheRequestTimeout(t *testing.T) {
	client := NewHTTPClient(&HTTPConfig{BaseURL: "https://api.example.com", Timeout: 5 * time.Second})

	c, ok := client.NewSSEClient("/updates").(*sseClient)
	if !ok {
		t.Fatalf("NewSSEClient returned %T, want *sseClient", client.NewSSEClient("/updates"))
	}
	if c.client.HTTPClient.Timeout != 0 {
		t.Errorf("stream HTTP client timeout = %v, want 0; a timeout would sever a healthy stream", c.client.HTTPClient.Timeout)
	}
}

func TestNewHTTPClient_DefaultsToStandardClientWhenNoneProvided(t *testing.T) {
	client := NewHTTPClient(&HTTPConfig{BaseURL: "https://api.example.com"})
	if client.httpClient == nil {
		t.Fatal("expected a default *http.Client to be constructed")
	}
}

func TestNewHTTPClient_UsesProvidedClient(t *testing.T) {
	custom := &http.Client{Timeout: 42 * time.Second}
	client := NewHTTPClient(&HTTPConfig{BaseURL: "https://api.example.com", HTTPClient: custom})
	if client.httpClient != custom {
		t.Error("expected the provided *http.Client to be used as-is")
	}
}

// validatingBody stands in for a generated polymorphic model, which implements Validate to
// report a missing type discriminator.
type validatingBody struct {
	Field string `json:"field"`
	err   error
}

func (b validatingBody) Validate() error { return b.err }

func TestRequestBodyIsValidatedBeforeItIsSent(t *testing.T) {
	var reached atomic.Bool
	client := newTestHTTPClient(t, func(w http.ResponseWriter, r *http.Request) {
		reached.Store(true)
		if _, err := w.Write([]byte(`{}`)); err != nil {
			t.Errorf("failed to write response: %v", err)
		}
	})

	want := errors.New("model: Thing.Type is empty, use NewThing")
	err := client.Post(context.Background(), "/things", validatingBody{err: want}, nil)

	if !errors.Is(err, want) {
		t.Fatalf("got %v, want %v", err, want)
	}
	if reached.Load() {
		t.Error("the request was sent; an invalid body must not reach the platform")
	}
}

func TestValidRequestBodyIsSent(t *testing.T) {
	var gotBody string
	client := newTestHTTPClient(t, func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		if _, err := w.Write([]byte(`{}`)); err != nil {
			t.Errorf("failed to write response: %v", err)
		}
	})

	if err := client.Post(context.Background(), "/things", validatingBody{Field: "v"}, nil); err != nil {
		t.Fatalf("Post: %v", err)
	}
	if gotBody != `{"field":"v"}` {
		t.Errorf("body = %s, want {\"field\":\"v\"}", gotBody)
	}
}

// A body with no Validate method is left alone, which is every non-polymorphic request type.
func TestRequestBodyWithoutValidateIsSent(t *testing.T) {
	var reached atomic.Bool
	client := newTestHTTPClient(t, func(w http.ResponseWriter, r *http.Request) {
		reached.Store(true)
		if _, err := w.Write([]byte(`{}`)); err != nil {
			t.Errorf("failed to write response: %v", err)
		}
	})

	if err := client.Post(context.Background(), "/things", map[string]string{"a": "b"}, nil); err != nil {
		t.Fatalf("Post: %v", err)
	}
	if !reached.Load() {
		t.Error("expected the request to be sent")
	}
}
