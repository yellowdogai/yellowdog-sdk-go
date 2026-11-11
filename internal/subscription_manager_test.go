package internal

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/yellowdogai/yellowdog-sdk-go/subscription"
)

type fakeManagedSSEClient struct {
	ready   chan struct{}
	stopped chan struct{}
	ended   chan error
	handler func(StreamEvent)
}

func newFakeManagedSSEClient() *fakeManagedSSEClient {
	return &fakeManagedSSEClient{
		ready:   make(chan struct{}),
		stopped: make(chan struct{}),
		ended:   make(chan error, 1),
	}
}

// Stream blocks until the context is cancelled or the stream is ended, as the real client does.
func (f *fakeManagedSSEClient) Stream(ctx context.Context, handler func(StreamEvent)) error {
	f.handler = handler
	close(f.ready)
	defer close(f.stopped)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-f.ended:
		return err
	}
}

// emit delivers an event as the read loop would.
func (f *fakeManagedSSEClient) emit(event StreamEvent) { f.handler(event) }

// end ends the stream with err, as the server closing it (a wrapped io.EOF) or a failure would.
func (f *fakeManagedSSEClient) end(err error) { f.ended <- err }

// recorder is a listener that records each of its calls.
type recorder struct {
	updates   chan *valueEvent
	errors    chan error
	cancelled chan struct{}
}

func newRecorder() *recorder {
	return &recorder{
		updates:   make(chan *valueEvent, 8),
		errors:    make(chan error, 8),
		cancelled: make(chan struct{}, 8),
	}
}

func (r *recorder) listener() subscription.SubscriptionEventListener[*valueEvent] {
	return subscription.SubscriptionEventListener[*valueEvent]{
		Updated:               func(v *valueEvent) { r.updates <- v },
		SubscriptionError:     func(err error) { r.errors <- err },
		SubscriptionCancelled: func() { r.cancelled <- struct{}{} },
	}
}

func (r *recorder) nextUpdate(t *testing.T) *valueEvent {
	t.Helper()
	select {
	case v := <-r.updates:
		return v
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for an update")
		return nil
	}
}

func (r *recorder) nextError(t *testing.T) error {
	t.Helper()
	select {
	case err := <-r.errors:
		return err
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for a subscription error")
		return nil
	}
}

func (r *recorder) waitCancelled(t *testing.T) {
	t.Helper()
	select {
	case <-r.cancelled:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for the subscription to be cancelled")
	}
}

// expectNothing fails if the listener is called within a short window.
func (r *recorder) expectNothing(t *testing.T) {
	t.Helper()
	select {
	case v := <-r.updates:
		t.Fatalf("unexpected update %v", v)
	case err := <-r.errors:
		t.Fatalf("unexpected subscription error %v", err)
	case <-r.cancelled:
		t.Fatal("unexpected subscription cancellation")
	case <-time.After(50 * time.Millisecond):
	}
}

func newTestManager(factory *fakeClientFactory, idleTimeout time.Duration) *SubscriptionManager[*valueEvent] {
	return NewSubscriptionManager(factory.newClient, DecodeJSON[valueEvent], idleTimeout)
}

type fakeClientFactory struct {
	mu      sync.Mutex
	calls   map[string]int
	clients map[string][]*fakeManagedSSEClient
}

func newFakeClientFactory() *fakeClientFactory {
	return &fakeClientFactory{
		calls:   make(map[string]int),
		clients: make(map[string][]*fakeManagedSSEClient),
	}
}

func (f *fakeClientFactory) newClient(id string) SSEClient {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls[id]++
	c := newFakeManagedSSEClient()
	f.clients[id] = append(f.clients[id], c)
	return c
}

func (f *fakeClientFactory) callCount(id string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls[id]
}

// nth waits for the n-th (1-indexed) fake client created for id and for it to
// have subscribed. Subscription setup runs in a goroutine, so
// tests must wait rather than assume the client exists immediately after
// AddListener returns.
func (f *fakeClientFactory) nth(t *testing.T, id string, n int) *fakeManagedSSEClient {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for {
		f.mu.Lock()
		clients := f.clients[id]
		if len(clients) >= n {
			c := clients[n-1]
			f.mu.Unlock()
			select {
			case <-c.ready:
			case <-time.After(2 * time.Second):
				t.Fatalf("timed out waiting for subscription for id %q (instance %d)", id, n)
			}
			return c
		}
		f.mu.Unlock()
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for client #%d for id %q", n, id)
		case <-time.After(time.Millisecond):
		}
	}
}

