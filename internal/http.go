package internal

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"
)

type HTTPConfig struct {
	BaseURL    string
	APIKey     string
	APISecret  string
	HTTPClient *http.Client
	Timeout    time.Duration
	UserAgent  string
}

type HTTPClient struct {
	baseURL    string
	apiKey     string
	apiSecret  string
	httpClient *http.Client
	userAgent  string
	logger     *slog.Logger
}

// SetDebug enables or disables debug logging for all HTTP requests.
func (c *HTTPClient) SetDebug(enabled bool) {
	if enabled {
		c.logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))
	} else {
		c.logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
}

func NewHTTPClient(config *HTTPConfig) *HTTPClient {
	client := config.HTTPClient
	if client == nil {
		transport := &http.Transport{
			Proxy:               http.ProxyFromEnvironment,
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		}
		if os.Getenv("YELLOWDOG_API_INSECURE") != "" {
			transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec
		}
		client = &http.Client{
			Timeout:   config.Timeout,
			Transport: transport,
		}
	}

	hc := &HTTPClient{
		baseURL:    config.BaseURL,
		apiKey:     config.APIKey,
		apiSecret:  config.APISecret,
		httpClient: client,
		userAgent:  config.UserAgent,
		logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	return hc
}

func (c *HTTPClient) Do(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Authorization", fmt.Sprintf("yd-key %s:%s", c.apiKey, c.apiSecret))
	return c.httpClient.Do(req)
}

func (c *HTTPClient) Get(ctx context.Context, path string, result interface{}) error {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("creating GET request for %s: %w", path, err)
	}

	c.logger.Debug("HTTP request",
		slog.String("method", "GET"),
		slog.String("url", c.baseURL+path),
	)

	return c.doJSON(req, result)
}

func (c *HTTPClient) Post(ctx context.Context, path string, body, result interface{}) error {
	return c.requestJSON(ctx, "POST", path, body, result)
}

func (c *HTTPClient) Put(ctx context.Context, path string, body, result interface{}) error {
	return c.requestJSON(ctx, "PUT", path, body, result)
}

func (c *HTTPClient) Delete(ctx context.Context, path string) error {
	req, err := http.NewRequestWithContext(ctx, "DELETE", c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("creating DELETE request for %s: %w", path, err)
	}

	return c.doJSON(req, nil)
}

func (c *HTTPClient) DeleteWithResult(ctx context.Context, path string, result interface{}) error {
	req, err := http.NewRequestWithContext(ctx, "DELETE", c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("creating DELETE request for %s: %w", path, err)
	}

	return c.doJSON(req, result)
}

func (c *HTTPClient) NewRequest(ctx context.Context, method, path string, body interface{}) (*http.Request, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshaling request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, err
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return req, nil
}

func (c *HTTPClient) requestJSON(ctx context.Context, method, path string, body, result interface{}) error {
	var bodyData []byte
	if body != nil {
		var err error
		bodyData, err = json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshaling request body: %w", err)
		}
		c.logger.Debug("HTTP request",
			slog.String("method", method),
			slog.String("url", c.baseURL+path),
			slog.Int("body_size", len(bodyData)),
			slog.String("body", string(bodyData)),
		)
	} else {
		c.logger.Debug("HTTP request",
			slog.String("method", method),
			slog.String("url", c.baseURL+path),
		)
	}

	var bodyReader io.Reader
	if len(bodyData) > 0 {
		bodyReader = bytes.NewReader(bodyData)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("creating %s request for %s: %w", method, path, err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return c.doJSON(req, result)
}

const maxResponseSizeBytes = 10 * 1024 * 1024 // 10MB

func (c *HTTPClient) doJSON(req *http.Request, result any) error {
	resp, err := c.Do(req)
	if err != nil {
		return fmt.Errorf("executing HTTP request to %s: %w", req.URL, err)
	}
	defer func() { _ = resp.Body.Close() }()

	limitedBody := http.MaxBytesReader(nil, resp.Body, maxResponseSizeBytes)
	body, _ := io.ReadAll(limitedBody)

	c.logger.Debug("HTTP response",
		slog.Int("status", resp.StatusCode),
		slog.String("status_text", resp.Status),
		slog.Int("body_size", len(body)),
		slog.String("body", string(body)),
	)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return newAPIErrorWithBody(resp, body)
	}

	if result != nil && resp.StatusCode != http.StatusNoContent && len(body) > 0 {
		if s, ok := result.(*string); ok {
			if err := json.Unmarshal(body, s); err != nil {
				// API returned a bare (unquoted) string — use it directly.
				*s = string(body)
			}
		} else if err := json.Unmarshal(body, result); err != nil {
			return fmt.Errorf("decoding response: %w", err)
		}
	}

	return nil
}

// BaseURL returns the base URL of the HTTP client.
func (c *HTTPClient) BaseURL() string {
	return c.baseURL
}

// AuthorizationHeader returns the authorization header value for API requests.
func (c *HTTPClient) AuthorizationHeader() string {
	return fmt.Sprintf("yd-key %s:%s", c.apiKey, c.apiSecret)
}

// APIError represents an error response from the YellowDog API.
type APIError struct {
	StatusCode int
	Status     string
	Message    string
	// ErrorType is the platform's exception type, e.g. InvalidRequestException.
	ErrorType string
	// Detail holds the specifics behind Message, such as which request values
	// failed validation.
	Detail    []string
	RequestId string
	Body      []byte
}

func (e *APIError) Error() string {
	var suffix string
	if e.Message != "" {
		suffix += ": " + e.Message
	}
	if len(e.Detail) > 0 {
		suffix += ": " + strings.Join(e.Detail, ", ")
	}
	return fmt.Sprintf("YellowDog API returned HTTP status %s%s", e.Status, suffix)
}

func newAPIErrorWithBody(resp *http.Response, body []byte) error {
	apiErr := &APIError{
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
		RequestId:  resp.Header.Get("X-Request-ID"),
	}
	return populateAPIError(apiErr, body)
}

func populateAPIError(apiErr *APIError, body []byte) error {
	apiErr.Body = body

	var errResp struct {
		Message   string          `json:"message"`
		Error     string          `json:"error"`
		ErrorType string          `json:"errorType"`
		Detail    json.RawMessage `json:"detail"`
	}
	if json.Unmarshal(body, &errResp) == nil {
		if errResp.Message != "" {
			apiErr.Message = errResp.Message
		} else if errResp.Error != "" {
			apiErr.Message = errResp.Error
		}
		apiErr.ErrorType = errResp.ErrorType
		apiErr.Detail = parseErrorDetail(errResp.Detail)
	}
	return apiErr
}

// parseErrorDetail reads an error's detail, which the platform sends as a list
// of strings but which comes back as a single string from errors raised
// outside it (RFC 9457 problem details, e.g. an unknown path).
func parseErrorDetail(raw json.RawMessage) []string {
	var list []string
	if json.Unmarshal(raw, &list) == nil {
		return list
	}
	var single string
	if json.Unmarshal(raw, &single) == nil && single != "" {
		return []string{single}
	}
	return nil
}
