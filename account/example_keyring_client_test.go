package account_test

import (
	"context"
	"log"

	yellowdog "github.com/yellowdogai/yellowdog-sdk-go"
	"github.com/yellowdogai/yellowdog-sdk-go/model"
	"github.com/yellowdogai/yellowdog-sdk-go/optional"
)

// ExampleKeyringClient demonstrates creating a keyring and managing credentials.
func ExampleKeyringClient() {
	config := yellowdog.DefaultConfig()
	config.APIKey = "your-api-key"
	config.APISecret = "your-api-secret"

	client, err := yellowdog.NewClient(config)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = client.Close() }()

	ctx := context.Background()

	// Create a keyring
	keyring, err := client.Keyring.CreateKeyring(ctx, "my-keyring", "")
	if err != nil {
		log.Fatal(err)
	}
	// The API response may not include the keyring name; set it from the request
	keyring.Name = optional.Of("my-keyring")
	log.Printf("Created keyring: %s\n", keyring.Name.Or(""))

	// Add an AWS credential to the keyring using the object overload
	awsCred := model.NewAwsCredential()
	awsCred.Name = "my-aws-cred"
	awsCred.AccessKeyId = "AKIAIOSFODNN7EXAMPLE"
	awsCred.SecretAccessKey = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"

	updated, err := client.Keyring.PutCredential(ctx, keyring, awsCred)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Keyring now has %d credential(s)\n", len(updated.Credentials))

	// Remove a credential using the object overload
	updated, err = client.Keyring.DeleteCredential(ctx, keyring, "my-aws-cred")
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Keyring now has %d credential(s)\n", len(updated.Credentials))

	// Cleanup
	if err := client.Keyring.DeleteKeyring(ctx, keyring); err != nil {
		log.Printf("Failed to delete keyring: %v", err)
	}
}
