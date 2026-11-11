package internal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"
)

// Event types emitted by the platform's entity update streams.
const (
	// EventEntityUpdated carries the entity, and is sent whenever it changes.
	// The platform always sends one immediately on subscribe.
	EventEntityUpdated = "entityUpdated"
	// EventEntityStable carries no data, and is sent once per polling interval
	// while the entity is unchanged. It is the stream's heartbeat.
	EventEntityStable = "entityStable"
	// EventEntityFinalised carries the entity and is the last event on the
	// stream: the platform closes the connection immediately afterwards.
	EventEntityFinalised = "entityFinalised"
)

var (
	// ErrEntityFinalised is returned when the stream reports the entity has
	// reached a terminal state without the wait condition ever holding. Waiting
	// longer cannot help, because no further events will be sent.
	ErrEntityFinalised = errors.New("entity reached a terminal state before the wait condition was met")

	// ErrStreamEnded is returned when the update stream closes without having
	// reported the entity as finalised.
	ErrStreamEnded = errors.New("update stream ended before the wait condition was met")

	// ErrStreamIdle is returned when no event arrives within the idle timeout.
	// The platform heartbeats every polling interval, so silence for longer
	// than a few intervals means the stream is no longer live.
	ErrStreamIdle = errors.New("update stream stopped sending events")
)

// Fetcher reads the current state of an entity, used to prime a wait.
type Fetcher[T any] func(context.Context) (*T, error)

// SSEWaiter blocks until a predicate holds for an entity reported on its
// update stream.
type SSEWaiter[T any] struct {
	client      SSEClient
	primer      Fetcher[T]
	idleTimeout time.Duration
}

func NewSSEWaiter[T any](client SSEClient) *SSEWaiter[T] {
	return &SSEWaiter[T]{client: client}
}

// WithPrimer supplies a fetcher read once, concurrently with opening the
// stream, and evaluated against the predicate like any streamed update. The
// platform already sends the current state as its first event, so this is
// insurance against a stream that does not.
func (w *SSEWaiter[T]) WithPrimer(primer Fetcher[T]) *SSEWaiter[T] {
	w.primer = primer
	return w
}

// WithIdleTimeout fails the wait if no event arrives within d. Pass a multiple
// of the stream's server-side polling interval; zero disables the check.
func (w *SSEWaiter[T]) WithIdleTimeout(d time.Duration) *SSEWaiter[T] {
	w.idleTimeout = d
	return w
}

func (w *SSEWaiter[T]) WaitUntil(ctx context.Context, predicate func(*T) bool) (*T, error) {
	// A child context, cancelled on every return path, is what stops the
	// subscription. Without it the connection outlives the wait.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	events := make(chan StreamEvent)
	subEnded := make(chan error, 1)
	primed := make(chan *T, 1)
	primeFailed := make(chan error, 1)

	go func() {
		subEnded <- w.client.Stream(ctx, func(event StreamEvent) {
			// Never block past cancellation: the handler runs on the read loop.
			select {
			case events <- event:
			case <-ctx.Done():
			}
		})
	}()

	if w.primer != nil {
		go func() {
			value, err := w.primer(ctx)
			if err != nil {
				select {
				case primeFailed <- err:
				case <-ctx.Done():
				}
				return
			}
			select {
			case primed <- value:
			case <-ctx.Done():
			}
		}()
	}

	idle := newIdleTimer(w.idleTimeout)
	defer idle.stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()

		case err := <-primeFailed:
			return nil, fmt.Errorf("reading current state: %w", err)

		case value := <-primed:
			if predicate(value) {
				return value, nil
			}

		case err := <-subEnded:
			// Stream returns only once the stream is finished. An orderly close
			// by the server surfaces as a wrapped io.EOF; anything else is a
			// genuine failure worth reporting verbatim.
			if err == nil || errors.Is(err, io.EOF) {
				return nil, ErrStreamEnded
			}
			return nil, fmt.Errorf("update stream failed: %w", err)

		case <-idle.C():
			return nil, fmt.Errorf("%w after %v", ErrStreamIdle, w.idleTimeout)

		case event := <-events:
			idle.reset()

			switch event.Type {
			case EventEntityStable:
				// Heartbeat, carries no entity. Resetting the idle timer above
				// is the whole of its purpose.
				continue
			case EventEntityFinalised:
				value, err := decodeEvent[T](event)
				if err != nil {
					return nil, err
				}
				if value != nil && predicate(value) {
					return value, nil
				}
				return nil, ErrEntityFinalised
			default:
				// entityUpdated, and any event type added later.
				value, err := decodeEvent[T](event)
				if err != nil {
					return nil, err
				}
				if value != nil && predicate(value) {
					return value, nil
				}
			}
		}
	}
}

// decodeEvent returns nil without error for an event carrying no data.
func decodeEvent[T any](event StreamEvent) (*T, error) {
	if event.Data == "" {
		return nil, nil
	}
	var value T
	if err := json.Unmarshal([]byte(event.Data), &value); err != nil {
		return nil, fmt.Errorf("decoding %q event: %w", event.Type, err)
	}
	return &value, nil
}

// idleTimer wraps an optional timer so the wait loop can select on it
// unconditionally; a zero duration yields a channel that never fires.
type idleTimer struct {
	d     time.Duration
	timer *time.Timer
}

func newIdleTimer(d time.Duration) *idleTimer {
	t := &idleTimer{d: d}
	if d > 0 {
		t.timer = time.NewTimer(d)
	}
	return t
}

func (t *idleTimer) C() <-chan time.Time {
	if t.timer == nil {
		return nil // a nil channel blocks forever, disabling the case
	}
	return t.timer.C
}

func (t *idleTimer) reset() {
	if t.timer != nil {
		t.timer.Reset(t.d)
	}
}

func (t *idleTimer) stop() {
	if t.timer != nil {
		t.timer.Stop()
	}
}