// latest waits for the first fake client created for id.
func (f *fakeClientFactory) latest(t *testing.T, id string) *fakeManagedSSEClient {
	return f.nth(t, id, 1)
}

func TestSubscriptionManager_SingleListenerReceivesEvents(t *testing.T) {
	factory := newFakeClientFactory()
	mgr := newTestManager(factory, 0)

	r := newRecorder()
	defer mgr.AddListener("pool-1", r.listener())()

	factory.latest(t, "pool-1").emit(StreamEvent{Data: `{"value":7}`})

	if v := r.nextUpdate(t); v.Value != 7 {
		t.Errorf("got %d, want 7", v.Value)
	}
}

func TestSubscriptionManager_MultipleListenersShareOneConnection(t *testing.T) {
	factory := newFakeClientFactory()
	mgr := newTestManager(factory, 0)

	r1, r2 := newRecorder(), newRecorder()
	defer mgr.AddListener("pool-1", r1.listener())()
	defer mgr.AddListener("pool-1", r2.listener())()

	client := factory.latest(t, "pool-1")
	if got := factory.callCount("pool-1"); got != 1 {
		t.Fatalf("expected exactly 1 SSE connection for 2 listeners on the same id, got %d", got)
	}
	client.emit(StreamEvent{Data: `{"value":9}`})

	for i, r := range []*recorder{r1, r2} {
		if v := r.nextUpdate(t); v.Value != 9 {
			t.Errorf("listener %d got %d, want 9", i+1, v.Value)
		}
	}
}

// Each listener receives its own copy, so one changing an update cannot change what another sees.
func TestSubscriptionManager_EachListenerReceivesItsOwnCopy(t *testing.T) {
	factory := newFakeClientFactory()
	mgr := newTestManager(factory, 0)

	r1, r2 := newRecorder(), newRecorder()
	defer mgr.AddListener("pool-1", r1.listener())()
	defer mgr.AddListener("pool-1", r2.listener())()

	factory.latest(t, "pool-1").emit(StreamEvent{Data: `{"value":9}`})

	v1, v2 := r1.nextUpdate(t), r2.nextUpdate(t)
	if v1 == v2 {
		t.Fatal("both listeners received the same value")
	}
}

func TestSubscriptionManager_DifferentIdsAreIndependent(t *testing.T) {
	factory := newFakeClientFactory()
	mgr := newTestManager(factory, 0)

	r1, r2 := newRecorder(), newRecorder()
	defer mgr.AddListener("pool-1", r1.listener())()
	defer mgr.AddListener("pool-2", r2.listener())()

	factory.latest(t, "pool-1").emit(StreamEvent{Data: `{"value":1}`})

	if v := r1.nextUpdate(t); v.Value != 1 {
		t.Errorf("got %d, want 1", v.Value)
	}
	r2.expectNothing(t)
}

// The platform closes the stream after the entity is finalised. That is reported,
// rather than updates simply stopping.
func TestSubscriptionManager_StreamClosedByServerIsReportedAsCancelled(t *testing.T) {
	factory := newFakeClientFactory()
	mgr := newTestManager(factory, 0)

	r := newRecorder()
	defer mgr.AddListener("pool-1", r.listener())()

	client := factory.latest(t, "pool-1")
	client.emit(StreamEvent{Type: EventEntityFinalised, Data: `{"value":3}`})
	client.end(fmt.Errorf("stream closed: %w", io.EOF))

	if v := r.nextUpdate(t); v.Value != 3 {
		t.Errorf("got %d, want the finalised entity 3", v.Value)
	}
	r.waitCancelled(t)
	r.expectNothing(t)
}

// The silent failure this replaces: a stream that breaks is reported to SubscriptionError, and a
// failed subscription is not also reported cancelled.
func TestSubscriptionManager_StreamFailureIsReportedAsError(t *testing.T) {
	factory := newFakeClientFactory()
	mgr := newTestManager(factory, 0)

	r := newRecorder()
	defer mgr.AddListener("pool-1", r.listener())()

	failure := errors.New("connection reset")
	factory.latest(t, "pool-1").end(failure)

	if err := r.nextError(t); !errors.Is(err, failure) {
		t.Errorf("got %v, want the stream's failure", err)
	}
	r.expectNothing(t)
}

