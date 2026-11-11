package internal

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"slices"
	"testing"
	"time"

	"github.com/yellowdogai/yellowdog-sdk-go/optional"
	"github.com/yellowdogai/yellowdog-sdk-go/pagination"
)

// newSlicedServer serves two slices, {"a","b"} then {"c"} for slice ID "s2", and fails any
// other slice ID as the platform does, with a 500 and no message. It records each request's
// query.
func newSlicedServer(t *testing.T) (*HTTPClient, *[]url.Values) {
	var queries []url.Values
	client := newTestHTTPClient(t, func(w http.ResponseWriter, r *http.Request) {
		queries = append(queries, r.URL.Query())
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Query().Get("sliceId") {
		case "":
			if _, err := io.WriteString(w, `{"items":["a","b"],"nextSliceId":"s2"}`); err != nil {
				t.Errorf("failed to write response: %v", err)
			}
		case "s2":
			if _, err := io.WriteString(w, `{"items":["c"]}`); err != nil {
				t.Errorf("failed to write response: %v", err)
			}
		default:
			w.WriteHeader(http.StatusInternalServerError)
			if _, err := io.WriteString(w, `{"errorType":"InternalServerException"}`); err != nil {
				t.Errorf("failed to write response: %v", err)
			}
		}
	})
	return client, &queries
}

func TestSearchClientSlice(t *testing.T) {
	client, queries := newSlicedServer(t)
	search := NewSearchClient[string](client, "/things", nil, Param{Name: "summaries", Value: "true"})

	slice, err := search.Slice(context.Background(), pagination.SliceReference{
		SliceId: optional.Of("s2"),
		Size:    optional.Of(5),
	})
	if err != nil {
		t.Fatalf("Slice: %v", err)
	}
	if !slices.Equal(slice.Items, []string{"c"}) {
		t.Errorf("Expected items [c], got %v", slice.Items)
	}
	if _, ok := slice.NextSliceId.Get(); ok {
		t.Errorf("Expected no nextSliceId, got %v", slice.NextSliceId)
	}

	q := (*queries)[0]
	for name, want := range map[string]string{"summaries": "true", "sliceId": "s2", "size": "5"} {
		if got := q.Get(name); got != want {
			t.Errorf("Expected %s=%s, got %s=%s", name, want, name, got)
		}
	}
	// sliced=true is sent only by the routes that pass it as a Param.
	if q.Has("sliced") {
		t.Errorf("Expected no sliced parameter, got sliced=%s", q.Get("sliced"))
	}
}

func TestSearchClientSliceZeroReference(t *testing.T) {
	client, queries := newSlicedServer(t)

	slice, err := NewSearchClient[string](client, "/things", nil).Slice(context.Background(), pagination.SliceReference{})
	if err != nil {
		t.Fatalf("Slice: %v", err)
	}
	if next, _ := slice.NextSliceId.Get(); next != "s2" {
		t.Errorf("Expected nextSliceId s2, got %q", next)
	}
	q := (*queries)[0]
	if q.Has("sliceId") || q.Has("size") {
		t.Errorf("Expected no sliceId or size for the zero SliceReference, got %v", q)
	}
}

type testShape interface{ Kind() string }

type testSquare struct{ Side int }

func (testSquare) Kind() string { return "square" }

func TestSearchClientSlicePolymorphic(t *testing.T) {
	client := newTestHTTPClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if _, err := io.WriteString(w, `{"items":[{"side":2},{"side":3}],"nextSliceId":"s2"}`); err != nil {
			t.Errorf("failed to write response: %v", err)
		}
	})
	unmarshal := func(raw []byte) (testShape, error) {
		var square testSquare
		err := json.Unmarshal(raw, &square)
		return square, err
	}

	slice, err := NewPolymorphicSearchClient[testShape](client, "/shapes", nil, unmarshal).
		Slice(context.Background(), pagination.SliceReference{})
	if err != nil {
		t.Fatalf("Slice: %v", err)
	}
	want := []testShape{testSquare{Side: 2}, testSquare{Side: 3}}
	if !slices.Equal(slice.Items, want) {
		t.Errorf("Expected items %v, got %v", want, slice.Items)
	}
	if next, _ := slice.NextSliceId.Get(); next != "s2" {
		t.Errorf("Expected nextSliceId s2, got %q", next)
	}
}

func TestSearchClientListAll(t *testing.T) {
	client, queries := newSlicedServer(t)

	items, err := NewSearchClient[string](client, "/things", nil).ListAll(context.Background())
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if !slices.Equal(items, []string{"a", "b", "c"}) {
		t.Errorf("Expected items [a b c], got %v", items)
	}
	if len(*queries) != 2 {
		t.Errorf("Expected 2 requests, got %d", len(*queries))
	}
}

func TestSearchClientIterateStopsEarly(t *testing.T) {
	client, queries := newSlicedServer(t)

	var items []string
	for item, err := range NewSearchClient[string](client, "/things", nil).Iterate(context.Background()) {
		if err != nil {
			t.Fatalf("Iterate: %v", err)
		}
		items = append(items, item)
		break
	}
	if !slices.Equal(items, []string{"a"}) {
		t.Errorf("Expected items [a], got %v", items)
	}
	// Breaking inside the first slice must not retrieve the second.
	if len(*queries) != 1 {
		t.Errorf("Expected 1 request, got %d", len(*queries))
	}
}

