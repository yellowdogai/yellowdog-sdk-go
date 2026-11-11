package scheduler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/yellowdogai/yellowdog-sdk-go/internal/testutil"
	"github.com/yellowdogai/yellowdog-sdk-go/model"
	"github.com/yellowdogai/yellowdog-sdk-go/optional"
)

func waitForCompletion(ctx context.Context, h *WorkRequirementHelper) (*model.WorkRequirement, error) {
	return h.WaitUntil(ctx, func(req *model.WorkRequirement) bool {
		status, ok := req.Status.Get()
		return ok && status.Finished()
	})
}

func newTestWorkClient(t *testing.T, handler http.HandlerFunc) *WorkClient {
	t.Helper()
	return NewWorkClient(testutil.NewHTTPClient(t, handler))
}

func workRequirementWith(id string, status model.WorkRequirementStatus) *model.WorkRequirement {
	req := &model.WorkRequirement{Name: "job"}
	req.Id = optional.Of(id)
	req.Status = optional.Of(status)
	return req
}

func marshalWorkRequirement(t *testing.T, id string, status model.WorkRequirementStatus) ([]byte, error) {
	t.Helper()
	b, err := json.Marshal(workRequirementWith(id, status))
	if err != nil {
		t.Errorf("failed to marshal fixture: %v", err)
	}
	return b, err
}

func writeWorkRequirement(t *testing.T, w http.ResponseWriter, id string, status model.WorkRequirementStatus) {
	t.Helper()
	b, err := json.Marshal(workRequirementWith(id, status))
	if err != nil {
		t.Errorf("failed to marshal fixture: %v", err)
		return
	}
	if _, err := w.Write(b); err != nil {
		t.Errorf("failed to write fixture: %v", err)
	}
}

// holdOpenStream serves an update stream that sends nothing, leaving the
// primer's read of current state as the only thing that can satisfy the wait.
func holdOpenStream(t *testing.T, w http.ResponseWriter, r *http.Request) {
	t.Helper()
	writeStream(t, w, r, nil)
}

func TestWaitForCompletion_MatchesTerminalStatuses(t *testing.T) {
	for _, status := range []model.WorkRequirementStatus{
		model.WorkRequirementStatusCompleted,
		model.WorkRequirementStatusCancelled,
		model.WorkRequirementStatusFailed,
	} {
		t.Run(string(status), func(t *testing.T) {
			client := newTestWorkClient(t, func(w http.ResponseWriter, r *http.Request) {
				if strings.HasSuffix(r.URL.Path, "/updates") {
					holdOpenStream(t, w, r)
					return
				}
				writeWorkRequirement(t, w, "req-1", status)
			})

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			result, err := waitForCompletion(ctx, NewWorkRequirementHelper(client, workRequirementWith("req-1", status)))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Status != optional.Of(status) {
				t.Errorf("got %v, want %v", result.Status.Or(""), status)
			}
		})
	}
}

func TestWaitForCompletion_RunningIsNotTerminal(t *testing.T) {
	client := newTestWorkClient(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/updates") {
			holdOpenStream(t, w, r)
			return
		}
		writeWorkRequirement(t, w, "req-1", model.WorkRequirementStatusRunning)
	})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := waitForCompletion(ctx, NewWorkRequirementHelper(client, workRequirementWith("req-1", model.WorkRequirementStatusRunning)))
	if err == nil {
		t.Fatal("expected an error/timeout since RUNNING should never satisfy waitForCompletion")
	}
}

func TestHelper_And_HelperById(t *testing.T) {
	client := newTestWorkClient(t, func(w http.ResponseWriter, r *http.Request) {
		writeWorkRequirement(t, w, "req-1", model.WorkRequirementStatusRunning)
	})

	req := workRequirementWith("req-1", model.WorkRequirementStatusRunning)
	if helper := client.Helper(req); helper == nil {
		t.Fatal("expected a non-nil helper from Helper()")
	}

	helper, err := client.HelperById(context.Background(), "req-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if helper.requirement.Id.Or("") != "req-1" {
		t.Errorf("got id %q, want req-1", helper.requirement.Id.Or(""))
	}
}

func TestHelperById_PropagatesFetchError(t *testing.T) {
	client := newTestWorkClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	_, err := client.HelperById(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected an error when the work requirement cannot be fetched")
	}
}

