package scheduler

import (
	"context"
	"fmt"

	"github.com/yellowdogai/yellowdog-sdk-go/internal"
	"github.com/yellowdogai/yellowdog-sdk-go/model"
)

// workClientProxy makes the requests for the generated WorkClient, which delegates each method to
// the method here of the same name with its first letter lowercased.
type workClientProxy struct {
	http *internal.HTTPClient
}

func newWorkClientProxy(httpClient *internal.HTTPClient) *workClientProxy {
	return &workClientProxy{http: httpClient}
}

const workServicePath = "/work"

// --- Work requirements --------------------------------------------------------------------------

func (p *workClientProxy) addWorkRequirement(ctx context.Context, workRequirement *model.WorkRequirement) (*model.WorkRequirement, error) {
	var result model.WorkRequirement
	if err := p.http.Post(ctx, workServicePath+"/requirements", workRequirement, &result); err != nil {
		return nil, fmt.Errorf("add work requirement: %w", err)
	}
	return &result, nil
}

func (p *workClientProxy) updateWorkRequirement(ctx context.Context, workRequirement *model.WorkRequirement) (*model.WorkRequirement, error) {
	var result model.WorkRequirement
	if err := p.http.Put(ctx, workServicePath+"/requirements", workRequirement, &result); err != nil {
		return nil, fmt.Errorf("update work requirement: %w", err)
	}
	return &result, nil
}

func (p *workClientProxy) getWorkRequirement(ctx context.Context, workRequirement *model.WorkRequirement) (*model.WorkRequirement, error) {
	return p.getWorkRequirementById(ctx, workRequirement.Id.Or(""))
}

func (p *workClientProxy) getWorkRequirementById(ctx context.Context, workRequirementId string) (*model.WorkRequirement, error) {
	var result model.WorkRequirement
	path := fmt.Sprintf("%s/requirements/%s", workServicePath, workRequirementId)
	if err := p.http.Get(ctx, path, &result); err != nil {
		return nil, fmt.Errorf("get work requirement: %w", err)
	}
	return &result, nil
}

func (p *workClientProxy) getWorkRequirementByName(ctx context.Context, namespace string, workRequirementName string) (*model.WorkRequirement, error) {
	var result model.WorkRequirement
	path := fmt.Sprintf("%s/namespaces/%s/requirements/%s", workServicePath, namespace, workRequirementName)
	if err := p.http.Get(ctx, path, &result); err != nil {
		return nil, fmt.Errorf("get work requirement: %w", err)
	}
	return &result, nil
}

func (p *workClientProxy) getWorkRequirements(search *WorkRequirementSearch) *internal.SearchClient[WorkRequirementSummary] {
	return internal.NewSearchClient[WorkRequirementSummary](p.http, workServicePath+"/requirements", search, internal.Param{Name: "sliced", Value: "true"})
}

// --- Work requirement transitions ---------------------------------------------------------------

// transitionWorkRequirement is the single route behind hold, start, finish and cancel. The abort
// parameter is always sent: the platform reads its absence as false, so sending it explicitly makes
// the request say what it means.
func (p *workClientProxy) transitionWorkRequirement(ctx context.Context, workRequirementId string, nextStatus model.WorkRequirementStatus, abort bool, action string) (*model.WorkRequirement, error) {
	var result model.WorkRequirement
	path := fmt.Sprintf("%s/requirements/%s/transition/%s?abort=%t", workServicePath, workRequirementId, nextStatus, abort)
	if err := p.http.Put(ctx, path, nil, &result); err != nil {
		return nil, fmt.Errorf("%s work requirement: %w", action, err)
	}
	return &result, nil
}

func (p *workClientProxy) holdWorkRequirement(ctx context.Context, workRequirement *model.WorkRequirement) (*model.WorkRequirement, error) {
	return p.holdWorkRequirementById(ctx, workRequirement.Id.Or(""))
}

func (p *workClientProxy) holdWorkRequirementById(ctx context.Context, workRequirementId string) (*model.WorkRequirement, error) {
	return p.transitionWorkRequirement(ctx, workRequirementId, model.WorkRequirementStatusHeld, false, "hold")
}

func (p *workClientProxy) startWorkRequirement(ctx context.Context, workRequirement *model.WorkRequirement) (*model.WorkRequirement, error) {
	return p.startWorkRequirementById(ctx, workRequirement.Id.Or(""))
}

func (p *workClientProxy) startWorkRequirementById(ctx context.Context, workRequirementId string) (*model.WorkRequirement, error) {
	return p.transitionWorkRequirement(ctx, workRequirementId, model.WorkRequirementStatusRunning, false, "start")
}

