package yellowdog

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync/atomic"
	"testing"
	"time"

	"github.com/yellowdogai/yellowdog-sdk-go/model"
	"github.com/yellowdogai/yellowdog-sdk-go/subscription"
)

// The pattern the platform matches to identify a client; a header that does not match it leaves
// the client unidentified.
var platformUserAgent = regexp.MustCompile(`yd/(\d+)\.(\d+)\.(\d+) ([^ /]+)/(\d+)\.(\d+)\.(\d+) ([^ /]+)/(.+)`)

func TestBaseUserAgentMatchesThePlatformFormat(t *testing.T) {
	match := platformUserAgent.FindStringSubmatch(baseUserAgent())
	if match == nil {
		t.Fatalf("BaseUserAgent() = %q, which the platform cannot parse", baseUserAgent())
	}
	if match[4] != "yellowdog-sdk-go" || match[8] != "go" {
		t.Errorf("BaseUserAgent() = %q, want application yellowdog-sdk-go and language go", baseUserAgent())
	}
}

func TestRequestsSendTheUserAgent(t *testing.T) {
	cases := []struct {
		name     string
		products []Product
		want     string
	}{
		{"no products", []Product{}, baseUserAgent()},
		{"one product", []Product{{"my-tool", "2.0.0"}}, baseUserAgent() + " my-tool/2.0.0"},
		{"multiple product", []Product{{"my-framework", "1.0.0"}, {"my-tool", "2.0.0"}}, baseUserAgent() + " my-framework/1.0.0 my-tool/2.0.0"},
		{"product with non token chars", []Product{{" a b/c ", " v / 1"}}, baseUserAgent() + " a_b#c/v_#_1"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var got string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				got = r.Header.Get("User-Agent")
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"id":"ns-1","namespace":"example"}`))
			}))
			defer server.Close()

			client, err := NewClient(&Config{ServicesSchema: ServicesSchema{DefaultURL: server.URL}, APIKey: APIKey{ID: "key", Secret: "secret"}, UserAgentProducts: c.products})
			if err != nil {
				t.Fatalf("NewClient: %v", err)
			}
			defer func() { _ = client.Close() }()

			if _, err := client.Namespaces.GetNamespace(context.Background(), "ns-1"); err != nil {
				t.Fatalf("GetNamespace: %v", err)
			}
			if got != c.want {
				t.Errorf("User-Agent = %q, want %q", got, c.want)
			}
		})
	}
}

func TestRequestsAndResponsesAreLoggedToTheConfiguredLogger(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"ns-1","namespace":"example"}`))
	}))
	defer server.Close()

	var logs bytes.Buffer
	config := DefaultConfig(ServicesSchema{DefaultURL: server.URL}, APIKey{ID: "key", Secret: "secret"})
	config.Logger = slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug}))
	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer func() { _ = client.Close() }()

	if _, err := client.Namespaces.GetNamespace(context.Background(), "ns-1"); err != nil {
		t.Fatalf("GetNamespace: %v", err)
	}
	for _, want := range []string{`msg="HTTP request" method=GET`, `msg="HTTP response" status=200`, `namespace\":\"example`} {
		if !bytes.Contains(logs.Bytes(), []byte(want)) {
			t.Errorf("logs do not contain %s:\n%s", want, logs.String())
		}
	}
}

func TestRequestsAreRetriedThroughAnHTTPClientOfYourOwn(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requests.Add(1) == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"ns-1","namespace":"example"}`))
	}))
	defer server.Close()

	config := DefaultConfig(ServicesSchema{DefaultURL: server.URL}, APIKey{ID: "key", Secret: "secret"})
	config.HTTPClient = server.Client()
	config.Retry = model.RetryProperties{MaxAttempts: 1}
	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer func() { _ = client.Close() }()

	if _, err := client.Namespaces.GetNamespace(context.Background(), "ns-1"); err != nil {
		t.Fatalf("GetNamespace: %v", err)
	}
	if requests.Load() != 2 {
		t.Errorf("requests = %d, want 2", requests.Load())
	}
}

// Close ends the streams behind every client's listeners, as removing their last listeners would,
// without notifying them.
func TestCloseEndsSubscriptionsWithoutNotifying(t *testing.T) {
	opened := make(chan string, 3)
	ended := make(chan string, 3)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		opened <- r.URL.Path
		<-r.Context().Done()
		ended <- r.URL.Path
	}))
	defer server.Close()

	client, err := NewClient(&Config{ServicesSchema: ServicesSchema{DefaultURL: server.URL}, APIKey: APIKey{ID: "key", Secret: "secret"}})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	notified := make(chan string, 6)
	client.Compute.AddComputeRequirementListener("cr-1", subscription.SubscriptionEventListener[*model.ComputeRequirement]{
		SubscriptionError:     func(err error) { notified <- "compute error: " + err.Error() },
		SubscriptionCancelled: func() { notified <- "compute cancelled" },
	})
	client.Work.AddWorkRequirementListener("wr-1", subscription.SubscriptionEventListener[*model.WorkRequirement]{
		SubscriptionError:     func(err error) { notified <- "work error: " + err.Error() },
		SubscriptionCancelled: func() { notified <- "work cancelled" },
	})
	client.WorkerPool.AddWorkerPoolListener("wp-1", subscription.SubscriptionEventListener[model.WorkerPool]{
		SubscriptionError:     func(err error) { notified <- "worker pool error: " + err.Error() },
		SubscriptionCancelled: func() { notified <- "worker pool cancelled" },
	})
	for range 3 {
		select {
		case <-opened:
		case <-time.After(5 * time.Second):
			t.Fatal("timed out waiting for the streams to open")
		}
	}

	if err := client.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	for range 3 {
		select {
		case <-ended:
		case <-time.After(5 * time.Second):
			t.Fatal("timed out waiting for Close to end the streams")
		}
	}
	select {
	case n := <-notified:
		t.Errorf("listener notified after Close: %s", n)
	case <-time.After(100 * time.Millisecond):
	}
}
