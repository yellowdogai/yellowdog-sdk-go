// Package subscription defines the listener the clients' Add…Listener methods take to receive
// live updates to an entity.
package subscription

// SubscriptionEventListener receives the events of a subscription to an entity's updates.
//
// All listeners on one entity share a subscription. Its calls are made one at a time, in order,
// on the goroutine reading the subscription's stream, so a call that blocks holds back the rest.
// Each listener receives its own copy of every update.
type SubscriptionEventListener[T any] struct {
	// Updated is called with each update. The listener that opens the subscription first receives
	// the entity's current state; a listener that joins an open subscription receives only changes
	// from then on. The update that reports the entity finalised is delivered like any other, and
	// is the last.
	Updated func(T)

	// SubscriptionError is called when the subscription fails: its stream cannot be opened,
	// breaks, or goes silent for longer than the platform's heartbeat allows
	// (yellowdog.ErrStreamIdle). No further calls are made after it, and adding a listener again
	// opens a new subscription. It is also called, with an error matching
	// yellowdog.ErrUndecodableUpdate, when one update cannot be decoded; that update is skipped
	// and the subscription continues, so the listener must not be added again.
	SubscriptionError func(error)

	// SubscriptionCancelled is called when the platform ends the subscription, which it does
	// after reporting the entity finalised. No further calls are made after it, and adding a
	// listener again opens a new subscription.
	SubscriptionCancelled func()
}
