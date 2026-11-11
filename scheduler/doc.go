// Package scheduler provides access to the YellowDog Work and WorkerPool APIs.
//
// It covers work requirements, task groups, tasks, worker pool management,
// node operations, and real-time status monitoring via Server-Sent Events (SSE).
//
// # Work scheduling
//
// A [model.WorkRequirement] contains one or more [model.TaskGroup]s. Each task
// group holds [model.Task]s and a [model.RunSpecification] that describes what
// kind of workers should run them. Use [WorkClient] to create and manage work
// requirements:
//
//	req, _ := client.Work.AddWorkRequirement(ctx, workReq)
//	client.Work.AddTasksToTaskGroup(ctx, &req.TaskGroups[0], tasks)
//
// Use [WorkRequirementHelper] (obtained via [WorkClient.Helper]) to wait for
// a requirement to reach a terminal status:
//
//	helper := client.Work.Helper(req)
//	final, _ := helper.WaitForStatus(ctx, model.WorkRequirementStatusCompleted)
//
// # Worker pools
//
// [WorkerPoolClient] manages worker pools. There are two pool kinds:
//
//   - Provisioned pools — backed by a [model.ComputeRequirement]; workers are
//     launched automatically by the platform.
//   - Configured pools — workers connect back using a token; useful for
//     on-premise or custom environments.
//
// # GetWorkerPool return type
//
// [WorkerPoolClient.GetWorkerPool] returns a [model.WorkerPool] interface.
// Type-assert to [model.ConfiguredWorkerPool] (or the provisioned equivalent)
// to access pool-specific fields or to use [WorkerPoolHelper]:
//
//	wp, _ := client.WorkerPool.GetWorkerPool(ctx, poolId)
//	pool, ok := wp.(*model.ConfiguredWorkerPool)
//	if !ok {
//	    log.Fatalf("expected ConfiguredWorkerPool, got %T", wp)
//	}
//	helper := NewWorkerPoolHelper(client.WorkerPool, pool)
//
// # Node actions
//
// Node actions allow the platform to direct workers to create or destroy worker
// processes. Use [WorkerPoolClient.AddNodeActions] (object overload) or
// [WorkerPoolClient.AddNodeActionsById] (ID overload).
package scheduler
