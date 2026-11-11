package internal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/yellowdogai/yellowdog-sdk-go/subscription"
)

// ErrUndecodableUpdate is reported to a listener's SubscriptionError for an update that cannot be
// decoded. The update is skipped and the subscription continues, unlike every other error
// reported there.
var ErrUndecodableUpdate = errors.New("update skipped")

// SubscriptionManager manages persistent SSE subscriptions with fan-out.
// Multiple listeners can register for the same resource ID; a single SSE connection
// is kept open per ID and torn down automatically when the last listener unsubscribes,
// or when the stream ends, after which the next listener opens a new one.
type SubscriptionManager[T any] struct {
	mu            sync.Mutex
	subscriptions map[string]*activeSubscription[T]
	newClient     func(id string) SSEClient
	decode        func(data []byte) (T, error)
	idleTimeout   time.Duration
}

type activeSubscription[T any] struct {
	listeners []*listenerEntry[T]
	cancel    context.CancelFunc
}

type listenerEntry[T any] struct {
	listener subscription.SubscriptionEventListener[T]
}

// NewSubscriptionManager creates a new manager. newClient is called to open the SSE stream
// for a given resource ID and must return a client pointed at the resource's update endpoint.
// decode turns an event's data into the value listeners receive, and is called once per listener
// so that each receives its own copy. A stream that sends nothing, not even a heartbeat, for
// idleTimeout fails with ErrStreamIdle; zero disables the check.
func NewSubscriptionManager[T any](newClient func(id string) SSEClient, decode func(data []byte) (T, error), idleTimeout time.Duration) *SubscriptionManager[T] {
	return &SubscriptionManager[T]{
		subscriptions: make(map[string]*activeSubscription[T]),
		newClient:     newClient,
		decode:        decode,
		idleTimeout:   idleTimeout,
	}
}

// DecodeJSON decodes an event's data as JSON into a new T.
func DecodeJSON[T any](data []byte) (*T, error) {
	var value T
	if err := json.Unmarshal(data, &value); err != nil {
		return nil, err
	}
	return &value, nil
}

// AddListener subscribes listener to live updates for the resource with the given ID.
// Returns an unsubscribe function; call it to remove the listener.
// If there is no open subscription for the ID a new SSE connection is opened.
func (m *SubscriptionManager[T]) AddListener(id string, listener subscription.SubscriptionEventListener[T]) func() {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry := &listenerEntry[T]{listener: listener}

	sub, ok := m.subscriptions[id]
	if !ok {
		ctx, cancel := context.WithCancel(context.Background())
		sub = &activeSubscription[T]{cancel: cancel}
		m.subscriptions[id] = sub
		go m.runSubscription(ctx, id, sub)
	}
	sub.listeners = append(sub.listeners, entry)

	return func() { m.removeListener(id, sub, entry) }
}

// Close tears down all active SSE connections and removes all listeners, without notifying them.
func (m *SubscriptionManager[T]) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, sub := range m.subscriptions {
		sub.cancel()
		delete(m.subscriptions, id)
	}
}

func (m *SubscriptionManager[T]) removeListener(id string, sub *activeSubscription[T], entry *listenerEntry[T]) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, e := range sub.listeners {
		if e == entry {
			sub.listeners = append(sub.listeners[:i], sub.listeners[i+1:]...)
			break
		}
	}
	if len(sub.listeners) == 0 {
		sub.cancel()
		if m.subscriptions[id] == sub {
			delete(m.subscriptions, id)
		}
	}
}

// snapshot returns the subscription's current listeners, so that they can be called without the
// lock held: a listener may unsubscribe from within a call.
func (m *SubscriptionManager[T]) snapshot(sub *activeSubscription[T]) []*listenerEntry[T] {
	m.mu.Lock()
	defer m.mu.Unlock()
	listeners := make([]*listenerEntry[T], len(sub.listeners))
	copy(listeners, sub.listeners)
	return listeners
}

// runSubscription streams updates for id until the stream ends or ctx is cancelled, which
// happens when the last listener unsubscribes or the manager is closed. An ending the listeners
// did not ask for is reported to them: an orderly close as SubscriptionCancelled, anything else
// as SubscriptionError.
func (m *SubscriptionManager[T]) runSubscription(ctx context.Context, id string, sub *activeSubscription[T]) {
	err := m.stream(ctx, id, sub)

	// The subscription is over, so a listener added from now on must open a new one rather than
	// join a stream that will never deliver again.
	m.mu.Lock()
	if m.subscriptions[id] == sub {
		delete(m.subscriptions, id)
	}
	m.mu.Unlock()

	if ctx.Err() != nil {
		return // torn down on request; nobody is left listening, or they asked to stop
	}
	listeners := m.snapshot(sub)
	if err == nil || errors.Is(err, io.EOF) {
		for _, l := range listeners {
			if l.listener.SubscriptionCancelled != nil {
				l.listener.SubscriptionCancelled()
			}
		}
		return
	}
	for _, l := range listeners {
		if l.listener.SubscriptionError != nil {
			l.listener.SubscriptionError(err)
		}
	}
}

func (m *SubscriptionManager[T]) stream(ctx context.Context, id string, sub *activeSubscription[T]) error {
	// The stream runs under its own context so that the idle watchdog can end it without ending
	// the subscription's.
	streamCtx, stopStream := context.WithCancel(ctx)
	defer stopStream()

	idle := newIdleTimer(m.idleTimeout)
	defer idle.stop()
	var idleTimeoutErr error
	var idleTimeoutErrMutex sync.Mutex
	watchdogDone := make(chan struct{})
	defer close(watchdogDone)
	go func() {
		select {
		case <-idle.C():
			idleTimeoutErrMutex.Lock()
			idleTimeoutErr = fmt.Errorf("%w after %v", ErrStreamIdle, m.idleTimeout)
			idleTimeoutErrMutex.Unlock()
			stopStream()
		case <-watchdogDone:
		}
	}()

	err := m.newClient(id).Stream(streamCtx, func(event StreamEvent) {
		if event.Type == EventEntityStable || event.Data == "" {
			idle.reset()
			return // heartbeat, no entity to deliver
		}
		// The timer is paused while the listeners run, as nothing reads the stream meanwhile, so
		// that a slow listener is not mistaken for a silent stream.
		idle.stop()
		m.deliver(sub, event)
		idle.reset()
	})

	idleTimeoutErrMutex.Lock()
	defer idleTimeoutErrMutex.Unlock()
	if idleTimeoutErr != nil {
		return idleTimeoutErr
	}
	if err != nil && !errors.Is(err, io.EOF) && ctx.Err() == nil {
		return fmt.Errorf("update stream failed: %w", err)
	}
	return err
}

// deliver decodes the event once per listener, so that each receives its own copy, and reports
// an event that cannot be decoded to each listener while the stream continues.
func (m *SubscriptionManager[T]) deliver(sub *activeSubscription[T], event StreamEvent) {
	eventData := []byte(event.Data)
	for _, entry := range m.snapshot(sub) {
		update, decodeErr := m.decode(eventData)
		if decodeErr != nil {
			if entry.listener.SubscriptionError != nil {
				entry.listener.SubscriptionError(fmt.Errorf("%w: decoding %q event: %w", ErrUndecodableUpdate, event.Type, decodeErr))
			}
			continue
		}
		if entry.listener.Updated != nil {
			entry.listener.Updated(update)
		}
	}
}
