package scheduler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/yellowdogai/yellowdog-sdk-go/compute"
	"github.com/yellowdogai/yellowdog-sdk-go/internal"
	"github.com/yellowdogai/yellowdog-sdk-go/model"
	"github.com/yellowdogai/yellowdog-sdk-go/optional"
)

// workerPoolClientProxy makes the requests for the generated WorkerPoolClient, which delegates each
// method to the method here of the same name with its first letter lowercased. The exceptions are
// addNodeActionsById and newWorkerPoolUpdatesSSEClient, which back hand-written methods in
// pool_helper.go.
type workerPoolClientProxy struct {
	http *internal.HTTPClient
}

func newWorkerPoolClientProxy(httpClient *internal.HTTPClient) *workerPoolClientProxy {
	return &workerPoolClientProxy{http: httpClient}
}

const workerPoolServicePath = "/workerPools"

// --- Configured worker pools ----------------------------------------------------------------------

func (p *workerPoolClientProxy) addConfiguredWorkerPool(ctx context.Context, request *AddConfiguredWorkerPoolRequest) (*AddConfiguredWorkerPoolResponse, error) {
	var result AddConfiguredWorkerPoolResponse
	if err := p.http.Post(ctx, workerPoolServicePath+"/configured", request, &result); err != nil {
		return nil, fmt.Errorf("add configured worker pool: %w", err)
	}
	return &result, nil
}

// tokenRequest issues one of the three configured-worker-pool token routes. The TTL is the body,
// and is sent as null when absent.
func (p *workerPoolClientProxy) tokenRequest(ctx context.Context, workerPoolId string, suffix string, tokenTtl *model.Duration, action string) (*model.WorkerPoolToken, error) {
	var result model.WorkerPoolToken
	path := fmt.Sprintf("%s/configured/%s/token%s", workerPoolServicePath, workerPoolId, suffix)

	var err error
	if suffix == "" {
		err = p.http.Get(ctx, path, &result)
	} else {
		err = p.http.Post(ctx, path, tokenTtl, &result)
	}
	if err != nil {
		return nil, fmt.Errorf("%s configured worker pool token: %w", action, err)
	}
	return &result, nil
}

func (p *workerPoolClientProxy) refreshConfiguredWorkerPoolToken(ctx context.Context, workerPool *model.ConfiguredWorkerPool) (*model.WorkerPoolToken, error) {
	return p.refreshConfiguredWorkerPoolTokenByIdWithTokenTtl(ctx, workerPool.GetId(), nil)
}

func (p *workerPoolClientProxy) refreshConfiguredWorkerPoolTokenWithTokenTtl(ctx context.Context, workerPool *model.ConfiguredWorkerPool, tokenTtl *model.Duration) (*model.WorkerPoolToken, error) {
	return p.refreshConfiguredWorkerPoolTokenByIdWithTokenTtl(ctx, workerPool.GetId(), tokenTtl)
}

func (p *workerPoolClientProxy) refreshConfiguredWorkerPoolTokenById(ctx context.Context, workerPoolId string) (*model.WorkerPoolToken, error) {
	return p.refreshConfiguredWorkerPoolTokenByIdWithTokenTtl(ctx, workerPoolId, nil)
}

func (p *workerPoolClientProxy) refreshConfiguredWorkerPoolTokenByIdWithTokenTtl(ctx context.Context, workerPoolId string, tokenTtl *model.Duration) (*model.WorkerPoolToken, error) {
	return p.tokenRequest(ctx, workerPoolId, "/refresh", tokenTtl, "refresh")
}

func (p *workerPoolClientProxy) regenerateConfiguredWorkerPoolToken(ctx context.Context, workerPool *model.ConfiguredWorkerPool) (*model.WorkerPoolToken, error) {
	return p.regenerateConfiguredWorkerPoolTokenByIdWithTokenTtl(ctx, workerPool.GetId(), nil)
}

func (p *workerPoolClientProxy) regenerateConfiguredWorkerPoolTokenWithTokenTtl(ctx context.Context, workerPool *model.ConfiguredWorkerPool, tokenTtl *model.Duration) (*model.WorkerPoolToken, error) {
	return p.regenerateConfiguredWorkerPoolTokenByIdWithTokenTtl(ctx, workerPool.GetId(), tokenTtl)
}

