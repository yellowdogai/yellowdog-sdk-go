package scheduler_test

import (
	"context"
	"fmt"
	"log"
	"time"

	yellowdog "github.com/yellowdogai/yellowdog-sdk-go"
	"github.com/yellowdogai/yellowdog-sdk-go/model"
	"github.com/yellowdogai/yellowdog-sdk-go/optional"
	"github.com/yellowdogai/yellowdog-sdk-go/scheduler"
)

// ExampleWorkClient demonstrates submitting a work requirement with tasks.
// Tasks are added to a task group after the requirement exists, so the requirement
// is created first and its task group is then passed to AddTasksToTaskGroup.
func ExampleWorkClient() {
	config := yellowdog.DefaultConfig()
	config.APIKey = "your-api-key"
	config.APISecret = "your-api-secret"

	client, err := yellowdog.NewClient(config)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = client.Close() }()

	ctx := context.Background()

	// Define a work requirement with one task group
	workReq := &model.WorkRequirement{
		Name:      "my-batch-job",
		Namespace: "my-namespace",
		TaskGroups: []model.TaskGroup{
			{
				Name: "processing",
				RunSpecification: model.RunSpecification{
					TaskTypes:  []string{"bash"},
					MaxWorkers: optional.Of(4),
					WorkerTags: []string{"my-worker-tag"},
				},
			},
		},
	}

	created, err := client.Work.AddWorkRequirement(ctx, workReq)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Created work requirement: %s (ID: %s)\n", created.Name, created.Id.Or(""))

	// Add tasks to the task group
	tasks := []*model.Task{
		{
			Name:      optional.Of("process-file-1"),
			TaskType:  optional.Of("bash"),
			Arguments: []string{"-c", "echo processing file 1"},
		},
		{
			Name:      optional.Of("process-file-2"),
			TaskType:  optional.Of("bash"),
			Arguments: []string{"-c", "echo processing file 2"},
		},
	}

	added, err := client.Work.AddTasksToTaskGroup(ctx, &created.TaskGroups[0], tasks)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Added %d tasks\n", len(added))

	// Cancel when done (or wait for completion via WorkRequirementHelper)
	_, err = client.Work.CancelWorkRequirementWithAbort(ctx, created, true)
	if err != nil {
		log.Printf("Failed to cancel: %v", err)
	}
}

// ExampleWorkRequirementHelper demonstrates waiting for a work requirement to complete.
// The helper streams live updates from the requirement's update endpoint, so bound
// the wait with a context deadline: a requirement that never completes never returns.
func ExampleWorkRequirementHelper() {
	config := yellowdog.DefaultConfig()
	config.APIKey = "your-api-key"
	config.APISecret = "your-api-secret"

	client, err := yellowdog.NewClient(config)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = client.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	// Fetch a previously created work requirement
	req, err := client.Work.GetWorkRequirementById(ctx, "ydid:workreq:000000:abc123")
	if err != nil {
		log.Fatal(err)
	}

	// Use the helper to wait for the requirement to finish
	helper := client.Work.Helper(req)
	final, err := helper.WaitForStatus(ctx, model.WorkRequirementStatusCompleted)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Work requirement completed: %s (status: %s)\n", final.Name, final.Status.Or(""))
}

// ExampleWorkerPoolClient_AddConfiguredWorkerPool demonstrates creating a configured worker pool.
// A configured worker pool registers workers that connect back to the platform,
// rather than being provisioned directly from a compute source. The token secret
// returned here is the only time it is available in full.
func ExampleWorkerPoolClient_AddConfiguredWorkerPool() {
	config := yellowdog.DefaultConfig()
	config.APIKey = "your-api-key"
	config.APISecret = "your-api-secret"

	client, err := yellowdog.NewClient(config)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = client.Close() }()

	ctx := context.Background()

	request := &scheduler.AddConfiguredWorkerPoolRequest{
		Name:      "my-configured-pool",
		Namespace: optional.Of("my-namespace"),
		TokenTtl:  optional.Of(model.MustParseDuration("4h")),
		Properties: optional.Of(model.ConfiguredWorkerPoolProperties{
			WorkerTag: optional.Of("my-worker-tag"),
		}),
	}

	resp, err := client.WorkerPool.AddConfiguredWorkerPool(ctx, request)
	if err != nil {
		log.Fatal(err)
	}

	if pool, ok := resp.WorkerPool.Get(); ok {
		fmt.Printf("Created configured worker pool: %s (ID: %s)\n", pool.GetName(), pool.GetId())
	}

	// The token secret in resp.Token can be passed to workers so they can register with the pool.
	if token, ok := resp.Token.Get(); ok {
		if secret, ok := token.Secret.Get(); ok {
			fmt.Printf("Worker token secret: %s\n", secret)
		}
	}
}

// ExampleWorkerPoolHelper demonstrates waiting for workers to connect to a pool.
// GetWorkerPool returns a model.WorkerPool interface; use a type assertion
// to access fields specific to model.ConfiguredWorkerPool.
func ExampleWorkerPoolHelper() {
	config := yellowdog.DefaultConfig()
	config.APIKey = "your-api-key"
	config.APISecret = "your-api-secret"

	client, err := yellowdog.NewClient(config)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = client.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	// GetWorkerPool returns a model.WorkerPool interface.
	// Type-assert to *model.ConfiguredWorkerPool to use WorkerPoolHelper.
	wp, err := client.WorkerPool.GetWorkerPool(ctx, "ydid:wrkrpool:000000:abc123")
	if err != nil {
		log.Fatal(err)
	}

	pool, ok := wp.(*model.ConfiguredWorkerPool)
	if !ok {
		log.Fatalf("expected ConfiguredWorkerPool, got %T", wp)
	}

	// Wait for at least one worker to connect
	helper := scheduler.NewWorkerPoolHelper(client.WorkerPool, pool)
	final, err := helper.WaitUntil(ctx, func(p *model.ConfiguredWorkerPool) bool {
		workers, ok := p.WorkerSummary.Get()
		return ok && workers.ClaimedWorkerCount >= 1
	})
	if err != nil {
		log.Fatal(err)
	}

	workers, _ := final.WorkerSummary.Get()
	fmt.Printf("Worker pool %s has %d worker(s)\n", final.GetName(), workers.ClaimedWorkerCount)
}
