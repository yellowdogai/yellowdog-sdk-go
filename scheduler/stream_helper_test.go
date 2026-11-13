package scheduler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/yellowdogai/yellowdog-sdk-go/internal"
	"github.com/yellowdogai/yellowdog-sdk-go/model"
	"github.com/yellowdogai/yellowdog-sdk-go/optional"
)

// writeStream serves an event stream that emits frames then holds the
// connection open, as the platform does.
func writeStream(t *testing.T, w http.ResponseWriter, r *http.Request, frames []struct{ Type, Data string }) {
	t.Helper()
	w.Header().Set("Content-Type", "text/event-stream")
	w.WriteHeader(http.StatusOK)
	f, ok := w.(http.Flusher)
	if !ok {
		t.Error("ResponseWriter is not a Flusher")
		return
	}
	f.Flush()
	for _, frame := range frames {
		if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", frame.Type, frame.Data); err != nil && r.Context().Err() == nil {
			t.Errorf("failed to write event: %v", err)
		}
		f.Flush()
	}
	<-r.Context().Done()
}

// The by-id count proves the primer read the current state once: the COMPLETED
// status can only have come off the stream.
func TestWorkRequirementHelper_WaitForStatus_StreamsWithoutPolling(t *testing.T) {
	var byIdCalls atomic.Int32
	client := newTestWorkClient(t, func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/updates") {
			byIdCalls.Add(1)
			writeWorkRequirement(t, w, "req-1", model.WorkRequirementStatusRunning)
			return
		}
		b, err := marshalWorkRequirement(t, "req-1", model.WorkRequirementStatusCompleted)
		if err != nil {
			return
		}
		writeStream(t, w, r, []struct{ Type, Data string }{
			{internal.EventEntityStable, ""},
			{internal.EventEntityUpdated, string(b)},
		})
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	helper := NewWorkRequirementHelper(client, workRequirementWith("req-1", model.WorkRequirementStatusRunning))

	result, err := helper.WaitForStatus(ctx, model.WorkRequirementStatusCompleted)
	if err != nil {
		t.Fatalf("WaitForStatus: %v", err)
	}
	if result.Status != optional.Of(model.WorkRequirementStatusCompleted) {
		t.Errorf("status = %v, want COMPLETED", result.Status.Or(""))
	}
	if got := byIdCalls.Load(); got > 1 {
		t.Errorf("by-id endpoint called %d times; expected at most the primer's single read", got)
	}
}

func TestWorkRequirementHelper_EntityFinalisedEndsTheWait(t *testing.T) {
	client := newTestWorkClient(t, func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/updates") {
			writeWorkRequirement(t, w, "req-1", model.WorkRequirementStatusRunning)
			return
		}
		b, err := marshalWorkRequirement(t, "req-1", model.WorkRequirementStatusCancelled)
		if err != nil {
			return
		}
		writeStream(t, w, r, []struct{ Type, Data string }{
			{internal.EventEntityFinalised, string(b)},
		})
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	start := time.Now()
	_, err := NewWorkRequirementHelper(client, workRequirementWith("req-1", model.WorkRequirementStatusRunning)).
		WaitForStatus(ctx, model.WorkRequirementStatusCompleted)

	if !errors.Is(err, internal.ErrEntityFinalised) {
		t.Fatalf("got %v, want ErrEntityFinalised", err)
	}
	if elapsed := time.Since(start); elapsed > 3*time.Second {
		t.Errorf("took %v; a finalised entity should end the wait immediately", elapsed)
	}
}

// A failing stream must be reported rather than quietly replaced by polling.
// The by-id endpoint reports RUNNING once and COMPLETED thereafter, so the
// primer's single read cannot satisfy the wait but a polling loop would, which
// makes a silent fallback show up as an unexpected success.
func TestWorkRequirementHelper_StreamFailureIsReportedNotPolled(t *testing.T) {
	var byIdCalls atomic.Int32
	client := newTestWorkClient(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/updates") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		status := model.WorkRequirementStatusCompleted
		if byIdCalls.Add(1) == 1 {
			status = model.WorkRequirementStatusRunning
		}
		writeWorkRequirement(t, w, "req-1", status)
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := NewWorkRequirementHelper(client, workRequirementWith("req-1", model.WorkRequirementStatusRunning)).
		WaitForStatus(ctx, model.WorkRequirementStatusCompleted)

	if err == nil {
		t.Fatal("expected the 404 on /updates to be reported, but the wait succeeded")
	}
	if got := byIdCalls.Load(); got > 1 {
		t.Errorf("by-id endpoint called %d times; a broken stream must not start a polling loop", got)
	}
}

func TestWorkerPoolHelper_StreamsWithoutPollingAndHonoursFinalised(t *testing.T) {
	t.Run("streams to a match", func(t *testing.T) {
		client := newTestWorkerPoolClient(t, func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasSuffix(r.URL.Path, "/updates") {
				if _, err := w.Write(configuredWorkerPoolJSON(t, "PENDING", "pool-1")); err != nil {
					t.Errorf("failed to write response: %v", err)
				}
				return
			}
			writeStream(t, w, r, []struct{ Type, Data string }{
				{internal.EventEntityStable, ""},
				{internal.EventEntityUpdated, string(configuredWorkerPoolJSON(t, "RUNNING", "pool-1"))},
			})
		})

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		pool := &model.ConfiguredWorkerPool{
			Id:     optional.Of("pool-1"),
			Status: optional.Of(model.WorkerPoolStatus("PENDING")),
		}

		result, err := NewWorkerPoolHelper(client, pool).
			WaitForStatus(ctx, "RUNNING")
		if err != nil {
			t.Fatalf("WaitForStatus: %v", err)
		}
		if s := result.GetStatus(); s != optional.Of(model.WorkerPoolStatus("RUNNING")) {
			t.Errorf("status = %v, want RUNNING", s.Or(""))
		}
	})

	t.Run("finalised ends the wait", func(t *testing.T) {
		client := newTestWorkerPoolClient(t, func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasSuffix(r.URL.Path, "/updates") {
				if _, err := w.Write(configuredWorkerPoolJSON(t, "PENDING", "pool-1")); err != nil {
					t.Errorf("failed to write response: %v", err)
				}
				return
			}
			writeStream(t, w, r, []struct{ Type, Data string }{
				{internal.EventEntityFinalised, string(configuredWorkerPoolJSON(t, "SHUTDOWN", "pool-1"))},
			})
		})

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		pool := &model.ConfiguredWorkerPool{
			Id:     optional.Of("pool-1"),
			Status: optional.Of(model.WorkerPoolStatus("PENDING")),
		}

		start := time.Now()
		_, err := NewWorkerPoolHelper(client, pool).
			WaitForStatus(ctx, "RUNNING")

		if !errors.Is(err, internal.ErrEntityFinalised) {
			t.Fatalf("got %v, want ErrEntityFinalised", err)
		}
		if elapsed := time.Since(start); elapsed > 3*time.Second {
			t.Errorf("took %v; a finalised entity should end the wait immediately", elapsed)
		}
	})
}
