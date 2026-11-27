package yellowdog

import (
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/yellowdogai/yellowdog-sdk-go/model"
)

// Config holds the configuration for the YellowDog Platform Client.
type Config struct {
	// ServicesSchema locates the YellowDog Platform services
	ServicesSchema ServicesSchema

	// APIKey authenticates the application
	APIKey APIKey

	// HTTPClient is the HTTP client to use for requests.
	// If nil, a default client with reasonable timeouts will be used.
	HTTPClient *http.Client

	// Timeout is the default timeout for API requests.
	// If zero, a reasonable default will be used (30 seconds).
	// It limits each attempt at a request separately, so retries can take a request longer.
	Timeout time.Duration

	// Retry sets how the client retries a request that fails with a rate limit (HTTP 429), a server error (HTTP 500 to
	// 511) or a connection error. It retries every method, except a POST that timed out after it was sent, as the
	// platform may still be processing it, or whose response was cut short, as the platform has already processed it.
	// Other errors, such as HTTP 401, 403 and 404, are returned straight away.
	// When retries run out, the last error is returned. Server-sent event streams, behind the clients' listeners,
	// aren't retried.
	//
	// MaxAttempts is the maximum number of retries after the first attempt, so a request is sent at most
	// MaxAttempts + 1 times. Zero turns retries off. Each wait before a retry is random, between zero and
	// InitialInterval doubled for each earlier retry, up to MaxInterval. The wait ends early, with the context's error,
	// when the request's context is done.
	Retry model.RetryProperties

	// UserAgentProducts that identify the application, e.g. Product{ Name: "my-app", Version: "1.2.0" }.
	//
	// The User-Agent header lists them in the order given, after the SDK and Go versions. List the application itself
	// last, and anything it is built on before it, such as a framework that your applications share. Identifying the
	// application is recommended. It lets YellowDog see which applications, and which versions of them, use the
	// platform, so YellowDog can provide you with better support.
	UserAgentProducts []Product

	// Logger receives a debug record for every request and response, with its body, and a warning for every retry.
	// Bodies can hold secrets, such as cloud provider credentials, so don't enable debug logging in production. If
	// nil, nothing is logged.
	Logger *slog.Logger
}

// ServicesSchema locates the YellowDog Platform services.
type ServicesSchema struct {
	// DefaultURL is the base URL of the YellowDog Platform, e.g. "https://api.yellowdog.ai"
	DefaultURL string
}

// APIKey is an application's API key, which authenticates the application with the YellowDog Platform.
type APIKey struct {
	// ID is the API key's ID
	ID string

	// Secret is the API key's secret
	Secret Secret
}

// Secret is a value to keep secret, such as an API key's secret. Formatting, logging or marshalling it gives
// [REDACTED] instead of the value, so it doesn't leak by accident. Convert it with string(s) to read the value.
type Secret string

const redacted = "[REDACTED]"

// String returns [REDACTED].
func (Secret) String() string {
	return redacted
}

// Format writes [REDACTED] for every verb, including %#v, %x and %q.
func (Secret) Format(f fmt.State, _ rune) {
	_, _ = fmt.Fprint(f, redacted)
}

// MarshalText returns [REDACTED], which encoding/json and other text-based encoders use.
func (Secret) MarshalText() ([]byte, error) {
	return []byte(redacted), nil
}

// LogValue returns [REDACTED], so log/slog never logs the value.
func (Secret) LogValue() slog.Value {
	return slog.StringValue(redacted)
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

// DefaultConfig returns a Config for the given services and API key, with sensible defaults for everything else. It
// retries a failed request up to 3 times, waiting at most 10 seconds before the first retry, doubling to at most a
// minute.
func DefaultConfig(servicesSchema ServicesSchema, apiKey APIKey, products ...Product) *Config {
	return &Config{
		ServicesSchema: servicesSchema,
		APIKey:         apiKey,
		HTTPClient:     nil,
		Timeout:        30 * time.Second,
		Retry: model.RetryProperties{
			MaxAttempts:     3,
			InitialInterval: model.Duration(10 * time.Second),
			MaxInterval:     model.Duration(time.Minute),
		},
		UserAgentProducts: products,
	}
}

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	if c.ServicesSchema.DefaultURL == "" {
		return &ConfigError{Field: "ServicesSchema.DefaultURL", Message: "default URL is required"}
	}
	if c.APIKey.ID == "" {
		return &ConfigError{Field: "APIKey.ID", Message: "API key ID is required"}
	}
	if c.APIKey.Secret == "" {
		return &ConfigError{Field: "APIKey.Secret", Message: "API key secret is required"}
	}
	if c.Retry.MaxAttempts < 0 {
		return &ConfigError{Field: "Retry.MaxAttempts", Message: "must not be negative"}
	}
	if c.Retry.InitialInterval < 0 {
		return &ConfigError{Field: "Retry.InitialInterval", Message: "must not be negative"}
	}
	if c.Retry.MaxInterval < 0 {
		return &ConfigError{Field: "Retry.MaxInterval", Message: "must not be negative"}
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
