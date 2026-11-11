package model

import "github.com/yellowdogai/yellowdog-sdk-go/optional"

// WorkerPoolWithState is implemented by both ConfiguredWorkerPool and ProvisionedWorkerPool,
// providing access to runtime status and worker summary.
type WorkerPoolWithState interface {
	WorkerPool
	GetStatus() optional.Optional[WorkerPoolStatus]
	GetWorkerSummary() optional.Optional[WorkerSummary]
}

func (c *ConfiguredWorkerPool) GetStatus() optional.Optional[WorkerPoolStatus] { return c.Status }
func (c *ConfiguredWorkerPool) GetWorkerSummary() optional.Optional[WorkerSummary] {
	return c.WorkerSummary
}

func (p *ProvisionedWorkerPool) GetStatus() optional.Optional[WorkerPoolStatus] { return p.Status }
func (p *ProvisionedWorkerPool) GetWorkerSummary() optional.Optional[WorkerSummary] {
	return p.WorkerSummary
}
