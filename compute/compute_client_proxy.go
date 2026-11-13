package compute

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/yellowdogai/yellowdog-sdk-go/internal"
	"github.com/yellowdogai/yellowdog-sdk-go/model"
)

// computeClientProxy makes every request the Compute client sends. Each method backs the
// ComputeClient method of the same name with its first letter lowercased.
type computeClientProxy struct {
	http *internal.HTTPClient
}

func newComputeClientProxy(httpClient *internal.HTTPClient) *computeClientProxy {
	return &computeClientProxy{http: httpClient}
}

const computeServicePath = "/compute"

// --- Compute requirements -------------------------------------------------------------------

func (p *computeClientProxy) addComputeRequirement(ctx context.Context, computeRequirement *model.ComputeRequirement) (*model.ComputeRequirement, error) {
	var result model.ComputeRequirement
	if err := p.http.Post(ctx, computeServicePath+"/requirements", computeRequirement, &result); err != nil {
		return nil, fmt.Errorf("add compute requirement: %w", err)
	}
	return &result, nil
}

func (p *computeClientProxy) updateComputeRequirement(ctx context.Context, computeRequirement *model.ComputeRequirement) (*model.ComputeRequirement, error) {
	return p.updateComputeRequirementWithReprovision(ctx, computeRequirement, false)
}

func (p *computeClientProxy) updateComputeRequirementWithReprovision(ctx context.Context, computeRequirement *model.ComputeRequirement, reprovision bool) (*model.ComputeRequirement, error) {
	var result model.ComputeRequirement
	path := fmt.Sprintf("%s/requirements?reprovision=%v", computeServicePath, reprovision)
	if err := p.http.Put(ctx, path, computeRequirement, &result); err != nil {
		return nil, fmt.Errorf("update compute requirement: %w", err)
	}
	return &result, nil
}

func (p *computeClientProxy) updateComputeRequirementAutoShutdown(ctx context.Context, id string) (*model.ComputeRequirement, error) {
	return p.updateComputeRequirementAutoShutdownWithAutoShutdown(ctx, id, nil)
}

func (p *computeClientProxy) updateComputeRequirementAutoShutdownWithAutoShutdown(ctx context.Context, id string, shutdown *model.AutoShutdown) (*model.ComputeRequirement, error) {
	var result model.ComputeRequirement
	path := fmt.Sprintf("%s/requirements/%s/autoShutdown", computeServicePath, id)
	if err := p.http.Put(ctx, path, shutdown, &result); err != nil {
		return nil, fmt.Errorf("update compute requirement auto shutdown: %w", err)
	}
	return &result, nil
}

func (p *computeClientProxy) getComputeRequirement(ctx context.Context, computeRequirement *model.ComputeRequirement) (*model.ComputeRequirement, error) {
	return p.getComputeRequirementById(ctx, computeRequirement.Id.Or(""))
}

func (p *computeClientProxy) getComputeRequirementById(ctx context.Context, computeRequirementId string) (*model.ComputeRequirement, error) {
	var result model.ComputeRequirement
	path := fmt.Sprintf("%s/requirements/%s", computeServicePath, computeRequirementId)
	if err := p.http.Get(ctx, path, &result); err != nil {
		return nil, fmt.Errorf("get compute requirement: %w", err)
	}
	return &result, nil
}

func (p *computeClientProxy) getComputeRequirementByName(ctx context.Context, namespace string, computeRequirementName string) (*model.ComputeRequirement, error) {
	var result model.ComputeRequirement
	path := fmt.Sprintf("%s/namespaces/%s/requirements/%s", computeServicePath, namespace, computeRequirementName)
	if err := p.http.Get(ctx, path, &result); err != nil {
		return nil, fmt.Errorf("get compute requirement: %w", err)
	}
	return &result, nil
}

func (p *computeClientProxy) isComputeRequirementUpdating(ctx context.Context, computeRequirement *model.ComputeRequirement) (bool, error) {
	return p.isComputeRequirementUpdatingById(ctx, computeRequirement.Id.Or(""))
}

