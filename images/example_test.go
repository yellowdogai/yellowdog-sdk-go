package images_test

import (
	"context"
	"fmt"
	"log"

	yellowdog "github.com/yellowdogai/yellowdog-sdk-go"
	"github.com/yellowdogai/yellowdog-sdk-go/model"
)

// Example demonstrates the image hierarchy: a family holds groups, a group holds
// images, and GetLatestImageGroupByFamily resolves the most recent group. Deleting
// the family removes the groups and images beneath it.
func Example() {
	config := yellowdog.DefaultConfig()
	config.APIKey = "your-api-key"
	config.APISecret = "your-api-secret"

	client, err := yellowdog.NewClient(config)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = client.Close() }()

	ctx := context.Background()

	family, err := client.Images.AddImageFamily(ctx, &model.MachineImageFamily{
		Name:      "ubuntu-20.04",
		Namespace: "my-namespace",
		OsType:    model.ImageOsTypeLinux,
		Access:    model.ImageAccessPrivate,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := client.Images.DeleteImageFamily(ctx, family); err != nil {
			log.Printf("Warning: failed to delete family: %v", err)
		}
	}()
	fmt.Printf("Created image family: %s\n", family.Name)

	group, err := client.Images.AddImageGroup(ctx, family, &model.MachineImageGroup{
		Name:   "v1.0",
		OsType: model.ImageOsTypeLinux,
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Created image group: %s\n", group.Name)

	// One image per region, all within the same group.
	for _, img := range []*model.MachineImage{
		{
			Name:            "ubuntu-us-east-1",
			Provider:        model.CloudProviderAws,
			ProviderImageId: "ami-12345678",
			Regions:         []string{"us-east-1"},
		},
		{
			Name:            "ubuntu-eu-west-2",
			Provider:        model.CloudProviderAws,
			ProviderImageId: "ami-87654321",
			Regions:         []string{"eu-west-2"},
		},
	} {
		added, err := client.Images.AddImage(ctx, group, img)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Added image: %s (%s)\n", added.Name, added.Provider)
	}

	latest, err := client.Images.GetLatestImageGroupByFamily(ctx, family.Id.Or(""))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Latest group: %s (%d images)\n", latest.Name, len(latest.Images))
}
