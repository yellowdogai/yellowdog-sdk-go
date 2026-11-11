package scheduler

import (
	"context"
	"sync"

	"github.com/yellowdogai/yellowdog-sdk-go/internal"
	"github.com/yellowdogai/yellowdog-sdk-go/model"
	"github.com/yellowdogai/yellowdog-sdk-go/subscription"
)

var workReqSubscriptions sync.Map // *WorkClient -> *internal.SubscriptionManager[*model.WorkRequirement]

func getWorkReqSubscriptions(c *WorkClient) *internal.SubscriptionManager[*model.WorkRequirement] {
	if v, ok := workReqSubscriptions.Load(c); ok {
		return v.(*internal.SubscriptionManager[*model.WorkRequirement])
	}
	mgr := internal.NewSubscriptionManager(func(id string) internal.SSEClient {
		return c.proxy.newWorkRequirementUpdatesSSEClient(id)
	}, internal.DecodeJSON[model.WorkRequirement], streamIdleTimeout)
	v, _ := workReqSubscriptions.LoadOrStore(c, mgr)
	return v.(*internal.SubscriptionManager[*model.WorkRequirement])
}

// Close ends the client's work requirement subscriptions without notifying their listeners. The
// client remains usable, and a listener added afterwards opens a new subscription.
func (c *WorkClient) Close() {
	if v, ok := workReqSubscriptions.LoadAndDelete(c); ok {
		v.(*internal.SubscriptionManager[*model.WorkRequirement]).Close()
	}
}

func (c *WorkClient) Helper(req *model.WorkRequirement) *WorkRequirementHelper {
	return NewWorkRequirementHelper(c, req)
}

func (c *WorkClient) HelperById(ctx context.Context, id string) (*WorkRequirementHelper, error) {
	req, err := c.GetWorkRequirementById(ctx, id)
	if err != nil {
		return nil, err
	}
	return NewWorkRequirementHelper(c, req), nil
}

// AddWorkRequirementListener subscribes listener to live updates to the work requirement with the
// given ID, as subscription.SubscriptionEventListener describes. A failed stream is reported to
// SubscriptionError and the platform ending it to SubscriptionCancelled. Listeners on one
// requirement share a connection. Returns an unsubscribe function; the connection is closed when
// the last listener unsubscribes.
func (c *WorkClient) AddWorkRequirementListener(requirementId string, listener subscription.SubscriptionEventListener[*model.WorkRequirement]) func() {
	return getWorkReqSubscriptions(c).AddListener(requirementId, listener)
}
