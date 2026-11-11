package internal

import (
	"context"
	"fmt"
	"net/http"

	gosse "github.com/tmaxmax/go-sse"
)

// StreamEvent is one event read from an update stream. Data is empty for an
// event that carries no payload, such as the stream's heartbeat.
type StreamEvent struct {
	Type string
	Data string
}

// SSEClient opens a single update stream. Deliberately narrow, and expressed in
// this package's own types: it is the only place the SSE library is visible, so
// changing library touches nothing else.
type SSEClient interface {
	// Stream delivers events to handler until the stream ends, then returns the
	// reason it ended. A wrapped io.EOF means the server closed the stream in an
	// orderly fashion; ctx cancellation returns the context's error.
	Stream(ctx context.Context, handler func(StreamEvent)) error
}

type sseClient struct {
	url    string
	header http.Header
	client *gosse.Client
}

func (s *sseClient) Stream(ctx context.Context, handler func(StreamEvent)) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.url, nil)
	if err != nil {
		return fmt.Errorf("creating stream request for %s: %w", s.url, err)
	}
	for name, values := range s.header {
		for _, value := range values {
			req.Header.Add(name, value)
		}
	}

	conn := s.client.NewConnection(req)
	conn.SubscribeToAll(func(event gosse.Event) {
		handler(StreamEvent{Type: event.Type, Data: event.Data})
	})
	return conn.Connect()
}

// NewSSEClient creates an authenticated client for the update stream at path.
func (c *HTTPClient) NewSSEClient(path string) SSEClient {
	header := http.Header{}
	header.Set("Authorization", c.AuthorizationHeader())
	if c.userAgent != "" {
		header.Set("User-Agent", c.userAgent)
	}

	return &sseClient{
		url:    c.baseURL + path,
		header: header,
		client: &gosse.Client{
			// A stream is long-lived, so it must not inherit HTTPConfig.Timeout
			// from the shared client: that timeout would sever a healthy
			// connection. The transport is shared, keeping connection pooling
			// and any TLS configuration.
			HTTPClient: &http.Client{Transport: c.httpClient.Transport},

			// Reconnection is disabled so that a stream ending is reported to
			// the caller rather than papered over. Left enabled, an orderly
			// close - which is how the platform ends a stream once an entity is
			// finalised - is retried indefinitely, reopening the stream and
			// redelivering its opening event each time.
			Backoff: gosse.Backoff{MaxRetries: -1},
		},
	}
}
