package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/yellowdogai/yellowdog-sdk-go/internal"
	"github.com/yellowdogai/yellowdog-sdk-go/internal/testutil"
	"github.com/yellowdogai/yellowdog-sdk-go/model"
	"github.com/yellowdogai/yellowdog-sdk-go/optional"
	"github.com/yellowdogai/yellowdog-sdk-go/subscription"
)

func newTestWorkerPoolClient(t *testing.T, handler http.HandlerFunc) *WorkerPoolClient {
	t.Helper()
	return NewWorkerPoolClient(testutil.NewHTTPClient(t, handler))
}

func configuredWorkerPoolJSON(t *testing.T, status model.WorkerPoolStatus, id string) []byte {
	t.Helper()
	pool := &model.ConfiguredWorkerPool{
		Id:     optional.Of(id),
		Status: optional.Of(status),
	}
	b, err := json.Marshal(pool)
	if err != nil {
		t.Fatalf("failed to marshal test fixture: %v", err)
	}
	return b
}

func provisionedWorkerPoolJSON(t *testing.T, id string) []byte {
	t.Helper()
	pool := &model.ProvisionedWorkerPool{
		Id: optional.Of(id),
	}
	b, err := json.Marshal(pool)
	if err != nil {
		t.Fatalf("failed to marshal test fixture: %v", err)
	}
	return b
}

func provisionedHelperById(ctx context.Context, c *WorkerPoolClient, id string) (*WorkerPoolHelper[model.ProvisionedWorkerPool, *model.ProvisionedWorkerPool], error) {
	wp, err := c.GetWorkerPool(ctx, id)
	if err != nil {
		return nil, err
	}
	pool, ok := wp.(*model.ProvisionedWorkerPool)
	if !ok {
		return nil, fmt.Errorf("worker pool %s is not a ProvisionedWorkerPool", id)
	}
	return NewWorkerPoolHelper[model.ProvisionedWorkerPool, *model.ProvisionedWorkerPool](c, pool), nil
}

func waitForWorkers[T any, PT workerPoolConstraint[T]](ctx context.Context, h *WorkerPoolHelper[T, PT], minWorkers int) (PT, error) {
	return h.WaitUntil(ctx, func(pool PT) bool {
		ws, ok := pool.GetWorkerSummary().Get()
		return ok && ws.ClaimedWorkerCount >= minWorkers
	})
}

func TestHelperById_Success(t *testing.T) {
	client := newTestWorkerPoolClient(t, func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write(configuredWorkerPoolJSON(t, model.WorkerPoolStatus("RUNNING"), "pool-1")); err != nil {
			t.Errorf("failed to write response: %v", err)
		}
	})

	helper, err := client.HelperById(context.Background(), "pool-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if helper.pool.GetId() != "pool-1" {
		t.Errorf("got id %q, want pool-1", helper.pool.GetId())
	}
}

func TestHelperById_WrongType(t *testing.T) {
	client := newTestWorkerPoolClient(t, func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write(provisionedWorkerPoolJSON(t, "pool-1")); err != nil {
			t.Errorf("failed to write response: %v", err)
		}
	})

	_, err := client.HelperById(context.Background(), "pool-1")
	if err == nil {
		t.Fatal("expected an error when fetching a ProvisionedWorkerPool via HelperById")
	}
}

func TestProvisionedHelperById_WrongType(t *testing.T) {
	client := newTestWorkerPoolClient(t, func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write(configuredWorkerPoolJSON(t, model.WorkerPoolStatus("RUNNING"), "pool-1")); err != nil {
			t.Errorf("failed to write response: %v", err)
		}
	})

	_, err := provisionedHelperById(context.Background(), client, "pool-1")
	if err == nil {
		t.Fatal("expected an error when fetching a ConfiguredWorkerPool via provisionedHelperById")
	}
}