func (p *computeClientProxy) isComputeRequirementUpdatingById(ctx context.Context, computeRequirementId string) (bool, error) {
	var result bool
	path := fmt.Sprintf("%s/requirements/%s/updating", computeServicePath, computeRequirementId)
	if err := p.http.Get(ctx, path, &result); err != nil {
		return result, fmt.Errorf("is compute requirement updating: %w", err)
	}
	return result, nil
}

func (p *computeClientProxy) getComputeRequirements(computeRequirementSearch *ComputeRequirementSearch) *internal.SearchClient[model.ComputeRequirement] {
	return internal.NewSearchClient[model.ComputeRequirement](p.http, computeServicePath+"/requirements", computeRequirementSearch)
}

// getComputeRequirementSummaries shares its path with getComputeRequirements; summaries=true is
// the only thing that distinguishes the two requests.
func (p *computeClientProxy) getComputeRequirementSummaries(computeRequirementSummarySearch *ComputeRequirementSummarySearch) *internal.SearchClient[ComputeRequirementSummary] {
	return internal.NewSearchClient[ComputeRequirementSummary](p.http, computeServicePath+"/requirements", computeRequirementSummarySearch, internal.Param{Name: "summaries", Value: "true"})
}

// --- Compute requirement transitions --------------------------------------------------------

// transitionComputeRequirement is the single route behind stop, start, terminate and reprovision.
func (p *computeClientProxy) transitionComputeRequirement(ctx context.Context, computeRequirementId string, nextStatus model.ComputeRequirementStatus, action string) (*model.ComputeRequirement, error) {
	var result model.ComputeRequirement
	path := fmt.Sprintf("%s/requirements/%s/transition/%s", computeServicePath, computeRequirementId, nextStatus)
	if err := p.http.Put(ctx, path, nil, &result); err != nil {
		return nil, fmt.Errorf("%s compute requirement: %w", action, err)
	}
	return &result, nil
}

func (p *computeClientProxy) stopComputeRequirement(ctx context.Context, computeRequirement *model.ComputeRequirement) (*model.ComputeRequirement, error) {
	return p.stopComputeRequirementById(ctx, computeRequirement.Id.Or(""))
}

func (p *computeClientProxy) stopComputeRequirementById(ctx context.Context, computeRequirementId string) (*model.ComputeRequirement, error) {
	return p.transitionComputeRequirement(ctx, computeRequirementId, model.ComputeRequirementStatusStopped, "stop")
}

func (p *computeClientProxy) startComputeRequirement(ctx context.Context, computeRequirement *model.ComputeRequirement) (*model.ComputeRequirement, error) {
	return p.startComputeRequirementById(ctx, computeRequirement.Id.Or(""))
}

func (p *computeClientProxy) startComputeRequirementById(ctx context.Context, computeRequirementId string) (*model.ComputeRequirement, error) {
	return p.transitionComputeRequirement(ctx, computeRequirementId, model.ComputeRequirementStatusRunning, "start")
}

func (p *computeClientProxy) terminateComputeRequirement(ctx context.Context, computeRequirement *model.ComputeRequirement) (*model.ComputeRequirement, error) {
	return p.terminateComputeRequirementById(ctx, computeRequirement.Id.Or(""))
}

func (p *computeClientProxy) terminateComputeRequirementById(ctx context.Context, computeRequirementId string) (*model.ComputeRequirement, error) {
	return p.transitionComputeRequirement(ctx, computeRequirementId, model.ComputeRequirementStatusTerminated, "terminate")
}

func (p *computeClientProxy) reprovisionComputeRequirement(ctx context.Context, computeRequirement *model.ComputeRequirement) (*model.ComputeRequirement, error) {
	return p.reprovisionComputeRequirementById(ctx, computeRequirement.Id.Or(""))
}

func (p *computeClientProxy) reprovisionComputeRequirementById(ctx context.Context, computeRequirementId string) (*model.ComputeRequirement, error) {
	return p.transitionComputeRequirement(ctx, computeRequirementId, model.ComputeRequirementStatusProvisioning, "reprovision")
}

// --- Instances ------------------------------------------------------------------------------

