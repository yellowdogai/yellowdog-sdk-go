package yellowdog

import (
	"os"

	"github.com/yellowdogai/yellowdog-sdk-go/account"
	"github.com/yellowdogai/yellowdog-sdk-go/cloudinfo"
	"github.com/yellowdogai/yellowdog-sdk-go/compute"
	"github.com/yellowdogai/yellowdog-sdk-go/images"
	"github.com/yellowdogai/yellowdog-sdk-go/internal"
	"github.com/yellowdogai/yellowdog-sdk-go/scheduler"
	"github.com/yellowdogai/yellowdog-sdk-go/usage"
)

// PlatformClient is the main entry point for the YellowDog SDK.
// It provides access to all YellowDog Platform services.
type PlatformClient struct {
	config     *Config
	httpClient *internal.HTTPClient

	Compute     *compute.ComputeClient
	Work        *scheduler.WorkClient
	WorkerPool  *scheduler.WorkerPoolClient
	Keyring     *account.KeyringClient
	Images      *images.ImagesClient
	Allowances  *usage.AllowancesClient
	CloudInfo   *cloudinfo.CloudInfoClient
	Account     *account.AccountClient
	Application *account.ApplicationClient
	Namespaces  *account.NamespacesClient
}

// NewClient creates a new YellowDog Platform Client.
func NewClient(config *Config) (*PlatformClient, error) {
	if config == nil {
		config = DefaultConfig()
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	userAgent := config.UserAgent
	if userAgent == "" {
		userAgent = DefaultUserAgent()
	}

	httpClient := internal.NewHTTPClient(&internal.HTTPConfig{
		BaseURL:    config.BaseURL,
		APIKey:     config.APIKey,
		APISecret:  config.APISecret,
		HTTPClient: config.HTTPClient,
		Timeout:    config.Timeout,
		UserAgent:  userAgent,
	})

	if os.Getenv("YELLOWDOG_DEBUG") == "1" || os.Getenv("YELLOWDOG_DEBUG") == "true" {
		httpClient.SetDebug(true)
	}

	client := &PlatformClient{
		config:     config,
		httpClient: httpClient,
	}

	client.Compute = compute.NewComputeClient(httpClient)
	client.Work = scheduler.NewWorkClient(httpClient)
	client.WorkerPool = scheduler.NewWorkerPoolClient(httpClient)
	client.Keyring = account.NewKeyringClient(httpClient)
	client.Images = images.NewImagesClient(httpClient)
	client.Allowances = usage.NewAllowancesClient(httpClient)
	client.CloudInfo = cloudinfo.NewCloudInfoClient(httpClient)
	client.Account = account.NewAccountClient(httpClient)
	client.Application = account.NewApplicationClient(httpClient)
	client.Namespaces = account.NewNamespacesClient(httpClient)

	return client, nil
}

// Close ends the client's subscriptions to live updates, without notifying their listeners. It
// always returns nil.
func (c *PlatformClient) Close() error {
	c.Compute.Close()
	c.Work.Close()
	c.WorkerPool.Close()
	return nil
}
