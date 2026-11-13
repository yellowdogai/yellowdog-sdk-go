// Package model contains the shared data types used across all YellowDog SDK packages.
//
// # Setting fields
//
// A model struct is a data-transfer type: the keyed composite literal is how one
// is built, and every field is set directly. Which form a field takes is visible
// in its type, and the compiler enforces it:
//
//	Field type                       How to set it
//	string, int, []string, …         assign it — the platform requires it
//	optional.Optional[T]             optional.Of(v)
//
// So:
//
//	source := &model.AwsInstancesComputeSource{
//		Name:    "my-source",           // required
//		KeyName: optional.Of("my-key"), // optional
//	}
//
// The concrete types of the polymorphic interfaces listed below need nothing
// more. Each writes the discriminator the platform uses to resolve its type when
// it is marshaled.
//
// # Polymorphic types
//
// Several API endpoints return polymorphic values — a single interface that may
// be one of several concrete types depending on a JSON "type" discriminator.
// The following interfaces are polymorphic:
//
//   - [Allowance] — [AccountAllowance], [SourcesAllowance], [RequirementAllowance], …
//   - [ComputeRequirementTemplate] — [ComputeRequirementStaticTemplate], [ComputeRequirementDynamicTemplate]
//   - [ComputeSource] — [AwsInstancesComputeSource], [AzureInstancesComputeSource], …
//   - [Credential] — [AwsCredential], [AzureClientCredential], [GoogleCloudCredential], …
//   - [Instance] — [AwsInstance], [AzureInstance], [GceInstance], [OciInstance], …
//   - [WorkerPool] — [ConfiguredWorkerPool], …
//
// The concrete types implement these interfaces with pointer receivers, so
// assert to the pointer type:
//
//	aws, ok := inst.(*model.AwsInstance)
//
// # Duration
//
// [Duration] is a time.Duration that is sent and received as an ISO 8601
// duration string, such as "PT30S". Build one from a Go duration string with
// [ParseDuration], or with [MustParseDuration] for literals:
//
//	timeout := model.MustParseDuration("30s")
//	window  := model.MustParseDuration("24h")
package model
