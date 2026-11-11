package internal

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type valueEvent struct {
	Value int `json:"value"`
}

// eventWriter writes SSE frames to a live response.
type eventWriter struct {
	t *testing.T
	r *http.Request
	w http.ResponseWriter
	f http.Flusher
}

func (e *eventWriter) send(eventType, data string) {
	var frame strings.Builder
	if eventType != "" {
		frame.WriteString("event: " + eventType + "\n")
	}
	if data != "" {
		frame.WriteString("data: " + data + "\n")
	}
	frame.WriteString("\n")
	// A write can fail once the client has its answer and hangs up; only a
	// failure while it is still listening means the event was lost.
	if _, err := io.WriteString(e.w, frame.String()); err != nil && e.r.Context().Err() == nil {
		e.t.Errorf("failed to write event: %v", err)
	}
	e.f.Flush()
}

// sseServer is a stub of the platform's entity update stream. script runs per
// request; returning from it closes the stream, as the platform does after
// entityFinalised.
type sseServer struct {
	*httptest.Server
	requests  atomic.Int32
	openConns atomic.Int32
}

func newSSEServer(t *testing.T, script func(w *eventWriter, r *http.Request)) *sseServer {
	t.Helper()
	s := &sseServer{}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.requests.Add(1)
		s.openConns.Add(1)
		defer s.openConns.Add(-1)

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.WriteHeader(http.StatusOK)
		f, ok := w.(http.Flusher)
		if !ok {
			t.Error("ResponseWriter is not a Flusher")
			return
		}
		f.Flush()
		script(&eventWriter{t: t, r: r, w: w, f: f}, r)
	}))
	t.Cleanup(s.Close)
	return s
}

func (s *sseServer) waiter(t *testing.T) *SSEWaiter[valueEvent] {
	t.Helper()
	hc := NewHTTPClient(&HTTPConfig{BaseURL: s.URL, APIKey: "test-key", APISecret: "test-secret"})
	return NewSSEWaiter[valueEvent](hc.NewSSEClient("/updates"))
}

func isFive(v *valueEvent) bool { return v.Value == 5 }

