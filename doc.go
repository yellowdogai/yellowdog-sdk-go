// Package yellowdog is the YellowDog Go SDK.
//
// It provides a client for the YellowDog Platform API, covering compute
// infrastructure, distributed work scheduling, machine images, account
// management, and cloud provider information.
//
// # Getting started
//
// Install the SDK:
//
//	go get github.com/yellowdogai/yellowdog-sdk-go
//
// Create a [PlatformClient] with your API credentials, then reach each service
// through its field on the client. See the package example below for a
// complete call.
//
// # Configuration
//
// [DefaultConfig] returns a [Config] populated from the environment:
//
//   - YELLOWDOG_API_KEY — application API key (required)
//   - YELLOWDOG_API_SECRET — application API secret (required)
//   - YELLOWDOG_API_URL — API base URL, defaulting to production (optional)
//   - YELLOWDOG_DEBUG — set to 1 to log every request and response in full
//
// Every field can also be set directly, including Timeout, UserAgent and
// HTTPClient for supplying your own *http.Client. [NewClient] validates the
// configuration and fails if either credential is missing.
//
// # Services
//
// The client exposes service-specific sub-clients as fields:
//
//	client.Compute     // *compute.ComputeClient
//	client.Work        // *scheduler.WorkClient
//	client.WorkerPool  // *scheduler.WorkerPoolClient
//	client.Keyring     // *account.KeyringClient
//	client.Images      // *images.ImagesClient
//	client.Allowances  // *usage.AllowancesClient
//	client.CloudInfo   // *cloudinfo.CloudInfoClient
//	client.Account     // *account.AccountClient
//	client.Application // *account.ApplicationClient
//	client.Namespaces  // *account.NamespacesClient
//
// # Searching and pagination
//
// Search methods do not take a context and do not return a slice. They take a
// search struct and return a SearchClient: Slice retrieves one slice from a
// pagination.SliceReference, ListAll retrieves every slice, and Iterate ranges
// over the items, retrieving each slice as the loop reaches it. The context is
// passed to that final call rather than to the search method.
//
// # Error handling
//
// API errors are returned as [APIError] values, which expose the HTTP status
// code and the platform error message. Match them with errors.As — see
// [APIError] for a worked example.
//
// # Optional fields
//
// Nullable fields are typed optional.Optional, which distinguishes "absent"
// from "present and zero". Set them with optional.Of, and read them with Get,
// which returns the value and whether it is present, or Or, which supplies a
// default. See [github.com/yellowdogai/yellowdog-sdk-go/optional].
//
// # Thread safety
//
// [PlatformClient] and the service clients hanging off it are safe for
// concurrent use. The helper types returned by Helper methods are per-call and
// should not be shared between goroutines.
//
// # Sub-packages
//
// Domain types shared across services live in model; the service packages hold
// their own clients, requests and search structs:
//
//   - [github.com/yellowdogai/yellowdog-sdk-go/compute] — compute requirements, instances, source and requirement templates
//   - [github.com/yellowdogai/yellowdog-sdk-go/scheduler] — work requirements, tasks, worker pools and nodes
//   - [github.com/yellowdogai/yellowdog-sdk-go/account] — users, groups, roles, applications, keyrings and namespaces
//   - [github.com/yellowdogai/yellowdog-sdk-go/images] — machine image families, groups and images
//   - [github.com/yellowdogai/yellowdog-sdk-go/usage] — allowances and usage tracking
//   - [github.com/yellowdogai/yellowdog-sdk-go/cloudinfo] — cloud provider regions, instance types and pricing
//   - [github.com/yellowdogai/yellowdog-sdk-go/model] — shared domain types, enums and constructors
//   - [github.com/yellowdogai/yellowdog-sdk-go/optional] — Optional[T], used for every nullable field
//   - [github.com/yellowdogai/yellowdog-sdk-go/pagination] — page references for search results
package yellowdog