func (p *workerPoolClientProxy) regenerateConfiguredWorkerPoolTokenById(ctx context.Context, workerPoolId string) (*model.WorkerPoolToken, error) {
	return p.regenerateConfiguredWorkerPoolTokenByIdWithTokenTtl(ctx, workerPoolId, nil)
}

func (p *workerPoolClientProxy) regenerateConfiguredWorkerPoolTokenByIdWithTokenTtl(ctx context.Context, workerPoolId string, tokenTtl *model.Duration) (*model.WorkerPoolToken, error) {
	return p.tokenRequest(ctx, workerPoolId, "/regenerate", tokenTtl, "regenerate")
}

func (p *workerPoolClientProxy) getConfiguredWorkerPoolToken(ctx context.Context, workerPool *model.ConfiguredWorkerPool) (*model.WorkerPoolToken, error) {
	return p.getConfiguredWorkerPoolTokenById(ctx, workerPool.GetId())
}

func (p *workerPoolClientProxy) getConfiguredWorkerPoolTokenById(ctx context.Context, workerPoolId string) (*model.WorkerPoolToken, error) {
	return p.tokenRequest(ctx, workerPoolId, "", nil, "get")
}

// --- Provisioned worker pools ---------------------------------------------------------------------

func (p *workerPoolClientProxy) provisionWorkerPool(ctx context.Context, computeRequirementTemplateUsage *compute.ComputeRequirementTemplateUsage) (*model.ProvisionedWorkerPool, error) {
	return p.provisionWorkerPoolWithProvisionedProperties(ctx, computeRequirementTemplateUsage, nil)
}

func (p *workerPoolClientProxy) provisionWorkerPoolWithProvisionedProperties(ctx context.Context, computeRequirementTemplateUsage *compute.ComputeRequirementTemplateUsage, provisionedProperties *model.ProvisionedWorkerPoolProperties) (*model.ProvisionedWorkerPool, error) {
	request := &ProvisionTemplateWorkerPoolRequest{RequirementTemplateUsage: *computeRequirementTemplateUsage}
	if provisionedProperties != nil {
		request.ProvisionedProperties = optional.Of(*provisionedProperties)
	}

	var result model.ProvisionedWorkerPool
	if err := p.http.Post(ctx, workerPoolServicePath+"/provisioned/template", request, &result); err != nil {
		return nil, fmt.Errorf("provision worker pool: %w", err)
	}
	return &result, nil
}

func (p *workerPoolClientProxy) resizeWorkerPool(ctx context.Context, workerPool *model.ProvisionedWorkerPool, size int) (*model.ProvisionedWorkerPool, error) {
	return p.resizeWorkerPoolById(ctx, workerPool.GetId(), size)
}

func (p *workerPoolClientProxy) resizeWorkerPoolById(ctx context.Context, workerPoolId string, size int) (*model.ProvisionedWorkerPool, error) {
	var result model.ProvisionedWorkerPool
	path := fmt.Sprintf("%s/provisioned/%s?size=%d", workerPoolServicePath, workerPoolId, size)
	if err := p.http.Put(ctx, path, nil, &result); err != nil {
		return nil, fmt.Errorf("resize worker pool: %w", err)
	}
	return &result, nil
}

// --- Worker pools ---------------------------------------------------------------------------------

func (p *workerPoolClientProxy) shutdownWorkerPool(ctx context.Context, workerPool model.WorkerPool) error {
	return p.shutdownWorkerPoolById(ctx, workerPool.GetId())
}

func (p *workerPoolClientProxy) shutdownWorkerPoolById(ctx context.Context, workerPoolId string) error {
	path := fmt.Sprintf("%s/%s", workerPoolServicePath, workerPoolId)
	if err := p.http.Delete(ctx, path); err != nil {
		return fmt.Errorf("shutdown worker pool: %w", err)
	}
	return nil
}

func (p *workerPoolClientProxy) getWorkerPool(ctx context.Context, workerPoolId string) (model.WorkerPool, error) {
	var raw json.RawMessage
	path := fmt.Sprintf("%s/%s", workerPoolServicePath, workerPoolId)
	if err := p.http.Get(ctx, path, &raw); err != nil {
		return nil, fmt.Errorf("get worker pool: %w", err)
	}
	return model.UnmarshalWorkerPool(raw)
}

