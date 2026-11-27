package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand/v2"
	"net"
	"net/http"
	"net/http/httptrace"
	"strings"
	"sync/atomic"
	"time"
)

type HTTPConfig struct {
	BaseURL    string
	APIKey     string
	APISecret  string
	HTTPClient *http.Client
	Timeout    time.Duration
	UserAgent  string
	Logger     *slog.Logger
	Retry      Retry
}

// Retry sets how requests are retried after a rate limit, a server error or a connection error.
type Retry struct {
	MaxAttempts     int
	InitialInterval time.Duration
	MaxInterval     time.Duration
}

type HTTPClient struct {
	baseURL    string
	apiKey     string
	apiSecret  string
	httpClient *http.Client
	userAgent  string
	logger     *slog.Logger
	retry      Retry
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
		client = &http.Client{
			Timeout:   config.Timeout,
			Transport: transport,
		}
	}

	logger := config.Logger
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}

	hc := &HTTPClient{
		baseURL:    config.BaseURL,
		apiKey:     config.APIKey,
		apiSecret:  config.APISecret,
		httpClient: client,
		userAgent:  config.UserAgent,
		logger:     logger,
		retry:      config.Retry,
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

	return c.doJSON(req, nil, result)
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

	return c.doJSON(req, nil, nil)
}

func (c *HTTPClient) DeleteWithResult(ctx context.Context, path string, result interface{}) error {
	req, err := http.NewRequestWithContext(ctx, "DELETE", c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("creating DELETE request for %s: %w", path, err)
	}

	return c.doJSON(req, nil, result)
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

	return c.doJSON(req, bodyData, result)
}

func (c *HTTPClient) doJSON(req *http.Request, reqBody []byte, result any) error {
	for retry := 0; ; retry++ {
		body, sent, err := c.attempt(req, reqBody)
		if err == nil {
			return decodeResult(body, result)
		}
		if retry == c.retry.MaxAttempts || !isRetryable(req, sent, err) {
			return err
		}

		wait := c.backoff(retry)
		c.logger.Warn("Retrying HTTP request",
			slog.String("method", req.Method),
			slog.String("url", req.URL.String()),
			slog.Int("retry", retry+1),
			slog.Duration("wait", wait),
			slog.String("error", err.Error()),
		)
		if err := sleep(req.Context(), wait); err != nil {
			return err
		}
	}
}

// attempt sends one copy of req, with a fresh body, and returns the response body of a 2xx response, and whether
// the request was sent in full.
func (c *HTTPClient) attempt(req *http.Request, reqBody []byte) ([]byte, bool, error) {
	if reqBody == nil {
		c.logger.Debug("HTTP request",
			slog.String("method", req.Method),
			slog.String("url", req.URL.String()),
		)
	} else {
		c.logger.Debug("HTTP request",
			slog.String("method", req.Method),
			slog.String("url", req.URL.String()),
			slog.Int("body_size", len(reqBody)),
			slog.String("body", string(reqBody)),
		)
	}

	var sent atomic.Bool
	trace := &httptrace.ClientTrace{
		WroteRequest: func(info httptrace.WroteRequestInfo) { sent.Store(info.Err == nil) },
	}
	attemptReq := req.Clone(httptrace.WithClientTrace(req.Context(), trace))
	if reqBody != nil {
		attemptReq.Body = io.NopCloser(bytes.NewReader(reqBody))
	}

	resp, err := c.Do(attemptReq)
	if err != nil {
		return nil, sent.Load(), fmt.Errorf("executing HTTP request to %s: %w", req.URL, err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, sent.Load(), fmt.Errorf("%w from %s: %w", errReadingResponse, req.URL, err)
	}

	c.logger.Debug("HTTP response",
		slog.Int("status", resp.StatusCode),
		slog.String("status_text", resp.Status),
		slog.Int("body_size", len(body)),
		slog.String("body", string(body)),
	)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, sent.Load(), newAPIErrorWithBody(resp, body)
	}
	if resp.StatusCode == http.StatusNoContent {
		return nil, sent.Load(), nil
	}
	return body, sent.Load(), nil
}

func decodeResult(body []byte, result any) error {
	if result == nil || len(body) == 0 {
		return nil
	}
	if s, ok := result.(*string); ok {
		if err := json.Unmarshal(body, s); err != nil {
			// API returned a bare (unquoted) string — use it directly.
			*s = string(body)
		}
	} else if err := json.Unmarshal(body, result); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}
	return nil
}

var errReadingResponse = errors.New("reading HTTP response")

// isRetryable reports whether err is likely to pass on a retry: a rate limit, a server error or a connection
// error. A POST isn't retried once it timed out after it was sent, as the platform may still be processing it, or
// once its response started, as the platform has processed it.
func isRetryable(req *http.Request, sent bool, err error) bool {
	if req.Context().Err() != nil {
		return false
	}
	if req.Method == http.MethodPost && errors.Is(err, errReadingResponse) {
		return false
	}

	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusTooManyRequests ||
			(apiErr.StatusCode >= 500 && apiErr.StatusCode <= 511)
	}

	var netErr net.Error
	timedOut := errors.As(err, &netErr) && netErr.Timeout()
	return req.Method != http.MethodPost || !sent || !timedOut
}

// backoff returns a random wait before the given retry, counting from zero, using full jitter: between zero and
// the initial interval doubled for each earlier retry, capped at the maximum interval.
func (c *HTTPClient) backoff(retry int) time.Duration {
	limit := c.retry.InitialInterval
	for i := 0; i < retry && limit < c.retry.MaxInterval; i++ {
		limit *= 2
	}
	limit = min(limit, c.retry.MaxInterval)
	if limit <= 0 {
		return 0
	}
	return rand.N(limit)
}

func sleep(ctx context.Context, wait time.Duration) error {
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
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