func TestSearchClientIterateYieldsError(t *testing.T) {
	client := newTestHTTPClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	var count int
	var gotErr error
	for _, err := range NewSearchClient[string](client, "/things", nil).Iterate(context.Background()) {
		count++
		gotErr = err
	}
	if count != 1 || gotErr == nil {
		t.Fatalf("Expected a single error, got %d iterations and err=%v", count, gotErr)
	}
	var apiErr *APIError
	if !errors.As(gotErr, &apiErr) {
		t.Errorf("Expected an *APIError, got %T", gotErr)
	}
}

func TestAddSearchParams_WithPointerString(t *testing.T) {
	type TestSearch struct {
		Name      *string `json:"name,omitempty"`
		Count     *int    `json:"count,omitempty"`
		Active    *bool   `json:"active,omitempty"`
		PlainText string  `json:"plainText,omitempty"`
	}

	name := "test-namespace"
	count := 42
	active := true

	search := &TestSearch{
		Name:      &name,
		Count:     &count,
		Active:    &active,
		PlainText: "plain",
	}

	client := &SearchClient[any]{
		search: search,
	}

	params := url.Values{}
	client.addSearchParams(params, search)

	if params.Get("name") != "test-namespace" {
		t.Errorf("Expected name=test-namespace, got name=%s", params.Get("name"))
	}

	if params.Get("count") != "42" {
		t.Errorf("Expected count=42, got count=%s", params.Get("count"))
	}

	if params.Get("active") != "true" {
		t.Errorf("Expected active=true, got active=%s", params.Get("active"))
	}

	if params.Get("plainText") != "plain" {
		t.Errorf("Expected plainText=plain, got plainText=%s", params.Get("plainText"))
	}
}

func TestAddSearchParams_WithNilPointers(t *testing.T) {
	type TestSearch struct {
		Name   *string `json:"name,omitempty"`
		Count  *int    `json:"count,omitempty"`
		Active *bool   `json:"active,omitempty"`
	}

	search := &TestSearch{
		Name:   nil,
		Count:  nil,
		Active: nil,
	}

	client := &SearchClient[any]{
		search: search,
	}

	params := url.Values{}
	client.addSearchParams(params, search)

	if params.Get("name") != "" {
		t.Errorf("Expected name to be empty, got name=%s", params.Get("name"))
	}

	if params.Get("count") != "" {
		t.Errorf("Expected count to be empty, got count=%s", params.Get("count"))
	}

	if params.Get("active") != "" {
		t.Errorf("Expected active to be empty, got active=%s", params.Get("active"))
	}
}

func TestAddSearchParams_WithOptionals(t *testing.T) {
	type Range struct {
		Min int `json:"min"`
		Max int `json:"max"`
	}
	type SortDirection string
	type TestSearch struct {
		Namespace     optional.Optional[string]        `json:"namespace,omitzero"`
		Size          optional.Optional[int]           `json:"size,omitzero"`
		Deletable     optional.Optional[bool]          `json:"deletable,omitzero"`
		SortDirection optional.Optional[SortDirection] `json:"sortDirection,omitzero"`
		CreatedTime   optional.Optional[time.Time]     `json:"createdTime,omitzero"`
		AllowedHours  optional.Optional[Range]         `json:"allowedHours,omitzero"`
		Unset         optional.Optional[string]        `json:"unset,omitzero"`
	}

	search := &TestSearch{
		Namespace:     optional.Of("my-namespace"),
		Size:          optional.Of(10),
		Deletable:     optional.Of(false),
		SortDirection: optional.Of(SortDirection("ASC")),
		CreatedTime:   optional.Of(time.Date(2026, 9, 10, 11, 30, 0, 0, time.UTC)),
		AllowedHours:  optional.Of(Range{Min: 1, Max: 5}),
	}

	client := &SearchClient[any]{search: search}

	params := url.Values{}
	client.addSearchParams(params, search)

	expected := map[string]string{
		"namespace":        "my-namespace",
		"size":             "10",
		"deletable":        "false",
		"sortDirection":    "ASC",
		"createdTime":      "2026-09-10T11:30:00Z",
		"allowedHours.min": "1",
		"allowedHours.max": "5",
	}
	for name, want := range expected {
		if got := params.Get(name); got != want {
			t.Errorf("Expected %s=%s, got %s=%s", name, want, name, got)
		}
	}

	// An absent optional contributes no parameter, rather than an empty one.
	if _, present := params["unset"]; present {
		t.Errorf("Expected unset to be absent, got unset=%s", params.Get("unset"))
	}
}

func TestAddSearchParams_WithTimeAndSlice(t *testing.T) {
	type TestSearch struct {
		FromTime   time.Time `json:"fromTime"`
		UntilTime  time.Time `json:"untilTime"`
		Namespaces []string  `json:"namespaces,omitempty"`
	}

	search := &TestSearch{
		FromTime:   time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
		Namespaces: []string{"one", "two"},
	}

	client := &SearchClient[any]{search: search}

	params := url.Values{}
	client.addSearchParams(params, search)

	if got := params.Get("fromTime"); got != "2026-09-01T00:00:00Z" {
		t.Errorf("Expected fromTime=2026-09-01T00:00:00Z, got fromTime=%s", got)
	}

	// The zero time is indistinguishable from unset, so it is not sent.
	if _, present := params["untilTime"]; present {
		t.Errorf("Expected untilTime to be absent, got untilTime=%s", params.Get("untilTime"))
	}

	if got := params["namespaces"]; len(got) != 2 || got[0] != "one" || got[1] != "two" {
		t.Errorf("Expected namespaces=[one two], got namespaces=%v", got)
	}
}
