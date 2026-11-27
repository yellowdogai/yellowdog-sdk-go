package internal

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestDoJSON_RetriesRateLimitsAndServerErrors(t *testing.T) {
	for _, status := range []int{
		http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusServiceUnavailable,
	} {
		t.Run(http.StatusText(status), func(t *testing.T) {

			// Given
			var requests atomic.Int32
			server := newTestHTTPServer(t, func(w http.ResponseWriter, r *http.Request) {
				// return the error status on first request, otherwise 200 OK
				if requests.Add(1) == 1 {
					w.WriteHeader(status)
					return
				}
				w.WriteHeader(http.StatusOK)
			})
			client := newClient(server, 5*time.Second, 3)

			// When
			err := client.Get(context.Background(), "/things/1", nil)

			// Then
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if requests.Load() != 2 {
				t.Errorf("requests = %d, want 2", requests.Load())
			}
		})
	}
}

func TestDoJSON_SendsTheSameBodyOnEveryAttempt(t *testing.T) {
	// Given
	var mu sync.Mutex
	var bodies []string
	// sends a 503, then a 200 OK, and captures the sent body in bodies
	server := newTestHTTPServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		bodies = append(bodies, string(body))
		n := len(bodies)
		mu.Unlock()
		if n == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	client := newClient(server, 5*time.Second, 3)

	// When
	err := client.Post(context.Background(), "/things", map[string]int{"value": 1}, nil)

	// Then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	want := []string{`{"value":1}`, `{"value":1}`}
	if !slices.Equal(bodies, want) {
		t.Errorf("bodies = %q, want %q", bodies, want)
	}
}

func TestDoJSON_ReturnsTheLastErrorWhenRetriesRunOut(t *testing.T) {
	// Given
	var requests atomic.Int32
	// returns 502 status on first request, otherwise 400 bad request
	server := newTestHTTPServer(t, func(w http.ResponseWriter, r *http.Request) {
		if requests.Add(1) == 1 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusTooManyRequests)
	})
	client := newClient(server, 5*time.Second, 2)

	// When
	err := client.Get(context.Background(), "/things/1", nil)

	// Then
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusTooManyRequests {
		t.Errorf("error = %v, want a 429 APIError", err)
	}
	if requests.Load() != 3 {
		t.Errorf("requests = %d, want 3", requests.Load())
	}
}

func TestDoJSON_DoesNotRetryClientErrors(t *testing.T) {
	for _, status := range []int{
		http.StatusBadRequest,
		http.StatusUnauthorized,
		http.StatusForbidden,
		http.StatusNotFound,
	} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			//Given
			var requests atomic.Int32
			server := newTestHTTPServer(t, func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				w.WriteHeader(status)
			})
			client := newClient(server, 5*time.Second, 3)

			// When
			err := client.Get(context.Background(), "/things/1", nil)

			// Then
			if err == nil {
				t.Fatal("expected an error")
			}
			if requests.Load() != 1 {
				t.Errorf("requests = %d, want 1", requests.Load())
			}
		})
	}
}

func TestDoJSON_DoesNotRetryWhenMaxAttemptsIsZero(t *testing.T) {
	// Given
	var requests atomic.Int32
	server := newTestHTTPServer(t, func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	})
	client := newClient(server, 5*time.Second, 0)

	// When
	err := client.Get(context.Background(), "/things/1", nil)

	// Then
	if err == nil {
		t.Fatal("expected an error")
	}
	if requests.Load() != 1 {
		t.Errorf("requests = %d, want 1", requests.Load())
	}
}

func TestDoJSON_RetriesConnectionErrors(t *testing.T) {
	for _, method := range []string{
		http.MethodGet,
		http.MethodPost,
	} {
		t.Run(method, func(t *testing.T) {
			// Given
			var requests atomic.Int32
			server := newTestHTTPServer(t, func(w http.ResponseWriter, r *http.Request) {
				if requests.Add(1) == 1 {
					conn, _, err := http.NewResponseController(w).Hijack()
					if err != nil {
						t.Errorf("hijacking the connection: %v", err)
						return
					}
					_ = conn.Close()
					return
				}
				w.WriteHeader(http.StatusOK)
			})
			client := newClient(server, 5*time.Second, 3)

			// When
			req, err := client.NewRequest(context.Background(), method, "/things", nil)

			// Then
			if err != nil {
				t.Fatal(err)
			}
			if err := client.doJSON(req, nil, nil); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if requests.Load() != 2 {
				t.Errorf("requests = %d, want 2", requests.Load())
			}
		})
	}
}