func (p *computeClientProxy) getInstances(instanceSearch *InstanceSearch) *internal.SearchClient[model.Instance] {
	return internal.NewPolymorphicSearchClient[model.Instance](p.http, computeServicePath+"/instances", instanceSearch, model.UnmarshalInstance)
}

// transitionInstances is the single route behind stop, start, restart and terminate. The status
// is typed, because the endpoint parses it as an InstanceStatus and answers 400 for anything that
// is not a member.
func (p *computeClientProxy) transitionInstances(ctx context.Context, computeRequirementId string, nextStatus model.InstanceStatus, instanceIds []*model.InstanceId, action string) error {
	path := fmt.Sprintf("%s/requirements/%s/instances/transition/%s", computeServicePath, computeRequirementId, nextStatus)
	if err := p.http.Put(ctx, path, instanceIds, nil); err != nil {
		return fmt.Errorf("%s instances: %w", action, err)
	}
	return nil
}

func (p *computeClientProxy) stopInstances(ctx context.Context, computeRequirement *model.ComputeRequirement, instances []model.Instance) error {
	return p.stopInstancesById(ctx, computeRequirement, instanceIdsOf(instances))
}

func (p *computeClientProxy) stopInstancesById(ctx context.Context, computeRequirement *model.ComputeRequirement, instanceIds []*model.InstanceId) error {
	return p.transitionInstances(ctx, computeRequirement.Id.Or(""), model.InstanceStatusStopped, instanceIds, "stop")
}

func (p *computeClientProxy) startInstances(ctx context.Context, computeRequirement *model.ComputeRequirement, instances []model.Instance) error {
	return p.startInstancesById(ctx, computeRequirement, instanceIdsOf(instances))
}

func (p *computeClientProxy) startInstancesById(ctx context.Context, computeRequirement *model.ComputeRequirement, instanceIds []*model.InstanceId) error {
	return p.transitionInstances(ctx, computeRequirement.Id.Or(""), model.InstanceStatusRunning, instanceIds, "start")
}

func (p *computeClientProxy) terminateInstances(ctx context.Context, computeRequirement *model.ComputeRequirement, instances []model.Instance) error {
	return p.terminateInstancesById(ctx, computeRequirement, instanceIdsOf(instances))
}

func (p *computeClientProxy) terminateInstancesById(ctx context.Context, computeRequirement *model.ComputeRequirement, instanceIds []*model.InstanceId) error {
	return p.transitionInstances(ctx, computeRequirement.Id.Or(""), model.InstanceStatusTerminated, instanceIds, "terminate")
}

func (p *computeClientProxy) restartInstances(ctx context.Context, computeRequirement *model.ComputeRequirement, instances []model.Instance) error {
	return p.restartInstancesById(ctx, computeRequirement, instanceIdsOf(instances))
}

// restartInstancesById transitions to PENDING, the status the platform uses for an instance being
// provisioned or rebooted.
func (p *computeClientProxy) restartInstancesById(ctx context.Context, computeRequirement *model.ComputeRequirement, instanceIds []*model.InstanceId) error {
	return p.transitionInstances(ctx, computeRequirement.Id.Or(""), model.InstanceStatusPending, instanceIds, "restart")
}

func (p *computeClientProxy) deprovisionInstances(ctx context.Context, computeRequirement *model.ComputeRequirement, instances []model.Instance) error {
	return p.deprovisionInstancesById(ctx, computeRequirement, instanceIdsOf(instances))
}

func (p *computeClientProxy) deprovisionInstancesById(ctx context.Context, computeRequirement *model.ComputeRequirement, instanceIds []*model.InstanceId) error {
	path := fmt.Sprintf("%s/requirements/%s/instances/deprovision", computeServicePath, computeRequirement.Id.Or(""))
	if err := p.http.Put(ctx, path, instanceIds, nil); err != nil {
		return fmt.Errorf("deprovision instances: %w", err)
	}
	return nil
}

func instanceIdsOf(instances []model.Instance) []*model.InstanceId {
	ids := make([]*model.InstanceId, 0, len(instances))
	for _, instance := range instances {
		id := instance.GetId()
		ids = append(ids, &id)
	}
	return ids
}

