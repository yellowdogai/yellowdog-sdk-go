package yellowdog

import (
	"context"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/yellowdogai/yellowdog-sdk-go/model"
	"github.com/yellowdogai/yellowdog-sdk-go/subscription"
)

// The pattern the platform matches to identify a client; a header that does not match it leaves
// the client unidentified.
var platformUserAgent = regexp.MustCompile(`yd/(\d+)\.(\d+)\.(\d+) ([^ /]+)/(\d+)\.(\d+)\.(\d+) ([^ /]+)/(.+)`)

func TestDefaultUserAgentMatchesThePlatformFormat(t *testing.T) {
	match := platformUserAgent.FindStringSubmatch(DefaultUserAgent())
	if match == nil {
		t.Fatalf("DefaultUserAgent() = %q, which the platform cannot parse", DefaultUserAgent())
	}
	if match[4] != "yellowdog-sdk-go" || match[8] != "go" {
		t.Errorf("DefaultUserAgent() = %q, want application yellowdog-sdk-go and language go", DefaultUserAgent())
	}
}

func TestRequestsSendTheUserAgent(t *testing.T) {
	cases := []struct {
		name      string
		userAgent string
		want      string
	}{
		{"default when unset", "", DefaultUserAgent()},
		{"configured value", "my-tool/2.0.0", "my-tool/2.0.0"},
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

			client, err := NewClient(&Config{BaseURL: server.URL, APIKey: "key", APISecret: "secret", UserAgent: c.userAgent})
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

	client, err := NewClient(&Config{BaseURL: server.URL, APIKey: "key", APISecret: "secret"})
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
