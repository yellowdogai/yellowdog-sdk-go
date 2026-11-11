package compute

import (
	"context"
	"sync"

	"github.com/yellowdogai/yellowdog-sdk-go/internal"
	"github.com/yellowdogai/yellowdog-sdk-go/model"
	"github.com/yellowdogai/yellowdog-sdk-go/subscription"
)

var computeReqSubscriptions sync.Map // *ComputeClient -> *internal.SubscriptionManager[*model.ComputeRequirement]

func getComputeReqSubscriptions(c *ComputeClient) *internal.SubscriptionManager[*model.ComputeRequirement] {
	if v, ok := computeReqSubscriptions.Load(c); ok {
		return v.(*internal.SubscriptionManager[*model.ComputeRequirement])
	}
	mgr := internal.NewSubscriptionManager(func(id string) internal.SSEClient {
		return c.proxy.newRequirementUpdatesSSEClient(id)
	}, internal.DecodeJSON[model.ComputeRequirement], streamIdleTimeout)
	v, _ := computeReqSubscriptions.LoadOrStore(c, mgr)
	return v.(*internal.SubscriptionManager[*model.ComputeRequirement])
}

// Close ends the client's compute requirement subscriptions without notifying their listeners.
// The client remains usable, and a listener added afterwards opens a new subscription.
func (c *ComputeClient) Close() {
	if v, ok := computeReqSubscriptions.LoadAndDelete(c); ok {
		v.(*internal.SubscriptionManager[*model.ComputeRequirement]).Close()
	}
}

func (c *ComputeClient) Helper(req *model.ComputeRequirement) *RequirementHelper {
	return NewRequirementHelper(c, req)
}

func (c *ComputeClient) HelperById(ctx context.Context, id string) (*RequirementHelper, error) {
	req, err := c.GetComputeRequirementById(ctx, id)
	if err != nil {
		return nil, err
	}
	return NewRequirementHelper(c, req), nil
}

// AddComputeRequirementListener subscribes listener to live updates to the compute requirement,
// as subscription.SubscriptionEventListener describes.
// A failed stream is reported to SubscriptionError and the platform ending it to SubscriptionCancelled,
// rather than updates simply stopping. Listeners on one requirement share a connection.
// Returns an unsubscribe function; the connection is closed when the last listener unsubscribes.
func (c *ComputeClient) AddComputeRequirementListener(requirementId string, listener subscription.SubscriptionEventListener[*model.ComputeRequirement]) func() {
	return getComputeReqSubscriptions(c).AddListener(requirementId, listener)
}
