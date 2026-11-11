// Package compute provides access to the YellowDog Compute API.
//
// It covers compute requirements (groups of cloud instances), compute source
// templates (cloud provider configuration), compute requirement templates
// (reusable provisioning blueprints), and real-time instance monitoring via
// Server-Sent Events (SSE).
//
// # Concepts
//
// A [model.ComputeRequirement] represents a running group of cloud instances.
// It is provisioned from a [model.ComputeRequirementTemplate] or directly from
// a list of source usages.
//
// A [model.ComputeSourceTemplate] wraps a cloud-provider-specific source
// (e.g. [model.AwsInstancesComputeSource]) and holds the configuration needed
// to launch instances in that provider.
//
// # Provision strategies
//
// Three strategies control how instances are allocated across sources:
//
//   - [model.SingleSourceProvisionStrategy] — all instances from one source
//   - [model.SplitProvisionStrategy] — instances spread across multiple sources
//   - [model.WaterfallProvisionStrategy] — sources tried in order until one succeeds
//
// # Polymorphic instances
//
// [ComputeClient.GetInstances] returns []model.Instance (an interface). Type-assert
// to the concrete cloud type, which implements the interface with pointer receivers:
//
//	instances, _ := client.Compute.GetInstances(search).ListAll(ctx)
//	for _, inst := range instances {
//	    if aws, ok := inst.(*model.AwsInstance); ok {
//	        fmt.Println(aws.InstanceId.Or(""))
//	    }
//	}
//
// # Real-time updates
//
// [ComputeClient.AddComputeRequirementListener] delivers [model.ComputeRequirement]
// updates to a [subscription.SubscriptionEventListener] as they happen, and reports a failed
// or ended stream to it rather than going quiet. [RequirementHelper] waits for a
// requirement to reach a given status.
package compute