// The claimed count arrives on the stream, not from a second read of the pool.
func TestWaitForWorkers_StreamsUntilMinimumReached(t *testing.T) {
	client := newTestWorkerPoolClient(t, func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/updates") {
			if _, err := w.Write(workerPoolWithClaimedCount(t, "pool-1", 0)); err != nil {
				t.Errorf("failed to write response: %v", err)
			}
			return
		}
		writeStream(t, w, r, []struct{ Type, Data string }{
			{internal.EventEntityUpdated, string(workerPoolWithClaimedCount(t, "pool-1", 5))},
		})
	})

	helper, err := client.HelperById(context.Background(), "pool-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := waitForWorkers(ctx, helper, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	summary, _ := result.GetWorkerSummary().Get()
	if summary.ClaimedWorkerCount != 5 {
		t.Errorf("got claimed count %d, want 5", summary.ClaimedWorkerCount)
	}
}

func workerPoolWithClaimedCount(t *testing.T, id string, claimed int) []byte {
	t.Helper()
	pool := &model.ConfiguredWorkerPool{
		Id:            optional.Of(id),
		WorkerSummary: optional.Of(model.WorkerSummary{ClaimedWorkerCount: claimed}),
	}
	b, err := json.Marshal(pool)
	if err != nil {
		t.Fatalf("failed to marshal fixture: %v", err)
	}
	return b
}

func TestHelperById_FetchErrorPropagates(t *testing.T) {
	client := newTestWorkerPoolClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	helper, err := client.HelperById(context.Background(), "pool-1")
	if err == nil {
		// Even the initial fetch inside HelperById will fail here, which
		// is itself the behaviour under test.
		t.Fatal("expected an error fetching the pool when the server returns 500")
	}
	_ = helper
}

func TestAddNodeActions_DelegatesToAddNodeActionsById(t *testing.T) {
	var gotPath, gotMethod string
	client := newTestWorkerPoolClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.WriteHeader(http.StatusNoContent)
	})

	pool := &model.ConfiguredWorkerPool{Id: optional.Of("pool-42")}
	err := client.AddNodeActions(context.Background(), pool, &AddNodeActionsRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %s, want POST", gotMethod)
	}
	wantPath := "/workerPools/pool-42/actions"
	if gotPath != wantPath {
		t.Errorf("path = %s, want %s", gotPath, wantPath)
	}
}

func TestSubscriptionManagerCaching(t *testing.T) {
	client := &WorkerPoolClient{}

	mgr1 := getWorkerPoolSubscriptions(client)
	mgr2 := getWorkerPoolSubscriptions(client)
	if mgr1 != mgr2 {
		t.Error("expected the same SubscriptionManager instance to be cached per client")
	}

	otherClient := &WorkerPoolClient{}
	mgr3 := getWorkerPoolSubscriptions(otherClient)
	if mgr3 == mgr1 {
		t.Error("expected different clients to get different SubscriptionManager instances")
	}

}

// The stream is decoded through the WorkerPool registry, as Java's listener receives WorkerPool,
// so a provisioned pool is delivered as itself rather than forced into a ConfiguredWorkerPool.
func TestAddWorkerPoolListener_DeliversEachSubtype(t *testing.T) {
	client := newTestWorkerPoolClient(t, func(w http.ResponseWriter, r *http.Request) {
		writeStream(t, w, r, []struct{ Type, Data string }{
			{internal.EventEntityUpdated, string(provisionedWorkerPoolJSON(t, "pool-1"))},
			{internal.EventEntityUpdated, string(configuredWorkerPoolJSON(t, model.WorkerPoolStatus("RUNNING"), "pool-1"))},
		})
	})

	pools := make(chan model.WorkerPool, 2)
	defer client.AddWorkerPoolListener("pool-1", subscription.SubscriptionEventListener[model.WorkerPool]{
		Updated: func(pool model.WorkerPool) { pools <- pool },
	})()

	for _, want := range []string{"*model.ProvisionedWorkerPool", "*model.ConfiguredWorkerPool"} {
		select {
		case pool := <-pools:
			if got := fmt.Sprintf("%T", pool); got != want {
				t.Errorf("got %s, want %s", got, want)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("timed out waiting for a %s update", want)
		}
	}
}
