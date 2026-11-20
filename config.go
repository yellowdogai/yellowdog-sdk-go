package yellowdog

import (
	"log/slog"
	"net/http"
	"os"
	"regexp"
	"strings"
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

	// UserAgentProducts that identify the application, e.g. Product{ Name: "my-app", Version: "1.2.0" }.
	//
	// The User-Agent header lists them in the order given, after the SDK and Go versions. List the application itself
	// last, and anything it is built on before it, such as a framework that your applications share. Identifying the
	// application is recommended. It lets YellowDog see which applications, and which versions of them, use the
	// platform, so YellowDog can provide you with better support.
	UserAgentProducts []Product
}

// Product identifies an application in the User-Agent header that the SDK sends. The SDK adds it after its own version
// and Go version, e.g. yd/1.1.0 yellowdog-sdk-go/16.13.0 go/1.26.2 my-app/1.2.0
//
// The Name and Version should each be a token, as defined in RFC 9110 section 5.6.2. That is, one or more ASCII
// letters, digits or any of !#$%&'*+-.^_`|~
type Product struct {
	// Name of the application, such as its package or artifact name
	Name string

	// Version the application's version
	Version string
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig(products ...Product) *Config {
	config := &Config{
		BaseURL:           "https://api.yellowdog.ai",
		HTTPClient:        nil,
		Timeout:           30 * time.Second,
		UserAgentProducts: products,
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

func (c *Config) userAgentSuffix() string {
	suffix := ""
	for _, p := range c.UserAgentProducts {
		suffix = suffix + " " + p.userAgentProduct()
	}
	return suffix
}

func (p Product) userAgentProduct() string {
	return sanitiseRfc9110Token("name", p.Name) + "/" + sanitiseRfc9110Token("version", p.Version)
}

const rfc9110TokenCharacters = `A-Za-z0-9!#$%&'*+.^_` + "`" + `|~-`

var (
	rfc9110TokenRegEx           = regexp.MustCompile("^[" + rfc9110TokenCharacters + "]+$")
	rfc9110TokenDisallowedRegEx = regexp.MustCompile("[^" + rfc9110TokenCharacters + "]")
)

func sanitiseRfc9110Token(component string, value string) string {
	if rfc9110TokenRegEx.MatchString(value) {
		return value
	}
	sanitised := strings.TrimSpace(value)
	sanitised = strings.ReplaceAll(sanitised, "/", "#")
	sanitised = rfc9110TokenDisallowedRegEx.ReplaceAllString(sanitised, "_")
	slog.Warn("Product "+component+" is not an RFC 9110 token, sanitised value will be sent instead", component, value, "sanitised", sanitised)
	return sanitised
}
