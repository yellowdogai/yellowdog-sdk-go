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
	awsInstances := NewAwsInstancesComputeSource()
	awsInstances.Name = "aws-src"
	awsFleet := NewAwsFleetComputeSource()
	awsFleet.Name = "aws-fleet-src"
	azureInstances := NewAzureInstancesComputeSource()
	azureInstances.Name = "azure-src"
	azureScaleSet := NewAzureScaleSetComputeSource()
	azureScaleSet.Name = "azure-scaleset-src"
	gceInstances := NewGceInstancesComputeSource()
	gceInstances.Name = "gce-src"
	gceInstanceGroup := NewGceInstanceGroupComputeSource()
	gceInstanceGroup.Name = "gce-group-src"
	ociInstances := NewOciInstancesComputeSource()
	ociInstances.Name = "oci-src"
	ociInstancePool := NewOciInstancePoolComputeSource()
	ociInstancePool.Name = "oci-pool-src"
	simulator := NewSimulatorComputeSource()
	simulator.Name = "sim-src"

	cases := []struct {
		name   string
		source ComputeSource
	}{
		{"AwsInstances", awsInstances},
		{"AwsFleet", awsFleet},
		{"AzureInstances", azureInstances},
		{"AzureScaleSet", azureScaleSet},
		{"GceInstances", gceInstances},
		{"GceInstanceGroup", gceInstanceGroup},
		{"OciInstances", ociInstances},
		{"OciInstancePool", ociInstancePool},
		{"Simulator", simulator},
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
			if discriminatorOf(decoded) != discriminatorOf(c.source) {
				t.Errorf("discriminator mismatch: got %q, want %q", discriminatorOf(decoded), discriminatorOf(c.source))
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
		{"Simulator", &SimulatorInstance{Type: "co.yellowdog.platform.model.SimulatorInstance", Id: optional.Of(id)}},
		{"Aws", &AwsInstance{Type: "co.yellowdog.platform.model.AwsInstance", Id: optional.Of(id)}},
		{"Azure", &AzureInstance{Type: "co.yellowdog.platform.model.AzureInstance", Id: optional.Of(id)}},
		{"Gce", &GceInstance{Type: "co.yellowdog.platform.model.GceInstance", Id: optional.Of(id)}},
		{"Oci", &OciInstance{Type: "co.yellowdog.platform.model.OciInstance", Id: optional.Of(id)}},
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
			if discriminatorOf(decoded) != discriminatorOf(c.instance) {
				t.Errorf("discriminator mismatch: got %q, want %q", discriminatorOf(decoded), discriminatorOf(c.instance))
			}
			if decoded.GetId() != id {
				t.Errorf("id mismatch: got %+v, want %+v", decoded.GetId(), id)
			}
		})
	}
}

// discriminatorOf reads the Type field a polymorphic struct carries its discriminator in, which
// is what is written to and read back from the JSON.
func discriminatorOf(model any) string {
	return reflect.ValueOf(model).Elem().FieldByName("Type").String()
}
