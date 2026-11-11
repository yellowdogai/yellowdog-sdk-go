package compute

import (
	"context"
	"time"

	"github.com/yellowdogai/yellowdog-sdk-go/internal"
	"github.com/yellowdogai/yellowdog-sdk-go/model"
	"github.com/yellowdogai/yellowdog-sdk-go/optional"
)

// RequirementPredicate is a function type that evaluates whether a compute requirement matches a condition.
type RequirementPredicate func(*model.ComputeRequirement) bool

// RequirementHelper provides utilities for waiting on compute requirement state changes.
type RequirementHelper struct {
	client      *ComputeClient
	requirement *model.ComputeRequirement
}

// NewRequirementHelper creates a new helper for the given compute requirement.
func NewRequirementHelper(client *ComputeClient, requirement *model.ComputeRequirement) *RequirementHelper {
	return &RequirementHelper{
		client:      client,
		requirement: requirement,
	}
}

// WaitForStatus blocks until the compute requirement reaches the specified status or the context is cancelled.
// Returns the updated compute requirement when the status is reached.
func (h *RequirementHelper) WaitForStatus(ctx context.Context, status model.ComputeRequirementStatus) (*model.ComputeRequirement, error) {
	return h.WaitUntil(ctx, func(req *model.ComputeRequirement) bool {
		return req.Status == optional.Of(status)
	})
}

// streamIdleTimeout is three times the interval at which the platform sends
// requirement updates (5s), so a stalled stream is detected without a single
// missed heartbeat ending the wait.
const streamIdleTimeout = 15 * time.Second

// WaitUntil blocks until the predicate returns true or the context is cancelled.
// Updates are streamed from /compute/requirements/{id}/updates.
//
// A streaming failure is returned rather than retried by polling, so a
// misconfigured endpoint or a rejected key surfaces as an error rather than a
// slow wait. Callers who want to poll should do so themselves.
func (h *RequirementHelper) WaitUntil(ctx context.Context, predicate RequirementPredicate) (*model.ComputeRequirement, error) {
	sseClient := h.client.proxy.newRequirementUpdatesSSEClient(h.requirement.Id.Or(""))
	waiter := internal.NewSSEWaiter[model.ComputeRequirement](sseClient).
		WithPrimer(func(ctx context.Context) (*model.ComputeRequirement, error) {
			return h.client.GetComputeRequirementById(ctx, h.requirement.Id.Or(""))
		}).
		WithIdleTimeout(streamIdleTimeout)
	return waiter.WaitUntil(ctx, func(r *model.ComputeRequirement) bool {
		return predicate(r)
	})
}
