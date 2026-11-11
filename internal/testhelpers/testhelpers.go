// Package testhelpers holds conveniences the SDK's own tests use. They are kept out of the
// client packages, they cannot be imported from outside this module, and nothing
// outside the tests imports them.
package testhelpers

import (
	"context"

	"github.com/yellowdogai/yellowdog-sdk-go/compute"
	"github.com/yellowdogai/yellowdog-sdk-go/model"
)

// WaitForTermination waits for the compute requirement to reach TERMINATED status.
func WaitForTermination(ctx context.Context, h *compute.RequirementHelper) (*model.ComputeRequirement, error) {
	return h.WaitForStatus(ctx, model.ComputeRequirementStatusTerminated)
}
