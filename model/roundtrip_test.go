package model

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/yellowdogai/yellowdog-sdk-go/optional"
)

func TestTaskDataJSONRoundTrip(t *testing.T) {
	original := &TaskData{
		Inputs: []TaskDataInput{{Source: "s3://bucket/in", Destination: "input.txt"}},
		Outputs: []TaskDataOutput{{
			Source:       "output.txt",
			Destination:  "s3://bucket/out",
			AlwaysUpload: true,
		}},
	}

	b, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded TaskData
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if !reflect.DeepEqual(*original, decoded) {
		t.Errorf("round-trip mismatch:\n original: %+v\n decoded:  %+v", *original, decoded)
	}
}

func TestTaskJSONRoundTrip(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)

	original := &Task{
		Name:      optional.Of("task-1"),
		Tag:       optional.Of("tag-1"),
		TaskType:  optional.Of("docker"),
		Arguments: []string{"--flag", "value"},
	}
	original.Id = optional.Of("task-id-1")
	original.Status = optional.Of(TaskStatusExecuting)
	original.RetryCount = optional.Of(0) // present zero value must survive omitzero
	original.Environment = map[string]string{"KEY": "value"}
	original.StartedTime = optional.Of(now)
	original.Data = optional.Of(TaskData{
		Inputs: []TaskDataInput{
			{Source: "in", Destination: "out"},
		},
	})
	original.Timeout = optional.Of(MustParseDuration("30s"))

	b, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded Task
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if !reflect.DeepEqual(*original, decoded) {
		t.Errorf("round-trip mismatch:\n original: %+v\n decoded:  %+v", *original, decoded)
	}
	if decoded.RetryCount.IsZero() {
		t.Error("present zero RetryCount must survive the round trip as present, not absent")
	}
}

func TestTaskGroupJSONRoundTrip(t *testing.T) {
	original := &TaskGroup{Name: "group-1"}
	original.Id = optional.Of("tg-1")
	original.Priority = 1.5
	original.RunSpecification = RunSpecification{
		TaskTypes:  []string{"docker"},
		MinWorkers: optional.Of(1),
		MaxWorkers: optional.Of(5),
	}
	original.TaskSummary = optional.Of(TaskSummary{
		TaskCount:    3,
		StatusCounts: map[TaskStatus]int{TaskStatusCompleted: 3},
	})
	original.TaskTemplate = optional.Of(TaskTemplate{
		TaskType:    optional.Of("docker"),
		Environment: map[string]string{"A": "B"},
	})

	b, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded TaskGroup
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if !reflect.DeepEqual(*original, decoded) {
		t.Errorf("round-trip mismatch:\n original: %+v\n decoded:  %+v", *original, decoded)
	}
}

func TestWorkRequirementJSONRoundTrip(t *testing.T) {
	group := &TaskGroup{Name: "group-1"}
	group.RunSpecification = RunSpecification{TaskTypes: []string{"docker"}}

	original := &WorkRequirement{
		Name:       "wr-1",
		Namespace:  "ns-1",
		Priority:   2,
		TaskGroups: []TaskGroup{*group},
	}
	original.Id = optional.Of("wr-id-1")
	original.Status = optional.Of(WorkRequirementStatusRunning)

	b, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded WorkRequirement
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if !reflect.DeepEqual(*original, decoded) {
		t.Errorf("round-trip mismatch:\n original: %+v\n decoded:  %+v", *original, decoded)
	}
	if len(decoded.TaskGroups) != 1 || decoded.TaskGroups[0].Name != "group-1" {
		t.Errorf("nested task groups did not survive the round trip: %+v", decoded.TaskGroups)
	}
}

// TestComputeSourceRoundTrip verifies that every registered ComputeSource type
// survives a marshal -> UnmarshalComputeSource round trip through its
// polymorphic discriminator.
func TestComputeSourceRoundTrip(t *testing.T) {
	cases := []struct {
		name   string
		source ComputeSource
	}{
		{"AwsInstances", &AwsInstancesComputeSource{Name: "aws-src"}},
		{"AwsFleet", &AwsFleetComputeSource{Name: "aws-fleet-src"}},
		{"AzureInstances", &AzureInstancesComputeSource{Name: "azure-src"}},
		{"AzureScaleSet", &AzureScaleSetComputeSource{Name: "azure-scaleset-src"}},
		{"GceInstances", &GceInstancesComputeSource{Name: "gce-src"}},
		{"GceInstanceGroup", &GceInstanceGroupComputeSource{Name: "gce-group-src"}},
		{"OciInstances", &OciInstancesComputeSource{Name: "oci-src"}},
		{"OciInstancePool", &OciInstancePoolComputeSource{Name: "oci-pool-src"}},
		{"Simulator", &SimulatorComputeSource{Name: "sim-src"}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b, err := json.Marshal(c.source)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}

			decoded, err := UnmarshalComputeSource(b)
			if err != nil {
				t.Fatalf("UnmarshalComputeSource: %v", err)
			}

			if reflect.TypeOf(decoded) != reflect.TypeOf(c.source) {
				t.Errorf("got type %T, want %T", decoded, c.source)
			}
			if decoded.GetName() != c.source.GetName() {
				t.Errorf("name mismatch: got %q, want %q", decoded.GetName(), c.source.GetName())
			}
		})
	}
}

// TestInstanceRoundTrip verifies that every registered Instance type survives
// a marshal -> UnmarshalInstance round trip through its polymorphic
// discriminator.
func TestInstanceRoundTrip(t *testing.T) {
	id := InstanceId{SourceId: "source-1", InstanceId: "instance-1"}

	cases := []struct {
		name     string
		instance Instance
	}{
		{"Simulator", &SimulatorInstance{Id: optional.Of(id)}},
		{"Aws", &AwsInstance{Id: optional.Of(id)}},
		{"Azure", &AzureInstance{Id: optional.Of(id)}},
		{"Gce", &GceInstance{Id: optional.Of(id)}},
		{"Oci", &OciInstance{Id: optional.Of(id)}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b, err := json.Marshal(c.instance)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}

			decoded, err := UnmarshalInstance(b)
			if err != nil {
				t.Fatalf("UnmarshalInstance: %v", err)
			}

			if reflect.TypeOf(decoded) != reflect.TypeOf(c.instance) {
				t.Errorf("got type %T, want %T", decoded, c.instance)
			}
			if decoded.GetId() != id {
				t.Errorf("id mismatch: got %+v, want %+v", decoded.GetId(), id)
			}
		})
	}
}

// A nil polymorphic field marshals as null, which has to read back as nil rather than fail for
// want of a discriminator.
func TestNilPolymorphicFieldRoundTrip(t *testing.T) {
	for _, value := range []any{&Cluster{}, &ClusterSpec{}, &ComputeRequirement{}, &ComputeSourceTemplate{}} {
		t.Run(reflect.TypeOf(value).Elem().Name(), func(t *testing.T) {
			data, err := json.Marshal(value)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			if err := json.Unmarshal(data, value); err != nil {
				t.Errorf("unmarshal %s: %v", data, err)
			}
		})
	}
}
