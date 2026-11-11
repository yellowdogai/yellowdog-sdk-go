package usage_test

import (
	"context"
	"fmt"
	"log"
	"time"

	yellowdog "github.com/yellowdogai/yellowdog-sdk-go"
	"github.com/yellowdogai/yellowdog-sdk-go/model"
	"github.com/yellowdogai/yellowdog-sdk-go/optional"
	"github.com/yellowdogai/yellowdog-sdk-go/usage"
)

// ExampleAllowancesClient demonstrates the allowance lifecycle: create, list,
// read back and delete. Allowance is a polymorphic interface, so AddAllowance,
// GetAllowances and GetAllowanceById all deal in model.Allowance; reach the
// concrete fields with a type switch or a type assertion on the returned value.
func ExampleAllowancesClient() {
	config := yellowdog.DefaultConfig()
	config.APIKey = "your-api-key"
	config.APISecret = "your-api-secret"

	client, err := yellowdog.NewClient(config)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = client.Close() }()

	ctx := context.Background()

	// An account-wide allowance capping total compute hours per month.
	allowance := model.NewAccountAllowance()
	allowance.Description = optional.Of("Monthly compute budget")
	allowance.AllowedHours = 500
	allowance.ResetType = model.AllowanceResetTypeMonths
	allowance.LimitEnforcement = model.AllowanceLimitEnforcementSoft
	allowance.EffectiveFrom = time.Now()
	allowance.MonitoredStatuses = []model.InstanceStatus{model.InstanceStatusRunning}

	created, err := client.Allowances.AddAllowance(ctx, allowance)
	if err != nil {
		log.Fatal(err)
	}

	// AllowedHours is declared on each concrete allowance type rather than on
	// the interface, so a type switch is needed to read it generically.
	var allowedHours int
	switch v := created.(type) {
	case *model.AccountAllowance:
		allowedHours = v.AllowedHours
	case *model.SourcesAllowance:
		allowedHours = v.AllowedHours
	case *model.RequirementAllowance:
		allowedHours = v.AllowedHours
	}
	fmt.Printf("Created allowance: %s (allowed: %d hours)\n", created.GetId(), allowedHours)

	allowances, err := client.Allowances.GetAllowances(&usage.AllowanceSearch{}).ListAll(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Account has %d allowance(s)\n", len(allowances))

	current, err := client.Allowances.GetAllowanceById(ctx, created.GetId())
	if err != nil {
		log.Fatal(err)
	}
	if a, ok := current.(*model.AccountAllowance); ok {
		if remaining, ok := a.RemainingHours.Get(); ok {
			fmt.Printf("Remaining hours: %.1f\n", remaining)
		}
	}

	if err := client.Allowances.DeleteAllowance(ctx, created); err != nil {
		log.Printf("Failed to delete allowance: %v", err)
	}
}