// A listener added after a subscription ends opens a new one, rather than joining a stream that
// will never deliver again.
func TestSubscriptionManager_ListenerAddedAfterTheStreamEndsOpensANewConnection(t *testing.T) {
	factory := newFakeClientFactory()
	mgr := newTestManager(factory, 0)

	r1 := newRecorder()
	defer mgr.AddListener("pool-1", r1.listener())()
	factory.latest(t, "pool-1").end(io.EOF)
	r1.waitCancelled(t)

	r2 := newRecorder()
	defer mgr.AddListener("pool-1", r2.listener())()
	client := factory.nth(t, "pool-1", 2)
	client.emit(StreamEvent{Data: `{"value":5}`})

	if v := r2.nextUpdate(t); v.Value != 5 {
		t.Errorf("got %d, want 5", v.Value)
	}
	r1.expectNothing(t)
}

// An update that cannot be decoded is reported to SubscriptionError and skipped, and the
// subscription continues.
func TestSubscriptionManager_UndecodableEventIsReportedAndTheStreamContinues(t *testing.T) {
	factory := newFakeClientFactory()
	mgr := newTestManager(factory, 0)

	r := newRecorder()
	defer mgr.AddListener("pool-1", r.listener())()

	client := factory.latest(t, "pool-1")
	client.emit(StreamEvent{Type: EventEntityUpdated, Data: `not json`})
	client.emit(StreamEvent{Data: `{"value":11}`})

	if err := r.nextError(t); !errors.Is(err, ErrUndecodableUpdate) || !strings.Contains(err.Error(), `decoding "entityUpdated" event`) {
		t.Errorf("got %v, want ErrUndecodableUpdate for the entityUpdated event", err)
	}
	if v := r.nextUpdate(t); v.Value != 11 {
		t.Errorf("got %d, want 11", v.Value)
	}
}

func TestSubscriptionManager_HeartbeatsAreNotDelivered(t *testing.T) {
	factory := newFakeClientFactory()
	mgr := newTestManager(factory, 0)

	r := newRecorder()
	defer mgr.AddListener("pool-1", r.listener())()

	client := factory.latest(t, "pool-1")
	client.emit(StreamEvent{})
	client.emit(StreamEvent{Type: EventEntityStable})

	r.expectNothing(t)
}

// A stream that stays open but sends nothing, not even the heartbeat, is reported rather than
// waited on forever.
func TestSubscriptionManager_IdleStreamIsReportedAsError(t *testing.T) {
	factory := newFakeClientFactory()
	mgr := newTestManager(factory, 200*time.Millisecond)

	r := newRecorder()
	defer mgr.AddListener("pool-1", r.listener())()
	factory.latest(t, "pool-1")

	if err := r.nextError(t); !errors.Is(err, ErrStreamIdle) {
		t.Errorf("got %v, want ErrStreamIdle", err)
	}
}

func TestSubscriptionManager_HeartbeatsKeepAnIdleTimeoutFromFiring(t *testing.T) {
	factory := newFakeClientFactory()
	mgr := newTestManager(factory, 300*time.Millisecond)

	r := newRecorder()
	defer mgr.AddListener("pool-1", r.listener())()
	client := factory.latest(t, "pool-1")

	for range 5 {
		time.Sleep(150 * time.Millisecond)
		client.emit(StreamEvent{Type: EventEntityStable})
	}
	r.expectNothing(t)
}

// A listener that takes longer than the idle timeout to handle an update does not end the
// subscription, as the heartbeats sent meanwhile are read once it returns.
func TestSubscriptionManager_SlowListenerDoesNotTripTheIdleTimeout(t *testing.T) {
	factory := newFakeClientFactory()
	mgr := newTestManager(factory, 200*time.Millisecond)

	r := newRecorder()
	listener := r.listener()
	recordUpdate := listener.Updated
	listener.Updated = func(v *valueEvent) {
		time.Sleep(500 * time.Millisecond)
		recordUpdate(v)
	}
	defer mgr.AddListener("pool-1", listener)()
	client := factory.latest(t, "pool-1")

	client.emit(StreamEvent{Data: `{"value":1}`})
	client.emit(StreamEvent{Type: EventEntityStable})

	if got := r.nextUpdate(t); got.Value != 1 {
		t.Errorf("got update %v, want value 1", got.Value)
	}
	r.expectNothing(t)
}

func TestSubscriptionManager_NilListenerFunctionsAreNotCalled(t *testing.T) {
	factory := newFakeClientFactory()
	mgr := newTestManager(factory, 0)

	defer mgr.AddListener("pool-1", subscription.SubscriptionEventListener[*valueEvent]{})()
	client := factory.latest(t, "pool-1")

	client.emit(StreamEvent{Data: `{"value":1}`})
	client.emit(StreamEvent{Data: `not json`})
	client.end(io.EOF)

	select {
	case <-client.stopped:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for the stream to end")
	}
}