func TestCancelWorkRequirementById_SetsAbortQueryParam(t *testing.T) {
	for _, abort := range []bool{true, false} {
		t.Run(map[bool]string{true: "abort", false: "no-abort"}[abort], func(t *testing.T) {
			var gotPath, gotMethod string
			client := newTestWorkClient(t, func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path + "?" + r.URL.RawQuery
				gotMethod = r.Method
				writeWorkRequirement(t, w, "req-1", model.WorkRequirementStatusCancelling)
			})

			result, err := client.CancelWorkRequirementByIdWithAbort(context.Background(), "req-1", abort)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotMethod != http.MethodPut {
				t.Errorf("method = %s, want PUT", gotMethod)
			}
			parsed, _ := url.Parse(gotPath)
			if parsed.Path != "/work/requirements/req-1/transition/CANCELLING" {
				t.Errorf("path = %s", parsed.Path)
			}
			wantAbort := "false"
			if abort {
				wantAbort = "true"
			}
			if parsed.Query().Get("abort") != wantAbort {
				t.Errorf("abort query param = %q, want %q", parsed.Query().Get("abort"), wantAbort)
			}
			if result.Id.Or("") != "req-1" {
				t.Errorf("got id %q, want req-1", result.Id.Or(""))
			}
		})
	}
}

func TestCancelWorkRequirement_DelegatesById(t *testing.T) {
	var gotPath string
	client := newTestWorkClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		writeWorkRequirement(t, w, "req-9", model.WorkRequirementStatusCancelling)
	})

	req := workRequirementWith("req-9", model.WorkRequirementStatusRunning)
	if _, err := client.CancelWorkRequirementWithAbort(context.Background(), req, true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotPath != "/work/requirements/req-9/transition/CANCELLING" {
		t.Errorf("path = %s", gotPath)
	}
}

func TestHoldWorkRequirementById_UsesHeldTransition(t *testing.T) {
	var gotPath string
	client := newTestWorkClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		writeWorkRequirement(t, w, "req-1", model.WorkRequirementStatusHeld)
	})

	if _, err := client.HoldWorkRequirementById(context.Background(), "req-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotPath != "/work/requirements/req-1/transition/HELD" {
		t.Errorf("path = %s", gotPath)
	}
}

func TestStartWorkRequirementById_UsesRunningTransition(t *testing.T) {
	var gotPath string
	client := newTestWorkClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		writeWorkRequirement(t, w, "req-1", model.WorkRequirementStatusRunning)
	})

	if _, err := client.StartWorkRequirementById(context.Background(), "req-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotPath != "/work/requirements/req-1/transition/RUNNING" {
		t.Errorf("path = %s", gotPath)
	}
}

func TestFinishWorkRequirementById_UsesFinishingTransition(t *testing.T) {
	var gotPath string
	client := newTestWorkClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		writeWorkRequirement(t, w, "req-1", model.WorkRequirementStatusFinishing)
	})

	if _, err := client.FinishWorkRequirementById(context.Background(), "req-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotPath != "/work/requirements/req-1/transition/FINISHING" {
		t.Errorf("path = %s", gotPath)
	}
}

func TestHoldStartFinishWorkRequirement_DelegateById(t *testing.T) {
	var gotPath string
	client := newTestWorkClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		writeWorkRequirement(t, w, "req-5", model.WorkRequirementStatusRunning)
	})

	req := workRequirementWith("req-5", model.WorkRequirementStatusHeld)

	if _, err := client.HoldWorkRequirement(context.Background(), req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotPath != "/work/requirements/req-5/transition/HELD" {
		t.Errorf("HoldWorkRequirement path = %s", gotPath)
	}

	if _, err := client.StartWorkRequirement(context.Background(), req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotPath != "/work/requirements/req-5/transition/RUNNING" {
		t.Errorf("StartWorkRequirement path = %s", gotPath)
	}

	if _, err := client.FinishWorkRequirement(context.Background(), req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotPath != "/work/requirements/req-5/transition/FINISHING" {
		t.Errorf("FinishWorkRequirement path = %s", gotPath)
	}
}

func TestTransitionErrorPropagates_WorkRequirement(t *testing.T) {
	client := newTestWorkClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
	})

	if _, err := client.StartWorkRequirementById(context.Background(), "req-1"); err == nil {
		t.Fatal("expected an error when the transition request fails")
	}
}

func TestGetWorkReqSubscriptions_CachesPerClient(t *testing.T) {
	client1 := &WorkClient{}
	client2 := &WorkClient{}

	mgr1 := getWorkReqSubscriptions(client1)
	mgr1Again := getWorkReqSubscriptions(client1)
	if mgr1 != mgr1Again {
		t.Error("expected the same SubscriptionManager instance to be cached per client")
	}

	mgr2 := getWorkReqSubscriptions(client2)
	if mgr2 == mgr1 {
		t.Error("expected different clients to get different SubscriptionManager instances")
	}
}