func TestDoJSON_RetriesTimeoutsExceptForPost(t *testing.T) {
	for method, wantRequests := range map[string]int32{
		http.MethodGet:  2,
		http.MethodPut:  2,
		http.MethodPost: 1, // Posts should not be retried, the Platform may still be processing the previous request
	} {
		t.Run(method, func(t *testing.T) {
			// Given
			var requests atomic.Int32
			server := newTestHTTPServer(t, func(w http.ResponseWriter, r *http.Request) {
				if requests.Add(1) == 1 {
					select {
					case <-r.Context().Done():
					case <-time.After(5 * time.Second):
					}
					return
				}
				w.WriteHeader(http.StatusOK)
			})
			client := newClient(server, 100*time.Millisecond, 3)

			// When
			req, err := client.NewRequest(context.Background(), method, "/things", nil)

			// Then
			if err != nil {
				t.Fatal(err)
			}
			err = client.doJSON(req, nil, nil)
			if (err == nil) != (wantRequests == 2) {
				t.Errorf("error = %v", err)
			}
			if requests.Load() != wantRequests {
				t.Errorf("requests = %d, want %d", requests.Load(), wantRequests)
			}
		})
	}
}

func TestDoJSON_RetriesAPostThatTimedOutBeforeItWasSent(t *testing.T) {
	// Given
	var dials atomic.Int32
	client := NewHTTPClient(&HTTPConfig{
		BaseURL: "https://platform.example.com",
		HTTPClient: &http.Client{
			Timeout: 100 * time.Millisecond,
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
					dials.Add(1)
					<-ctx.Done()
					return nil, ctx.Err()
				},
			},
		},
		Retry: Retry{MaxAttempts: 1},
	})

	// When
	err := client.Post(context.Background(), "/things", map[string]int{"value": 1}, nil)

	// Then
	if err == nil {
		t.Fatal("expected an error")
	}
	if dials.Load() != 2 {
		t.Errorf("dials = %d, want 2", dials.Load())
	}
}

func TestDoJSON_RetriesAResponseBodyCutShortExceptForPost(t *testing.T) {
	for method, wantRequests := range map[string]int32{
		http.MethodGet:  2,
		http.MethodPut:  2,
		http.MethodPost: 1, // Posts should not be retried, the Platform has already processed the previous request
	} {
		t.Run(method, func(t *testing.T) {
			// Given
			var requests atomic.Int32
			// cuts the body short on the first request, otherwise returns the whole body
			server := newTestHTTPServer(t, func(w http.ResponseWriter, r *http.Request) {
				if requests.Add(1) == 1 {
					w.Header().Set("Content-Length", "100")
					_, _ = w.Write([]byte(`{"val`))
					return
				}
				_, _ = w.Write([]byte(`{"value":2}`))
			})
			client := newClient(server, 5*time.Second, 3)

			// When
			req, err := client.NewRequest(context.Background(), method, "/things", nil)
			if err != nil {
				t.Fatal(err)
			}
			err = client.doJSON(req, nil, nil)

			// Then
			if (err == nil) != (wantRequests == 2) {
				t.Errorf("error = %v", err)
			}
			if requests.Load() != wantRequests {
				t.Errorf("requests = %d, want %d", requests.Load(), wantRequests)
			}
		})
	}
}

func TestDoJSON_StopsWaitingToRetryWhenTheContextIsDone(t *testing.T) {
	// Given
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(server.Close)
	client := NewHTTPClient(&HTTPConfig{
		BaseURL: server.URL,
		Retry:   Retry{MaxAttempts: 3, InitialInterval: time.Hour, MaxInterval: time.Hour},
	})
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// When
	err := client.Get(ctx, "/things/1", nil)

	// Then
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("error = %v, want %v", err, context.DeadlineExceeded)
	}
}

func TestBackoff_WaitsUpToTheInitialIntervalDoubledForEachRetryAndCapped(t *testing.T) {
	client := &HTTPClient{retry: Retry{MaxAttempts: 5, InitialInterval: time.Second, MaxInterval: 3 * time.Second}}

	for retry, limit := range []time.Duration{
		time.Second,
		2 * time.Second,
		3 * time.Second,
		3 * time.Second,
	} {
		// Given
		var longest time.Duration

		// When
		for range 1000 {
			wait := client.backoff(retry)
			if wait < 0 || wait >= limit {
				t.Fatalf("retry %d waited %v, want [0, %v)", retry, wait, limit)
			}
			longest = max(longest, wait)
		}

		// Then
		if longest < limit*9/10 {
			t.Errorf("retry %d waited at most %v over 1000 retries, want close to %v", retry, longest, limit)
		}
	}
}

func newClient(server *httptest.Server, timeout time.Duration, maxAttempts int) *HTTPClient {
	return NewHTTPClient(&HTTPConfig{
		BaseURL: server.URL,
		Timeout: timeout,
		Retry:   Retry{MaxAttempts: maxAttempts},
	})
}

func newTestHTTPServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return server
}
