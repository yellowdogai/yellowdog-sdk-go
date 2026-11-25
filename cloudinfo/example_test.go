package cloudinfo_test

import (
	"context"
	"fmt"
	"log"

	yellowdog "github.com/yellowdogai/yellowdog-sdk-go"
	"github.com/yellowdogai/yellowdog-sdk-go/cloudinfo"
	"github.com/yellowdogai/yellowdog-sdk-go/model"
)

// Example demonstrates querying cloud provider metadata. Every cloudinfo search
// takes a search struct and returns a paginated client; ListAll walks every page.
func Example() {
	config := yellowdog.DefaultConfig(
		yellowdog.ServicesSchema{DefaultURL: "https://api.yellowdog.ai"},
		yellowdog.APIKey{ID: "your-api-key-id", Secret: "your-api-key-secret"},
	)

	client, err := yellowdog.NewClient(config)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = client.Close() }()

	ctx := context.Background()

	regionSearch := &cloudinfo.RegionSearch{
		Providers: []model.CloudProvider{
			model.CloudProviderAws,
		},
	}

	regions, err := client.CloudInfo.GetRegions(regionSearch).ListAll(ctx)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d AWS regions\n", len(regions))
	for _, r := range regions {
		fmt.Printf("  - %s\n", r.Name.Or(""))
	}
}
