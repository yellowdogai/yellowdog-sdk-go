package usage

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/yellowdogai/yellowdog-sdk-go/internal"
	"github.com/yellowdogai/yellowdog-sdk-go/model"
)

// allowancesClientProxy makes every request the Allowances client sends. Each method backs the
// AllowancesClient method of the same name with its first letter lowercased.
type allowancesClientProxy struct {
	http *internal.HTTPClient
}

func newAllowancesClientProxy(httpClient *internal.HTTPClient) *allowancesClientProxy {
	return &allowancesClientProxy{http: httpClient}
}

const allowancesServicePath = "/allowances"

func (p *allowancesClientProxy) addAllowance(ctx context.Context, allowance model.Allowance) (model.Allowance, error) {
	var raw json.RawMessage
	if err := p.http.Post(ctx, allowancesServicePath, allowance, &raw); err != nil {
		return nil, fmt.Errorf("add allowance: %w", err)
	}
	return model.UnmarshalAllowance(raw)
}

func (p *allowancesClientProxy) deleteAllowance(ctx context.Context, allowance model.Allowance) error {
	return p.deleteAllowanceById(ctx, allowance.GetId())
}

func (p *allowancesClientProxy) deleteAllowanceById(ctx context.Context, allowanceId string) error {
	path := fmt.Sprintf("%s/%s", allowancesServicePath, allowanceId)
	if err := p.http.Delete(ctx, path); err != nil {
		return fmt.Errorf("delete allowance: %w", err)
	}
	return nil
}

func (p *allowancesClientProxy) getAllowance(ctx context.Context, allowance model.Allowance) (model.Allowance, error) {
	return p.getAllowanceById(ctx, allowance.GetId())
}

func (p *allowancesClientProxy) getAllowanceById(ctx context.Context, allowanceId string) (model.Allowance, error) {
	var raw json.RawMessage
	path := fmt.Sprintf("%s/%s", allowancesServicePath, allowanceId)
	if err := p.http.Get(ctx, path, &raw); err != nil {
		return nil, fmt.Errorf("get allowance: %w", err)
	}
	return model.UnmarshalAllowance(raw)
}

func (p *allowancesClientProxy) boostAllowance(ctx context.Context, allowance model.Allowance, boostHours int) (model.Allowance, error) {
	return p.boostAllowanceById(ctx, allowance.GetId(), boostHours)
}

func (p *allowancesClientProxy) boostAllowanceById(ctx context.Context, allowanceId string, boostHours int) (model.Allowance, error) {
	var raw json.RawMessage
	path := fmt.Sprintf("%s/%s/boost/%d", allowancesServicePath, allowanceId, boostHours)
	if err := p.http.Post(ctx, path, nil, &raw); err != nil {
		return nil, fmt.Errorf("boost allowance: %w", err)
	}
	return model.UnmarshalAllowance(raw)
}

func (p *allowancesClientProxy) getAllowances(allowanceSearch *AllowanceSearch) *internal.SearchClient[model.Allowance] {
	return internal.NewPolymorphicSearchClient[model.Allowance](p.http, allowancesServicePath, allowanceSearch, model.UnmarshalAllowance)
}

func (p *allowancesClientProxy) checkComputeRequirementExhaustion(ctx context.Context, computeRequirement *model.ComputeRequirement) ([]*AllowanceExhaustedNotification, error) {
	var result []*AllowanceExhaustedNotification
	if err := p.http.Put(ctx, allowancesServicePath+"/requirement", computeRequirement, &result); err != nil {
		return nil, fmt.Errorf("check compute requirement exhaustion: %w", err)
	}
	return result, nil
}