func testCtx(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func TestSSEWaiter_ReturnsOnMatchingEntityUpdated(t *testing.T) {
	server := newSSEServer(t, func(w *eventWriter, r *http.Request) {
		w.send(EventEntityUpdated, `{"value":1}`)
		w.send(EventEntityUpdated, `{"value":5}`)
		<-r.Context().Done()
	})

	result, err := server.waiter(t).WaitUntil(testCtx(t), isFive)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Value != 5 {
		t.Errorf("got %d, want 5", result.Value)
	}
}

// entityStable is the stream's heartbeat and carries no entity. It must not end
// the wait, and must not be mistaken for an update.
func TestSSEWaiter_SkipsStableHeartbeats(t *testing.T) {
	server := newSSEServer(t, func(w *eventWriter, r *http.Request) {
		w.send(EventEntityStable, "")
		w.send(EventEntityStable, "")
		w.send(EventEntityUpdated, `{"value":5}`)
		<-r.Context().Done()
	})

	result, err := server.waiter(t).WaitUntil(testCtx(t), isFive)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Value != 5 {
		t.Errorf("got %d, want 5", result.Value)
	}
}

func TestSSEWaiter_ReturnsMatchingEntityFinalised(t *testing.T) {
	server := newSSEServer(t, func(w *eventWriter, r *http.Request) {
		w.send(EventEntityFinalised, `{"value":5}`)
	})

	result, err := server.waiter(t).WaitUntil(testCtx(t), isFive)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Value != 5 {
		t.Errorf("got %d, want 5", result.Value)
	}
}

// The entity reached a terminal state that the predicate does not accept. No
// further events can arrive, so the wait must end at once rather than block
// until the caller's context expires.
func TestSSEWaiter_EntityFinalisedWithoutMatchFailsImmediately(t *testing.T) {
	server := newSSEServer(t, func(w *eventWriter, r *http.Request) {
		w.send(EventEntityFinalised, `{"value":9}`)
	})

	start := time.Now()
	_, err := server.waiter(t).WaitUntil(testCtx(t), isFive)
	if !errors.Is(err, ErrEntityFinalised) {
		t.Fatalf("got %v, want ErrEntityFinalised", err)
	}
	if elapsed := time.Since(start); elapsed > 3*time.Second {
		t.Errorf("took %v; should fail as soon as the finalised event arrives", elapsed)
	}
}

// A stream that closes without finalising the entity is reported. The library
// treats an orderly close as success and goes silent, so this is the case that
// would otherwise hang.
func TestSSEWaiter_StreamClosedWithoutFinalisedIsReported(t *testing.T) {
	server := newSSEServer(t, func(w *eventWriter, r *http.Request) {
		w.send(EventEntityUpdated, `{"value":1}`)
		// return, closing the stream
	})

	start := time.Now()
	_, err := server.waiter(t).WaitUntil(testCtx(t), isFive)
	if !errors.Is(err, ErrStreamEnded) {
		t.Fatalf("got %v, want ErrStreamEnded", err)
	}
	if elapsed := time.Since(start); elapsed > 3*time.Second {
		t.Errorf("took %v; a closed stream should be reported promptly", elapsed)
	}
	// Reconnection must be off. Enabled, the library reopens the stream and
	// redelivers its opening event indefinitely instead of reporting the close.
	if got := server.requests.Load(); got != 1 {
		t.Errorf("server saw %d requests, want 1; the stream must not be reopened", got)
	}
}

// A stream outlives any single request, so HTTPConfig.Timeout must not apply to
// it. Sharing the configured client would sever this stream mid-wait.
func TestSSEWaiter_SlowStreamOutlivesTheRequestTimeout(t *testing.T) {
	const requestTimeout = 300 * time.Millisecond

	server := newSSEServer(t, func(w *eventWriter, r *http.Request) {
		time.Sleep(4 * requestTimeout)
		w.send(EventEntityUpdated, `{"value":5}`)
		<-r.Context().Done()
	})

	hc := NewHTTPClient(&HTTPConfig{
		BaseURL:   server.URL,
		APIKey:    "test-key",
		APISecret: "test-secret",
		Timeout:   requestTimeout,
	})
	waiter := NewSSEWaiter[valueEvent](hc.NewSSEClient("/updates"))

	result, err := waiter.WaitUntil(testCtx(t), isFive)
	if err != nil {
		t.Fatalf("stream was severed by the request timeout: %v", err)
	}
	if result.Value != 5 {
		t.Errorf("got %d, want 5", result.Value)
	}
}

// A non-200 is a permanent failure: the endpoint is wrong or the credentials
// were rejected, so it must reach the caller immediately rather than be retried.
func TestSSEWaiter_ConnectFailureSurfacesImmediately(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)

	hc := NewHTTPClient(&HTTPConfig{BaseURL: server.URL, APIKey: "k", APISecret: "s"})
	waiter := NewSSEWaiter[valueEvent](hc.NewSSEClient("/updates"))

	start := time.Now()
	_, err := waiter.WaitUntil(testCtx(t), isFive)
	if err == nil {
		t.Fatal("expected an error for a 404 stream")
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Errorf("took %v; a 404 should be reported immediately", elapsed)
	}
	t.Logf("reported after %v: %v", time.Since(start).Round(time.Millisecond), err)
}

// The platform heartbeats every polling interval, so prolonged silence on an
// open connection means the stream is no longer live.
func TestSSEWaiter_IdleStreamTimesOut(t *testing.T) {
	server := newSSEServer(t, func(w *eventWriter, r *http.Request) {
		<-r.Context().Done() // connection open, no events
	})

	waiter := server.waiter(t).WithIdleTimeout(300 * time.Millisecond)
	_, err := waiter.WaitUntil(testCtx(t), isFive)
	if !errors.Is(err, ErrStreamIdle) {
		t.Fatalf("got %v, want ErrStreamIdle", err)
	}
}

// Each event refreshes the idle deadline, so a slow but live stream is not
// mistaken for a dead one.
func TestSSEWaiter_HeartbeatsRefreshIdleDeadline(t *testing.T) {
	server := newSSEServer(t, func(w *eventWriter, r *http.Request) {
		for i := 0; i < 5; i++ {
			time.Sleep(150 * time.Millisecond)
			w.send(EventEntityStable, "")
		}
		w.send(EventEntityUpdated, `{"value":5}`)
		<-r.Context().Done()
	})

	// Total run exceeds the idle timeout; no single gap does.
	waiter := server.waiter(t).WithIdleTimeout(400 * time.Millisecond)
	result, err := waiter.WaitUntil(testCtx(t), isFive)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Value != 5 {
		t.Errorf("got %d, want 5", result.Value)
	}
}