func TestSubscriptionManager_RemovingLastListenerTearsDownWithoutNotifyingAndReconnects(t *testing.T) {
	factory := newFakeClientFactory()
	mgr := newTestManager(factory, 0)

	r := newRecorder()
	unsub := mgr.AddListener("pool-1", r.listener())
	client := factory.latest(t, "pool-1")

	unsub()

	select {
	case <-client.stopped:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for the subscription to be cancelled after removing the last listener")
	}
	r.expectNothing(t)

	// Re-adding a listener for the same id must open a fresh connection.
	defer mgr.AddListener("pool-1", newRecorder().listener())()
	factory.nth(t, "pool-1", 2)
	if got := factory.callCount("pool-1"); got != 2 {
		t.Errorf("expected a second SSE connection to be opened, got %d calls", got)
	}
}

func TestSubscriptionManager_RemovingOneOfSeveralListenersKeepsConnectionOpen(t *testing.T) {
	factory := newFakeClientFactory()
	mgr := newTestManager(factory, 0)

	r1, r2 := newRecorder(), newRecorder()
	unsub1 := mgr.AddListener("pool-1", r1.listener())
	defer mgr.AddListener("pool-1", r2.listener())()

	client := factory.latest(t, "pool-1")
	unsub1()

	client.emit(StreamEvent{Data: `{"value":3}`})

	if v := r2.nextUpdate(t); v.Value != 3 {
		t.Errorf("got %d, want 3", v.Value)
	}
	r1.expectNothing(t)
	if got := factory.callCount("pool-1"); got != 1 {
		t.Errorf("expected the connection to stay open with a listener still registered, got %d calls", got)
	}
}

func TestSubscriptionManager_CloseTearsDownAllSubscriptionsWithoutNotifying(t *testing.T) {
	factory := newFakeClientFactory()
	mgr := newTestManager(factory, 0)

	r1, r2 := newRecorder(), newRecorder()
	mgr.AddListener("pool-1", r1.listener())
	mgr.AddListener("pool-2", r2.listener())
	client1 := factory.latest(t, "pool-1")
	client2 := factory.latest(t, "pool-2")

	mgr.Close()

	for _, c := range []*fakeManagedSSEClient{client1, client2} {
		select {
		case <-c.stopped:
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for the subscription to be cancelled after Close")
		}
	}
	r1.expectNothing(t)
	r2.expectNothing(t)

	mgr.mu.Lock()
	remaining := len(mgr.subscriptions)
	mgr.mu.Unlock()
	if remaining != 0 {
		t.Errorf("expected no subscriptions to remain after Close, got %d", remaining)
	}
}

// The real stream client against the stub platform: closing the stream after the entity is
// finalised arrives as SubscriptionCancelled.
func TestSubscriptionManager_RealStreamClosedAfterFinalisedIsReportedAsCancelled(t *testing.T) {
	server := newSSEServer(t, func(w *eventWriter, r *http.Request) {
		w.send(EventEntityUpdated, `{"value":1}`)
		w.send(EventEntityFinalised, `{"value":2}`)
		// return, closing the stream as the platform does
	})
	hc := NewHTTPClient(&HTTPConfig{BaseURL: server.URL, APIKey: "k", APISecret: "s"})
	mgr := NewSubscriptionManager(func(string) SSEClient { return hc.NewSSEClient("/updates") }, DecodeJSON[valueEvent], 0)

	r := newRecorder()
	defer mgr.AddListener("pool-1", r.listener())()

	if v := r.nextUpdate(t); v.Value != 1 {
		t.Errorf("got %d, want 1", v.Value)
	}
	if v := r.nextUpdate(t); v.Value != 2 {
		t.Errorf("got %d, want the finalised entity 2", v.Value)
	}
	r.waitCancelled(t)
}

// The real stream client against a server that rejects the request: reported, not silent.
func TestSubscriptionManager_RealStreamRejectedIsReportedAsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(server.Close)
	hc := NewHTTPClient(&HTTPConfig{BaseURL: server.URL, APIKey: "k", APISecret: "s"})
	mgr := NewSubscriptionManager(func(string) SSEClient { return hc.NewSSEClient("/updates") }, DecodeJSON[valueEvent], 0)

	r := newRecorder()
	defer mgr.AddListener("pool-1", r.listener())()

	if err := r.nextError(t); !strings.Contains(err.Error(), "update stream failed") {
		t.Errorf("got %v, want an update stream failure", err)
	}
	r.expectNothing(t)
}
