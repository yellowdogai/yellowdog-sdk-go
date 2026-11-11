package scheduler

import (
	"context"
	"time"

	"github.com/yellowdogai/yellowdog-sdk-go/internal"
	"github.com/yellowdogai/yellowdog-sdk-go/model"
	"github.com/yellowdogai/yellowdog-sdk-go/optional"
)

// streamIdleTimeout is three times the platform's 15s update interval for work
// requirements and worker pools, so a stalled stream is detected without a single
// missed heartbeat ending the wait.
const streamIdleTimeout = 45 * time.Second

type WorkRequirementPredicate func(*model.WorkRequirement) bool

type WorkRequirementHelper struct {
	client      *WorkClient
	requirement *model.WorkRequirement
}

func NewWorkRequirementHelper(client *WorkClient, requirement *model.WorkRequirement) *WorkRequirementHelper {
	return &WorkRequirementHelper{
		client:      client,
		requirement: requirement,
	}
}

func (h *WorkRequirementHelper) WaitForStatus(ctx context.Context, status model.WorkRequirementStatus) (*model.WorkRequirement, error) {
	return h.WaitUntil(ctx, func(req *model.WorkRequirement) bool {
		return req.Status == optional.Of(status)
	})
}

// WaitUntil blocks until the predicate returns true or the context is cancelled.
// Updates are streamed from /work/requirements/{id}/updates. A streaming failure
// is returned rather than retried by polling; a caller who wants to poll owns
// that loop.
func (h *WorkRequirementHelper) WaitUntil(ctx context.Context, predicate WorkRequirementPredicate) (*model.WorkRequirement, error) {
	sseClient := h.client.proxy.newWorkRequirementUpdatesSSEClient(h.requirement.Id.Or(""))
	waiter := internal.NewSSEWaiter[model.WorkRequirement](sseClient).
		WithPrimer(func(ctx context.Context) (*model.WorkRequirement, error) {
			return h.client.GetWorkRequirementById(ctx, h.requirement.Id.Or(""))
		}).
		WithIdleTimeout(streamIdleTimeout)
	return waiter.WaitUntil(ctx, func(r *model.WorkRequirement) bool {
		return predicate(r)
	})
}
