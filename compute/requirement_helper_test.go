package compute

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/yellowdogai/yellowdog-sdk-go/internal"
	"github.com/yellowdogai/yellowdog-sdk-go/internal/testutil"
	"github.com/yellowdogai/yellowdog-sdk-go/model"
	"github.com/yellowdogai/yellowdog-sdk-go/optional"
)

func waitForRunning(ctx context.Context, h *RequirementHelper) (*model.ComputeRequirement, error) {
	return h.WaitForStatus(ctx, model.ComputeRequirementStatusRunning)
}

func waitForTermination(ctx context.Context, h *RequirementHelper) (*model.ComputeRequirement, error) {
	return h.WaitForStatus(ctx, model.ComputeRequirementStatusTerminated)
}

func newTestComputeClient(t *testing.T, handler http.HandlerFunc) *ComputeClient {
	t.Helper()
	return NewComputeClient(testutil.NewHTTPClient(t, handler))
}

func computeRequirementWith(id string, status model.ComputeRequirementStatus) *model.ComputeRequirement {
	return &model.ComputeRequirement{
		Id:     optional.Of(id),
		Status: optional.Of(status),
		ProvisionStrategy: &model.SingleSourceProvisionStrategy{
			Type: "co.yellowdog.platform.model.SingleSourceProvisionStrategy",
		},
	}
}

func writeComputeRequirement(t *testing.T, w http.ResponseWriter, id string, status model.ComputeRequirementStatus) {
	t.Helper()
	b, err := json.Marshal(computeRequirementWith(id, status))
	if err != nil {
		t.Errorf("failed to marshal fixture: %v", err)
		return
	}
	if _, err := w.Write(b); err != nil {
		t.Errorf("failed to write fixture: %v", err)
	}
}

// sseRequirementServer serves the requirement's update stream, and the by-id
// endpoint the primer reads. byIdCalls distinguishes the primer's single read
// from a polling loop.
type sseRequirementServer struct {
	client    *ComputeClient
	byIdCalls atomic.Int32
}

func newSSERequirementServer(t *testing.T, id string, events []struct {
	Type   string
	Status model.ComputeRequirementStatus
}) *sseRequirementServer {
	t.Helper()
	s := &sseRequirementServer{}
	updatesPath := "/compute/requirements/" + id + "/updates"

	s.client = newTestComputeClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != updatesPath {
			// The primer's single read of current state.
			s.byIdCalls.Add(1)
			writeComputeRequirement(t, w, id, model.ComputeRequirementStatusNew)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		f := w.(http.Flusher)
		f.Flush()

		for _, e := range events {
			b, err := json.Marshal(computeRequirementWith(id, e.Status))
			if err != nil {
				t.Errorf("failed to marshal fixture: %v", err)
				return
			}
			if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", e.Type, b); err != nil && r.Context().Err() == nil {
				t.Errorf("failed to write event: %v", err)
			}
			f.Flush()
		}
		<-r.Context().Done()
	})
	return s
}

// The by-id count proves the primer read the current state once and that
// nothing else re-read it: the RUNNING status can only have come off the stream.
func TestRequirementHelper_WaitForRunning_StreamsWithoutPolling(t *testing.T) {
	server := newSSERequirementServer(t, "req-sse", []struct {
		Type   string
		Status model.ComputeRequirementStatus
	}{
		{internal.EventEntityUpdated, model.ComputeRequirementStatusNew},
		{internal.EventEntityUpdated, model.ComputeRequirementStatusProvisioning},
		{internal.EventEntityUpdated, model.ComputeRequirementStatusRunning},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	helper := server.client.Helper(computeRequirementWith("req-sse", model.ComputeRequirementStatusNew))

	req, err := waitForRunning(ctx, helper)
	if err != nil {
		t.Fatalf("WaitForRunning: %v", err)
	}
	if req.Status != optional.Of(model.ComputeRequirementStatusRunning) {
		t.Errorf("status = %v, want RUNNING", req.Status.Or(""))
	}
	if got := server.byIdCalls.Load(); got > 1 {
		t.Errorf("by-id endpoint called %d times; expected at most the primer's single read", got)
	}
}

// Waiting for RUNNING on a requirement that terminates must fail at once
// rather than block until the caller's context expires.
func TestRequirementHelper_EntityFinalisedEndsTheWait(t *testing.T) {
	server := newSSERequirementServer(t, "req-fin", []struct {
		Type   string
		Status model.ComputeRequirementStatus
	}{
		{internal.EventEntityFinalised, model.ComputeRequirementStatusTerminated},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	start := time.Now()
	_, err := waitForRunning(ctx, server.client.Helper(computeRequirementWith("req-fin", model.ComputeRequirementStatusNew)))

	if !errors.Is(err, internal.ErrEntityFinalised) {
		t.Fatalf("got %v, want ErrEntityFinalised", err)
	}
	if elapsed := time.Since(start); elapsed > 3*time.Second {
		t.Errorf("took %v; a finalised entity should end the wait immediately", elapsed)
	}
}

// A failing stream must be reported rather than quietly replaced by polling.
// The by-id endpoint reports NEW once and RUNNING thereafter, so the primer's
// single read cannot satisfy the wait but a polling loop would, which makes a
// silent fallback show up as an unexpected success.
func TestRequirementHelper_StreamFailureIsReportedNotPolled(t *testing.T) {
	var byIdCalls atomic.Int32
	client := newTestComputeClient(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/updates") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		status := model.ComputeRequirementStatusRunning
		if byIdCalls.Add(1) == 1 {
			status = model.ComputeRequirementStatusNew
		}
		writeComputeRequirement(t, w, "req-broken", status)
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	start := time.Now()
	_, err := waitForRunning(ctx, client.Helper(computeRequirementWith("req-broken", model.ComputeRequirementStatusNew)))

	if err == nil {
		t.Fatal("expected the 404 on /updates to be reported, but the wait succeeded")
	}
	if elapsed := time.Since(start); elapsed > 3*time.Second {
		t.Errorf("took %v; a 404 should be reported immediately", elapsed)
	}
	t.Logf("reported after %v: %v", time.Since(start).Round(time.Millisecond), err)

	if got := byIdCalls.Load(); got > 1 {
		t.Errorf("by-id endpoint called %d times; a broken stream must not start a polling loop", got)
	}
}

func TestRequirementHelper_WaitForTermination_StreamsUntilTerminated(t *testing.T) {
	server := newSSERequirementServer(t, "req-term", []struct {
		Type   string
		Status model.ComputeRequirementStatus
	}{
		{internal.EventEntityUpdated, model.ComputeRequirementStatusTerminating},
		{internal.EventEntityUpdated, model.ComputeRequirementStatusTerminated},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := waitForTermination(ctx, server.client.Helper(computeRequirementWith("req-term", model.ComputeRequirementStatusRunning)))
	if err != nil {
		t.Fatalf("WaitForTermination: %v", err)
	}
	if result.Status != optional.Of(model.ComputeRequirementStatusTerminated) {
		t.Errorf("status = %v, want TERMINATED", result.Status.Or(""))
	}
}
