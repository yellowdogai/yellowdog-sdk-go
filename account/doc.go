// Package account provides access to the YellowDog Account API.
//
// It covers user management, group membership, roles and permissions,
// application credentials, keyring and credential storage, and namespaces.
//
// # Clients
//
// The following clients are available through [yellowdog.PlatformClient]:
//
//   - [AccountClient] — users, groups, roles, permissions
//   - [ApplicationClient] — application tokens and API keys
//   - [KeyringClient] — keyrings and cloud provider credentials
//   - [NamespacesClient] — namespace creation and policy management
//
// # Keyring credentials
//
// A keyring stores cloud provider credentials (AWS, Azure, GCP, OCI, …).
// Credentials are identified by name within a keyring. Both name-based and
// object-based overloads are provided:
//
//	// Name-based
//	kr, err := client.Keyring.PutCredentialByName(ctx, "my-keyring", cred)
//
//	// Object-based (convenience wrapper)
//	kr, err := client.Keyring.PutCredential(ctx, keyring, cred)
package account
