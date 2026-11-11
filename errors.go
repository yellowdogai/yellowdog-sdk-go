package yellowdog

import (
	"fmt"

	"github.com/yellowdogai/yellowdog-sdk-go/internal"
)

// Error is the base error type for YellowDog SDK errors.
type Error struct {
	Message    string
	StatusCode int
	Err        error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("yellowdog: %s: %v", e.Message, e.Err)
	}
	return fmt.Sprintf("yellowdog: %s", e.Message)
}

func (e *Error) Unwrap() error {
	return e.Err
}

// ConfigError represents a configuration error.
type ConfigError struct {
	Field   string
	Message string
}

func (e *ConfigError) Error() string {
	return fmt.Sprintf("config error: %s: %s", e.Field, e.Message)
}

// APIError represents an error response from the YellowDog API.
type APIError = internal.APIError

// Errors reported by the helpers that wait on an entity's update stream. Each
// means the wait cannot be satisfied by waiting longer, so they are worth
// distinguishing from a context deadline with errors.Is.
var (
	// ErrEntityFinalised means the platform reported the entity had reached a
	// terminal state without the wait condition ever holding: a compute
	// requirement that terminated while waiting for it to run, for example. No
	// further updates will be sent.
	ErrEntityFinalised = internal.ErrEntityFinalised

	// ErrStreamEnded means the update stream closed without reporting the
	// entity as finalised. Retrying the wait is reasonable.
	ErrStreamEnded = internal.ErrStreamEnded

	// ErrStreamIdle means no event arrived for several of the stream's
	// heartbeat intervals, so it is no longer live. Retrying is reasonable.
	ErrStreamIdle = internal.ErrStreamIdle
)

// ErrUndecodableUpdate is reported to a subscription.SubscriptionEventListener's
// SubscriptionError for an update that cannot be decoded. The update is skipped and the
// subscription continues; every other error reported there means the subscription has ended.
var ErrUndecodableUpdate = internal.ErrUndecodableUpdate

// ValidationError represents a validation error.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error: %s: %s", e.Field, e.Message)
}