func TestSSEWaiter_ClosesConnectionOnSuccess(t *testing.T) {
	server := newSSEServer(t, func(w *eventWriter, r *http.Request) {
		w.send(EventEntityUpdated, `{"value":5}`)
		<-r.Context().Done()
	})

	if _, err := server.waiter(t).WaitUntil(testCtx(t), isFive); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if server.openConns.Load() == 0 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Errorf("%d connection(s) still open after the wait returned", server.openConns.Load())
}

func TestSSEWaiter_ContextCancellationReturnsContextError(t *testing.T) {
	server := newSSEServer(t, func(w *eventWriter, r *http.Request) {
		<-r.Context().Done()
	})

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		_, err := server.waiter(t).WaitUntil(ctx, isFive)
		errCh <- err
	}()

	time.Sleep(200 * time.Millisecond)
	cancel()

	select {
	case err := <-errCh:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("got %v, want context.Canceled", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("WaitUntil did not return after cancellation")
	}
}

// The primer is the one read of current state, evaluated like any update.
func TestSSEWaiter_PrimerCanSatisfyTheWait(t *testing.T) {
	server := newSSEServer(t, func(w *eventWriter, r *http.Request) {
		<-r.Context().Done() // stream reports nothing
	})

	var primerCalls atomic.Int32
	waiter := server.waiter(t).WithPrimer(func(ctx context.Context) (*valueEvent, error) {
		primerCalls.Add(1)
		return &valueEvent{Value: 5}, nil
	})

	result, err := waiter.WaitUntil(testCtx(t), isFive)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Value != 5 {
		t.Errorf("got %d, want 5", result.Value)
	}
	if got := primerCalls.Load(); got != 1 {
		t.Errorf("primer called %d times, want exactly 1", got)
	}
}

func TestSSEWaiter_PrimerErrorIsReturned(t *testing.T) {
	server := newSSEServer(t, func(w *eventWriter, r *http.Request) {
		<-r.Context().Done()
	})

	wantErr := errors.New("requirement not found")
	waiter := server.waiter(t).WithPrimer(func(ctx context.Context) (*valueEvent, error) {
		return nil, wantErr
	})

	_, err := waiter.WaitUntil(testCtx(t), isFive)
	if !errors.Is(err, wantErr) {
		t.Errorf("got %v, want it to wrap %v", err, wantErr)
	}
}

func TestSSEWaiter_MalformedEventDataIsReported(t *testing.T) {
	server := newSSEServer(t, func(w *eventWriter, r *http.Request) {
		w.send(EventEntityUpdated, `not json`)
		<-r.Context().Done()
	})

	_, err := server.waiter(t).WaitUntil(testCtx(t), isFive)
	if err == nil {
		t.Fatal("expected malformed event data to be reported")
	}
	t.Logf("reported: %v", err)
}

// The request the client actually sends: path, Accept and credentials.
func TestSSEWaiter_SendsAuthenticatedStreamRequest(t *testing.T) {
	type request struct{ path, accept, auth string }
	requests := make(chan request, 1)

	server := newSSEServer(t, func(w *eventWriter, r *http.Request) {
		requests <- request{
			path:   r.URL.Path,
			accept: r.Header.Get("Accept"),
			auth:   r.Header.Get("Authorization"),
		}
		w.send(EventEntityUpdated, `{"value":5}`)
		<-r.Context().Done()
	})

	if _, err := server.waiter(t).WaitUntil(testCtx(t), isFive); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := <-requests
	if got.path != "/updates" {
		t.Errorf("path = %q, want /updates", got.path)
	}
	if got.accept != "text/event-stream" {
		t.Errorf("Accept = %q, want text/event-stream", got.accept)
	}
	if got.auth != "yd-key test-key:test-secret" {
		t.Errorf("Authorization = %q, want yd-key test-key:test-secret", got.auth)
	}
}
