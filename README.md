# YellowDog Go SDK

Official Go SDK for the [YellowDog Platform](https://yellowdog.ai) — provision
compute across cloud providers and schedule work onto it.

[![Go Reference](https://pkg.go.dev/badge/github.com/yellowdogai/yellowdog-sdk-go.svg)](https://pkg.go.dev/github.com/yellowdogai/yellowdog-sdk-go)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

```bash
go get github.com/yellowdogai/yellowdog-sdk-go
```

Requires Go 1.25 or later.

## What it covers

YellowDog provisions compute from multiple clouds against a single requirement and
schedules containerised or shell work onto the resulting workers. The SDK is a
typed client for that platform:

- **Compute** — Manage compute requirements and the instances behind them, as well as 
  reusable source and requirement templates with different deployment strategies.
- **Work scheduling** — Manage work requirements, task groups and tasks, as well as
  provisioned and configured pools of nodes (worker pools) for scheduling tasks onto.
- **Machine images** — Register families of images, versioned groups, and per-region 
  images for use in compute requirements.
- **Account and access** — Manage users, groups, roles and applications; keyrings 
  holding the cloud credentials used to provision; and namespaces for isolating 
  resources.
- **Usage** — allowances enable controlling compute usage account-wide, per source,
  or per requirement, with soft and hard enforcement.
- **Cloud information** — regions, instance types and pricing across providers can
  be used to build compute requirements dynamically. Custom attributes can also be
  defined in support of workload-specific provisioning requirements.

Provisioning is supported on **AWS** (instances and fleets), **Azure** (instances and
scale sets), **Google Cloud** (instances and instance groups) and **OCI** (instances
and instance pools).

## Documentation

**[pkg.go.dev/github.com/yellowdogai/yellowdog-sdk-go](https://pkg.go.dev/github.com/yellowdogai/yellowdog-sdk-go)**
— getting started, configuration, the service map, and runnable examples for every
package. 

Platform concepts and the underlying REST API are documented at
[docs.yellowdog.co](https://docs.yellowdog.co).

## Status

Pre-release and untagged. The API surface may change without notice and does not yet
follow semantic versioning.

## Development

`go test -short ./...` runs the unit tests, which need no credentials or network access.

## License

Apache License 2.0 — see [LICENSE](LICENSE).

Copyright © 2026 YellowDog Limited
