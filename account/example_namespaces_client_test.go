package account_test

import (
	"context"
	"log"

	yellowdog "github.com/yellowdogai/yellowdog-sdk-go"
	"github.com/yellowdogai/yellowdog-sdk-go/account"
	"github.com/yellowdogai/yellowdog-sdk-go/model"
)

// ExampleNamespacesClient demonstrates creating and managing namespaces.
func ExampleNamespacesClient() {
	config := yellowdog.DefaultConfig()
	config.APIKey = "your-api-key"
	config.APISecret = "your-api-secret"

	client, err := yellowdog.NewClient(config)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = client.Close() }()

	ctx := context.Background()

	// Create a namespace
	id, err := client.Namespaces.CreateNamespace(ctx, &account.CreateNamespaceRequest{
		Namespace: "my-namespace",
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Created namespace with ID: %s\n", id)

	// Retrieve it by ID
	ns, err := client.Namespaces.GetNamespace(ctx, id)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Namespace: %s\n", ns.Namespace)

	// Apply a policy
	if err := client.Namespaces.SaveNamespacePolicy(ctx, &model.NamespacePolicy{
		Namespace: ns.Namespace,
	}); err != nil {
		log.Fatal(err)
	}

	// Cleanup
	if err := client.Namespaces.DeleteNamespace(ctx, id); err != nil {
		log.Fatal(err)
	}
}