func (p *workerPoolClientProxy) getWorkerPoolByName(ctx context.Context, namespace string, name string) (model.WorkerPool, error) {
	var raw json.RawMessage
	path := fmt.Sprintf("%s/%s/%s", workerPoolServicePath, namespace, name)
	if err := p.http.Get(ctx, path, &raw); err != nil {
		return nil, fmt.Errorf("get worker pool: %w", err)
	}
	return model.UnmarshalWorkerPool(raw)
}

func (p *workerPoolClientProxy) getWorkerPools(search *WorkerPoolSearch) *internal.SearchClient[WorkerPoolSummary] {
	return internal.NewSearchClient[WorkerPoolSummary](p.http, workerPoolServicePath, search, internal.Param{Name: "sliced", Value: "true"})
}

// --- Nodes -----------------------------------------------------------------------------------------

func (p *workerPoolClientProxy) getNode(ctx context.Context, node *model.Node) (*model.Node, error) {
	return p.getNodeById(ctx, node.Id.Or(""))
}

func (p *workerPoolClientProxy) getNodeById(ctx context.Context, nodeId string) (*model.Node, error) {
	var result model.Node
	path := fmt.Sprintf("%s/nodes/%s", workerPoolServicePath, nodeId)
	if err := p.http.Get(ctx, path, &result); err != nil {
		return nil, fmt.Errorf("get node: %w", err)
	}
	return &result, nil
}

func (p *workerPoolClientProxy) getNodeByWorkerId(ctx context.Context, workerId string) (*model.Node, error) {
	var result model.Node
	path := fmt.Sprintf("%s/workers/%s/node", workerPoolServicePath, workerId)
	if err := p.http.Get(ctx, path, &result); err != nil {
		return nil, fmt.Errorf("get node by worker id: %w", err)
	}
	return &result, nil
}

func (p *workerPoolClientProxy) shutdownNode(ctx context.Context, node *model.Node) (*model.Node, error) {
	return p.shutdownNodeById(ctx, node.Id.Or(""))
}

func (p *workerPoolClientProxy) shutdownNodeById(ctx context.Context, nodeId string) (*model.Node, error) {
	var result model.Node
	path := fmt.Sprintf("%s/nodes/%s", workerPoolServicePath, nodeId)
	if err := p.http.DeleteWithResult(ctx, path, &result); err != nil {
		return nil, fmt.Errorf("shutdown node: %w", err)
	}
	return &result, nil
}

func (p *workerPoolClientProxy) getNodes(search *NodeSearch) *internal.SearchClient[model.Node] {
	return internal.NewSearchClient[model.Node](p.http, workerPoolServicePath+"/nodes", search)
}

func (p *workerPoolClientProxy) getNodeActions(ctx context.Context, node *model.Node) (*NodeActionQueueSnapshot, error) {
	return p.getNodeActionsById(ctx, node.Id.Or(""))
}

func (p *workerPoolClientProxy) getNodeActionsById(ctx context.Context, nodeId string) (*NodeActionQueueSnapshot, error) {
	var result NodeActionQueueSnapshot
	path := fmt.Sprintf("%s/nodes/%s/actions", workerPoolServicePath, nodeId)
	if err := p.http.Get(ctx, path, &result); err != nil {
		return nil, fmt.Errorf("get node actions: %w", err)
	}
	return &result, nil
}

func (p *workerPoolClientProxy) addNodeActionsById(ctx context.Context, workerPoolId string, request *AddNodeActionsRequest) error {
	path := fmt.Sprintf("%s/%s/actions", workerPoolServicePath, workerPoolId)
	if err := p.http.Post(ctx, path, request, nil); err != nil {
		return fmt.Errorf("add node actions: %w", err)
	}
	return nil
}

// --- Server-sent events -----------------------------------------------------------------------------

func (p *workerPoolClientProxy) newWorkerPoolUpdatesSSEClient(workerPoolId string) internal.SSEClient {
	return p.http.NewSSEClient(fmt.Sprintf("%s/%s/updates", workerPoolServicePath, workerPoolId))
}