func (p *workClientProxy) finishWorkRequirement(ctx context.Context, workRequirement *model.WorkRequirement) (*model.WorkRequirement, error) {
	return p.finishWorkRequirementById(ctx, workRequirement.Id.Or(""))
}

func (p *workClientProxy) finishWorkRequirementById(ctx context.Context, workRequirementId string) (*model.WorkRequirement, error) {
	return p.transitionWorkRequirement(ctx, workRequirementId, model.WorkRequirementStatusFinishing, false, "finish")
}

func (p *workClientProxy) cancelWorkRequirement(ctx context.Context, workRequirement *model.WorkRequirement) (*model.WorkRequirement, error) {
	return p.cancelWorkRequirementByIdWithAbort(ctx, workRequirement.Id.Or(""), false)
}

func (p *workClientProxy) cancelWorkRequirementWithAbort(ctx context.Context, workRequirement *model.WorkRequirement, abort bool) (*model.WorkRequirement, error) {
	return p.cancelWorkRequirementByIdWithAbort(ctx, workRequirement.Id.Or(""), abort)
}

func (p *workClientProxy) cancelWorkRequirementById(ctx context.Context, workRequirementId string) (*model.WorkRequirement, error) {
	return p.cancelWorkRequirementByIdWithAbort(ctx, workRequirementId, false)
}

func (p *workClientProxy) cancelWorkRequirementByIdWithAbort(ctx context.Context, workRequirementId string, abort bool) (*model.WorkRequirement, error) {
	return p.transitionWorkRequirement(ctx, workRequirementId, model.WorkRequirementStatusCancelling, abort, "cancel")
}

// --- Task groups and tasks ------------------------------------------------------------------------

func (p *workClientProxy) addTasksToTaskGroup(ctx context.Context, taskGroup *model.TaskGroup, tasks []*model.Task) ([]*model.Task, error) {
	return p.addTasksToTaskGroupById(ctx, taskGroup.Id.Or(""), tasks)
}

func (p *workClientProxy) addTasksToTaskGroupById(ctx context.Context, taskGroupId string, tasks []*model.Task) ([]*model.Task, error) {
	var result []*model.Task
	path := fmt.Sprintf("%s/taskGroups/%s/tasks", workServicePath, taskGroupId)
	if err := p.http.Post(ctx, path, tasks, &result); err != nil {
		return nil, fmt.Errorf("add tasks to task group: %w", err)
	}
	return result, nil
}

func (p *workClientProxy) addTasksToTaskGroupByName(ctx context.Context, namespace string, workRequirementName string, taskGroupName string, tasks []*model.Task) ([]*model.Task, error) {
	var result []*model.Task
	path := fmt.Sprintf("%s/namespaces/%s/requirements/%s/taskGroups/%s/tasks",
		workServicePath, namespace, workRequirementName, taskGroupName)
	if err := p.http.Post(ctx, path, tasks, &result); err != nil {
		return nil, fmt.Errorf("add tasks to task group: %w", err)
	}
	return result, nil
}

func (p *workClientProxy) getTask(ctx context.Context, task *model.Task) (*model.Task, error) {
	return p.getTaskById(ctx, task.Id.Or(""))
}

func (p *workClientProxy) getTaskById(ctx context.Context, taskId string) (*model.Task, error) {
	var result model.Task
	path := fmt.Sprintf("%s/tasks/%s", workServicePath, taskId)
	if err := p.http.Get(ctx, path, &result); err != nil {
		return nil, fmt.Errorf("get task: %w", err)
	}
	return &result, nil
}

func (p *workClientProxy) cancelTask(ctx context.Context, task *model.Task, abort bool) (*model.Task, error) {
	return p.cancelTaskById(ctx, task.Id.Or(""), abort)
}

func (p *workClientProxy) cancelTaskById(ctx context.Context, taskId string, abort bool) (*model.Task, error) {
	var result model.Task
	path := fmt.Sprintf("%s/tasks/%s/cancel?abort=%t", workServicePath, taskId, abort)
	if err := p.http.Put(ctx, path, nil, &result); err != nil {
		return nil, fmt.Errorf("cancel task: %w", err)
	}
	return &result, nil
}

func (p *workClientProxy) getTasks(search *TaskSearch) *internal.SearchClient[model.Task] {
	return internal.NewSearchClient[model.Task](p.http, workServicePath+"/tasks", search)
}

// --- Server-sent events ---------------------------------------------------------------------------

// newWorkRequirementUpdatesSSEClient backs the hand-written subscription API in work_helpers.go and
// work_helper.go rather than a generated method.
func (p *workClientProxy) newWorkRequirementUpdatesSSEClient(workRequirementId string) internal.SSEClient {
	return p.http.NewSSEClient(fmt.Sprintf("%s/requirements/%s/updates", workServicePath, workRequirementId))
}
