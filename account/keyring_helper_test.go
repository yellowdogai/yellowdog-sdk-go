package account

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/yellowdogai/yellowdog-sdk-go/internal"
	"github.com/yellowdogai/yellowdog-sdk-go/optional"
)

// These tests assert the verb and path that the hand-written GetKeyring, and the
// hand-written proxy method behind GetKeyrings, put on the wire.

func newKeyringTestClient(t *testing.T, handler http.HandlerFunc) *KeyringClient {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	return NewKeyringClient(internal.NewHTTPClient(&internal.HTTPConfig{
		BaseURL:   server.URL,
		APIKey:    "test-key",
		APISecret: "test-secret",
		UserAgent: "test-agent",
		Timeout:   5 * time.Second,
	}))
}

func TestGetKeyringsRequestShape(t *testing.T) {
	var gotMethod, gotPath string
	var gotQuery url.Values
	client := newKeyringTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		gotQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		if _, err := io.WriteString(w, `{"items":[{"id":"ydid:keyring:1","name":"one"}],"nextSliceId":""}`); err != nil {
			t.Errorf("failed to write response: %v", err)
		}
	})

	summaries, err := client.GetKeyrings(&KeyringSearch{Name: optional.Of("one")}).ListAll(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotMethod != http.MethodGet {
		t.Errorf("expected GET, got %s", gotMethod)
	}
	if gotPath != "/keyrings" {
		t.Errorf("expected path /keyrings, got %s", gotPath)
	}
	if gotQuery.Get("sliced") != "true" {
		t.Errorf("expected sliced=true, got %q", gotQuery.Get("sliced"))
	}
	if gotQuery.Get("name") != "one" {
		t.Errorf("expected search field name=one, got %q", gotQuery.Get("name"))
	}
	if len(summaries) != 1 || summaries[0].Name.Or("") != "one" {
		t.Errorf("response not decoded into KeyringSummary: %+v", summaries)
	}
}

func TestGetKeyringRequestShape(t *testing.T) {
	var gotMethod, gotPath string
	client := newKeyringTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		if _, err := io.WriteString(w, `{"id":"ydid:keyring:1","name":"my-keyring"}`); err != nil {
			t.Errorf("failed to write response: %v", err)
		}
	})

	keyring, err := client.GetKeyring(context.Background(), "my-keyring")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotMethod != http.MethodGet {
		t.Errorf("expected GET, got %s", gotMethod)
	}
	if gotPath != "/keyrings/my-keyring" {
		t.Errorf("expected path /keyrings/my-keyring, got %s", gotPath)
	}
	if keyring.Name.Or("") != "my-keyring" {
		t.Errorf("expected name my-keyring, got %s", keyring.Name.Or(""))
	}
}