// --- Compute source templates -----------------------------------------------------------------

func (p *computeClientProxy) addComputeSourceTemplate(ctx context.Context, computeSourceTemplate *model.ComputeSourceTemplate) (*model.ComputeSourceTemplate, error) {
	var result model.ComputeSourceTemplate
	if err := p.http.Post(ctx, computeServicePath+"/templates/sources", computeSourceTemplate, &result); err != nil {
		return nil, fmt.Errorf("add compute source template: %w", err)
	}
	return &result, nil
}

func (p *computeClientProxy) updateComputeSourceTemplate(ctx context.Context, computeSourceTemplate *model.ComputeSourceTemplate) (*model.ComputeSourceTemplate, error) {
	var result model.ComputeSourceTemplate
	if err := p.http.Put(ctx, computeServicePath+"/templates/sources", computeSourceTemplate, &result); err != nil {
		return nil, fmt.Errorf("update compute source template: %w", err)
	}
	return &result, nil
}

func (p *computeClientProxy) deleteComputeSourceTemplate(ctx context.Context, computeSourceTemplate *model.ComputeSourceTemplate) error {
	return p.deleteComputeSourceTemplateById(ctx, computeSourceTemplate.Id.Or(""))
}

func (p *computeClientProxy) deleteComputeSourceTemplateById(ctx context.Context, computeSourceTemplateId string) error {
	path := fmt.Sprintf("%s/templates/sources/%s", computeServicePath, computeSourceTemplateId)
	if err := p.http.Delete(ctx, path); err != nil {
		return fmt.Errorf("delete compute source template: %w", err)
	}
	return nil
}

func (p *computeClientProxy) getComputeSourceTemplate(ctx context.Context, computeSourceTemplateId string) (*model.ComputeSourceTemplate, error) {
	var result model.ComputeSourceTemplate
	path := fmt.Sprintf("%s/templates/sources/%s", computeServicePath, computeSourceTemplateId)
	if err := p.http.Get(ctx, path, &result); err != nil {
		return nil, fmt.Errorf("get compute source template: %w", err)
	}
	return &result, nil
}

func (p *computeClientProxy) getComputeSourceTemplateByName(ctx context.Context, namespace string, computeSourceTemplateName string) (*model.ComputeSourceTemplate, error) {
	var result model.ComputeSourceTemplate
	path := fmt.Sprintf("%s/namespaces/%s/templates/sources/%s", computeServicePath, namespace, computeSourceTemplateName)
	if err := p.http.Get(ctx, path, &result); err != nil {
		return nil, fmt.Errorf("get compute source template by name: %w", err)
	}
	return &result, nil
}

func (p *computeClientProxy) getComputeSourceTemplates(search *ComputeSourceTemplateSearch) *internal.SearchClient[ComputeSourceTemplateSummary] {
	return internal.NewSearchClient[ComputeSourceTemplateSummary](p.http, computeServicePath+"/templates/sources", search, internal.Param{Name: "sliced", Value: "true"})
}

// --- Compute requirement templates --------------------------------------------------------------

func (p *computeClientProxy) addComputeRequirementTemplate(ctx context.Context, computeRequirementTemplate model.ComputeRequirementTemplate) (model.ComputeRequirementTemplate, error) {
	var raw json.RawMessage
	if err := p.http.Post(ctx, computeServicePath+"/templates/requirements", computeRequirementTemplate, &raw); err != nil {
		return nil, fmt.Errorf("add compute requirement template: %w", err)
	}
	return model.UnmarshalComputeRequirementTemplate(raw)
}

func (p *computeClientProxy) updateComputeRequirementTemplate(ctx context.Context, computeRequirementTemplate model.ComputeRequirementTemplate) (model.ComputeRequirementTemplate, error) {
	var raw json.RawMessage
	if err := p.http.Put(ctx, computeServicePath+"/templates/requirements", computeRequirementTemplate, &raw); err != nil {
		return nil, fmt.Errorf("update compute requirement template: %w", err)
	}
	return model.UnmarshalComputeRequirementTemplate(raw)
}

