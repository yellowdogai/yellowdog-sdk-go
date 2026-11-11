// Package usage provides access to the YellowDog Usage and Allowances API.
//
// Allowances cap compute consumption (measured in instance-hours) and are
// enforced by the platform. Three allowance scopes are available:
//
//   - [model.AccountAllowance] — caps total usage across the entire account
//   - [model.SourcesAllowance] — caps usage for a specific cloud provider,
//     region, or set of instance types
//   - [model.RequirementAllowance] — caps usage for a single compute requirement
//
// # Creating allowances
//
// Use the constructors in the model package (e.g. [model.NewAccountAllowance])
// to build allowance values, then submit them via [AllowancesClient.AddAllowance]:
//
//	allowance := model.NewAccountAllowance()
//	allowance.AllowedHours = 500
//	allowance.ResetType = model.AllowanceResetTypeMonths
//	allowance.LimitEnforcement = model.AllowanceLimitEnforcementSoft
//	allowance.EffectiveFrom = time.Now()
//	created, _ := client.Allowances.AddAllowance(ctx, allowance)
//
// # Polymorphic return types
//
// [AllowancesClient.GetAllowanceById] and [AllowancesClient.GetAllowances]
// return the [model.Allowance] interface. Type-assert to the concrete type
// to access scope-specific fields:
//
//	a, _ := client.Allowances.GetAllowanceById(ctx, id)
//	if acct, ok := a.(*model.AccountAllowance); ok {
//	    if remaining, ok := acct.RemainingHours.Get(); ok {
//	        fmt.Println(remaining)
//	    }
//	}
//
// # Boosting
//
// Allowances can be temporarily extended with [AllowancesClient.BoostAllowance]
// or [AllowancesClient.BoostAllowanceById]. Boost hours are one-off additions
// that do not affect the base AllowedHours and are reset on the next cycle.
package usage
