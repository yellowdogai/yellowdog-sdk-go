package compute_test

import (
	"context"
	"errors"
	"fmt"
	"log"

	yellowdog "github.com/yellowdogai/yellowdog-sdk-go"
	"github.com/yellowdogai/yellowdog-sdk-go/compute"
	"github.com/yellowdogai/yellowdog-sdk-go/model"
	"github.com/yellowdogai/yellowdog-sdk-go/optional"
	"github.com/yellowdogai/yellowdog-sdk-go/subscription"
)

// Example demonstrates provisioning compute end to end using the Simulator
// provider, which needs no cloud credentials. Provisioning is a three-step
// build-up: a source template describes where instances come from, a
// requirement template describes how many and with which strategy, and
// ProvisionComputeRequirementTemplate turns the template into a live
// requirement.
func Example() {
	ctx := context.Background()

	config := yellowdog.DefaultConfig()
	config.APIKey = "your-api-key"
	config.APISecret = "your-api-secret"

	client, err := yellowdog.NewClient(config)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = client.Close() }()

	simulatorSource := &model.SimulatorComputeSource{}
	simulatorSource.Name = "test-source"
	simulatorSource.Region = "test-region"
	simulatorSource.InstanceType = "t3.micro"
	simulatorSource.ImageId = "test-image"

	source, err := client.Compute.AddComputeSourceTemplate(ctx, &model.ComputeSourceTemplate{
		Source: simulatorSource,
	})
	if err != nil {
		log.Fatal(err)
	}

	template := &model.ComputeRequirementStaticTemplate{}
	template.Name = "test-template"
	template.StrategyType = model.SingleSourceStrategyType
	template.Sources = append(template.Sources, model.ComputeSourceUsage{
		SourceTemplateId: source.Id.Or(""),
	})

	provisionedTemplate, err := client.Compute.AddComputeRequirementTemplate(ctx, template)
	if err != nil {
		log.Fatal(err)
	}

	requirement, err := client.Compute.ProvisionComputeRequirementTemplate(ctx, &compute.ComputeRequirementTemplateUsage{
		RequirementName:      "test-requirement",
		RequirementNamespace: optional.Of("my-namespace"),
		TemplateId:           provisionedTemplate.GetId(),
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Created compute requirement: %s (ID: %s)\n", requirement.Name, requirement.Id.Or(""))

	// GetInstances returns the polymorphic model.Instance; narrow it to the
	// concrete type matching the source provider. A Simulator source yields
	// SimulatorInstance, an AWS source yields AwsInstance, and so on.
	instances, err := client.Compute.GetInstances(&compute.InstanceSearch{
		ComputeRequirementId: requirement.Id,
	}).ListAll(ctx)
	if err != nil {
		log.Fatal(err)
	}

	for _, inst := range instances {
		simInst, ok := inst.(*model.SimulatorInstance)
		if !ok {
			log.Fatalf("expected SimulatorInstance, got %T", inst)
		}
		fmt.Printf("Instance: %s (Status: %s, Hostname: %s)\n",
			simInst.GetId(), simInst.Status.Or(""), simInst.Hostname.Or(""))
	}

	if _, err := client.Compute.TerminateComputeRequirement(ctx, requirement); err != nil {
		log.Printf("Failed to terminate: %v", err)
	}
}

// ExampleComputeClient_GetComputeRequirements demonstrates searching with
// pagination. Every Get* search method returns a SearchClient rather than a
// slice: ListAll retrieves every slice, and Slice or Iterate retrieve fewer.
func ExampleComputeClient_GetComputeRequirements() {
	ctx := context.Background()

	config := yellowdog.DefaultConfig()
	client, _ := yellowdog.NewClient(config)
	defer func() { _ = client.Close() }()

	search := &compute.ComputeRequirementSearch{
		Namespace: optional.Of("my-namespace"),
		Statuses: []model.ComputeRequirementStatus{
			model.ComputeRequirementStatusRunning,
			model.ComputeRequirementStatusNew,
		},
	}

	requirements, err := client.Compute.GetComputeRequirements(search).ListAll(ctx)
	if err != nil {
		log.Fatal(err)
	}

	for _, req := range requirements {
		fmt.Printf("Requirement: %s (Status: %s, Instances (expected vs. target): %d/%d)\n",
			req.Name, req.Status.Or(""), req.ExpectedInstanceCount, req.TargetInstanceCount)
	}
}

// Example_provisionStrategies demonstrates the three provision strategies.
// SingleSource draws from one source; Split spreads instances across every
// source at once; Waterfall tries them in order until one satisfies the request.
func Example_provisionStrategies() {
	sourceA := &model.SimulatorComputeSource{}
	sourceA.Name = "source-a"
	sourceB := &model.SimulatorComputeSource{}
	sourceB.Name = "source-b"

	singleSource := &model.SingleSourceProvisionStrategy{}
	singleSource.Sources = model.ComputeSourceList{sourceA}

	split := &model.SplitProvisionStrategy{}
	split.Sources = model.ComputeSourceList{sourceA, sourceB}

	waterfall := &model.WaterfallProvisionStrategy{}
	waterfall.Sources = model.ComputeSourceList{sourceA, sourceB}

	fmt.Printf("SingleSource: %d source(s)\n", len(singleSource.Sources))
	fmt.Printf("Split: %d source(s)\n", len(split.Sources))
	fmt.Printf("Waterfall: %d source(s)\n", len(waterfall.Sources))

	// Output:
	// SingleSource: 1 source(s)
	// Split: 2 source(s)
	// Waterfall: 2 source(s)
}

// ExampleComputeRequirement_SupportedOperations demonstrates checking what
// operations are supported on a compute requirement. The platform decides these
// from the underlying sources, so check them before calling Stop, Start or
// Reprovision rather than assuming.
func ExampleComputeRequirement_SupportedOperations() {
	// Assume we have a compute requirement
	req := &model.ComputeRequirement{
		Name: "my-requirement",
		SupportedOperations: optional.Of(model.ComputeRequirementSupportedOperations{
			CanStopStart:   true,
			CanScaleOut:    true,
			CanReprovision: false,
		}),
	}

	ops, _ := req.SupportedOperations.Get()
	if ops.CanStopStart {
		fmt.Println("Can stop/start instances")
	}

	if ops.CanScaleOut {
		fmt.Println("Can add more instances")
	}

	if ops.CanReprovision {
		fmt.Println("Can reprovision instances")
	} else {
		fmt.Println("Cannot reprovision instances")
	}

	// Output:
	// Can stop/start instances
	// Can add more instances
	// Cannot reprovision instances
}

// ExampleComputeClient_AddComputeRequirementListener demonstrates using SSE
// to receive real-time updates about a compute requirement. The returned
// function unsubscribes; the underlying connection is shared between listeners
// on the same requirement and torn down when the last one goes away.
func ExampleComputeClient_AddComputeRequirementListener() {
	config := yellowdog.DefaultConfig()
	client, _ := yellowdog.NewClient(config)
	defer func() { _ = client.Close() }()

	// Subscribe to live updates for a compute requirement
	requirementId := "ydid:compreq:000000:abc123"
	unsubscribe := client.Compute.AddComputeRequirementListener(requirementId, subscription.SubscriptionEventListener[*model.ComputeRequirement]{
		Updated: func(req *model.ComputeRequirement) {
			fmt.Printf("Update: %s (instances: %d/%d)\n",
				req.Status.Or(""), req.ExpectedInstanceCount, req.TargetInstanceCount)
		},
		SubscriptionError: func(err error) {
			if errors.Is(err, yellowdog.ErrUndecodableUpdate) {
				// One update was skipped; the subscription continues.
				log.Printf("update skipped: %v", err)
				return
			}
			// The stream failed, or went silent; updates have stopped. Add the listener again
			// to resume.
			log.Printf("subscription failed: %v", err)
		},
		// The platform ended the subscription after the requirement was finalised.
		SubscriptionCancelled: func() { log.Print("subscription ended") },
	})
	defer unsubscribe()

	// Keep running until context cancelled or done
	// select { case <-ctx.Done(): }
}
