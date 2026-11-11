package compute

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/yellowdogai/yellowdog-sdk-go/model"
	"github.com/yellowdogai/yellowdog-sdk-go/optional"
)

// These tests cover the logic computeClientProxy carries either side of a request: projecting an
// object form onto the scalar the route takes, and surfacing a failed request as an error. Paths
// and verbs are not asserted, because such a test would only compare a literal with its own copy.

func idOf(instanceId, sourceId string) model.InstanceId {
	return model.InstanceId{SourceId: sourceId, InstanceId: instanceId}
}

func instanceWithId(instanceId string) model.Instance {
	inst := &model.SimulatorInstance{Type: "co.yellowdog.platform.model.SimulatorInstance"}
	inst.Id = optional.Of(idOf(instanceId, "source-1"))
	return inst
}

func TestInstanceIdsOf(t *testing.T) {
	instances := []model.Instance{instanceWithId("i-1"), instanceWithId("i-2")}
	ids := instanceIdsOf(instances)
	if len(ids) != 2 {
		t.Fatalf("got %d ids, want 2", len(ids))
	}
	if ids[0].InstanceId != "i-1" || ids[1].InstanceId != "i-2" {
		t.Errorf("got %+v", ids)
	}
}

// The instance transitions take instances at the public API and ids on the wire. The projection
// happens in the proxy, so the request body is the only place it is observable.
func TestStopInstancesSendsExtractedIds(t *testing.T) {
	var gotBody []*model.InstanceId
	client := newTestComputeClient(t, func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Errorf("failed to decode body: %v", err)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	req := &model.ComputeRequirement{Id: optional.Of("req-1")}
	instances := []model.Instance{instanceWithId("i-1")}
	if err := client.StopInstances(context.Background(), req, instances); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(gotBody) != 1 || gotBody[0].InstanceId != "i-1" {
		t.Errorf("got body %+v, want the ids of the given instances", gotBody)
	}
}

func TestTransitionErrorPropagates(t *testing.T) {
	client := newTestComputeClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
	})

	if _, err := client.StartComputeRequirementById(context.Background(), "req-1"); err == nil {
		t.Fatal("expected an error when the transition request fails")
	}
}