func (p *computeClientProxy) deleteComputeRequirementTemplate(ctx context.Context, computeRequirementTemplate model.ComputeRequirementTemplate) error {
	return p.deleteComputeRequirementTemplateById(ctx, computeRequirementTemplate.GetId())
}

func (p *computeClientProxy) deleteComputeRequirementTemplateById(ctx context.Context, computeRequirementTemplateId string) error {
	path := fmt.Sprintf("%s/templates/requirements/%s", computeServicePath, computeRequirementTemplateId)
	if err := p.http.Delete(ctx, path); err != nil {
		return fmt.Errorf("delete compute requirement template: %w", err)
	}
	return nil
}

func (p *computeClientProxy) getComputeRequirementTemplate(ctx context.Context, computeRequirementTemplateId string) (model.ComputeRequirementTemplate, error) {
	var raw json.RawMessage
	path := fmt.Sprintf("%s/templates/requirements/%s", computeServicePath, computeRequirementTemplateId)
	if err := p.http.Get(ctx, path, &raw); err != nil {
		return nil, fmt.Errorf("get compute requirement template: %w", err)
	}
	return model.UnmarshalComputeRequirementTemplate(raw)
}

func (p *computeClientProxy) getComputeRequirementTemplateByName(ctx context.Context, namespace string, name string) (model.ComputeRequirementTemplate, error) {
	var raw json.RawMessage
	path := fmt.Sprintf("%s/namespaces/%s/templates/requirements/%s", computeServicePath, namespace, name)
	if err := p.http.Get(ctx, path, &raw); err != nil {
		return nil, fmt.Errorf("get compute requirement template: %w", err)
	}
	return model.UnmarshalComputeRequirementTemplate(raw)
}

func (p *computeClientProxy) getComputeRequirementTemplates(search *ComputeRequirementTemplateSearch) *internal.SearchClient[ComputeRequirementTemplateSummary] {
	return internal.NewSearchClient[ComputeRequirementTemplateSummary](p.http, computeServicePath+"/templates/requirements", search, internal.Param{Name: "sliced", Value: "true"})
}

// --- Provisioning ---------------------------------------------------------------------------

func (p *computeClientProxy) provisionComputeRequirementTemplate(ctx context.Context, computeRequirementTemplateUsage *ComputeRequirementTemplateUsage) (*model.ComputeRequirement, error) {
	var result model.ComputeRequirement
	if err := p.http.Post(ctx, computeServicePath+"/templates/provision", computeRequirementTemplateUsage, &result); err != nil {
		return nil, fmt.Errorf("provision compute requirement template: %w", err)
	}
	return &result, nil
}

func (p *computeClientProxy) testComputeRequirementTemplate(ctx context.Context, computeRequirementTemplateUsage *ComputeRequirementTemplateUsage) (ComputeRequirementTemplateTestResult, error) {
	var raw json.RawMessage
	if err := p.http.Put(ctx, computeServicePath+"/templates/test", computeRequirementTemplateUsage, &raw); err != nil {
		return nil, fmt.Errorf("test compute requirement template: %w", err)
	}
	return UnmarshalComputeRequirementTemplateTestResult(raw)
}

func (p *computeClientProxy) getBestComputeSourceReportByComputeRequirement(ctx context.Context, provisionedComputeRequirementId string) (*model.BestComputeSourceReport, error) {
	var result model.BestComputeSourceReport
	path := fmt.Sprintf("%s/templates/provision/report/requirements/%s", computeServicePath, provisionedComputeRequirementId)
	if err := p.http.Get(ctx, path, &result); err != nil {
		return nil, fmt.Errorf("get best compute source report by compute requirement: %w", err)
	}
	return &result, nil
}

// --- Server-sent events -----------------------------------------------------------------------

// newRequirementUpdatesSSEClient opens the update stream for a requirement. It has no
// ComputeClient counterpart; AddComputeRequirementListener and RequirementHelper consume it.
func (p *computeClientProxy) newRequirementUpdatesSSEClient(computeRequirementId string) internal.SSEClient {
	return p.http.NewSSEClient(fmt.Sprintf("%s/requirements/%s/updates", computeServicePath, computeRequirementId))
}
