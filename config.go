package yellowdog

import (
	"net/http"
	"os"
	"time"
)

// Config holds the configuration for the YellowDog Platform Client.
type Config struct {
	// BaseURL is the base URL for the YellowDog API (e.g., "https://api.yellowdog.ai")
	BaseURL string

	// APIKey is the application API key
	APIKey string

	// APISecret is the application API secret
	APISecret string

	// HTTPClient is the HTTP client to use for requests.
	// If nil, a default client with reasonable timeouts will be used.
	HTTPClient *http.Client

	// Timeout is the default timeout for API requests.
	// If zero, a reasonable default will be used (30 seconds).
	Timeout time.Duration

	// UserAgent is the User-Agent header to send with requests. If empty, DefaultUserAgent is
	// used. A value set here replaces the default entirely, and the platform identifies a client
	// only from the format DefaultUserAgent documents.
	UserAgent string
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
	config := &Config{
		BaseURL:    "https://api.yellowdog.ai",
		HTTPClient: nil,
		Timeout:    30 * time.Second,
		UserAgent:  DefaultUserAgent(),
	}

	if os.Getenv("YELLOWDOG_API_URL") != "" {
		config.BaseURL = os.Getenv("YELLOWDOG_API_URL")
	}
	if os.Getenv("YELLOWDOG_API_KEY") != "" {
		config.APIKey = os.Getenv("YELLOWDOG_API_KEY")
	}
	if os.Getenv("YELLOWDOG_API_SECRET") != "" {
		config.APISecret = os.Getenv("YELLOWDOG_API_SECRET")
	}
	return config
}

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	if c.BaseURL == "" {
		return &ConfigError{Field: "BaseURL", Message: "base URL is required"}
	}
	if c.APIKey == "" {
		return &ConfigError{Field: "APIKey", Message: "API key is required"}
	}
	if c.APISecret == "" {
		return &ConfigError{Field: "APISecret", Message: "API secret is required"}
	}
	return nil
}
