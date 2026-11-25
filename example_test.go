package yellowdog_test

import (
	"context"
	"errors"
	"fmt"
	"log"

	yellowdog "github.com/yellowdogai/yellowdog-sdk-go"
	"github.com/yellowdogai/yellowdog-sdk-go/compute"
	"github.com/yellowdogai/yellowdog-sdk-go/model"
)

// Example demonstrates creating a client and reaching a service through it.
func Example() {
	config := yellowdog.DefaultConfig(
		yellowdog.ServicesSchema{DefaultURL: "https://api.yellowdog.ai"},
		yellowdog.APIKey{ID: "your-api-key-id", Secret: "your-api-key-secret"},
		yellowdog.Product{
			Name:    "my-app",
			Version: "0.1.0",
		},
	)

	client, err := yellowdog.NewClient(config)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = client.Close() }()

	ctx := context.Background()

	// A search returns a SearchClient rather than a slice; ListAll retrieves
	// every slice of results.
	requirements, err := client.Compute.GetComputeRequirementSummaries(&compute.ComputeRequirementSummarySearch{
		Statuses: []model.ComputeRequirementStatus{
			model.ComputeRequirementStatusRunning,
		},
	}).ListAll(ctx)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d running compute requirements\n", len(requirements))
}

// ExampleAPIError demonstrates inspecting an error returned by the platform.
// APIError is matched as a pointer: Error is declared on *APIError, so passing
// errors.As a *APIError target (rather than **APIError) makes it panic.
func ExampleAPIError() {
	// In real code this is the error returned by any client call.
	err := error(&yellowdog.APIError{
		StatusCode: 404,
		Message:    "namespace not found",
	})

	var apiErr *yellowdog.APIError
	if errors.As(err, &apiErr) {
		fmt.Printf("API error %d: %s\n", apiErr.StatusCode, apiErr.Message)
	} else {
		fmt.Printf("Error: %v\n", err)
	}

	// Output:
	// API error 404: namespace not found
}
