package account

import (
	"context"
	"fmt"

	"github.com/yellowdogai/yellowdog-sdk-go/internal"
)

// applicationClientProxy makes the requests for the generated ApplicationClient, which delegates
// each method to the method here of the same name with its first letter lowercased.
type applicationClientProxy struct {
	http *internal.HTTPClient
}

func newApplicationClientProxy(httpClient *internal.HTTPClient) *applicationClientProxy {
	return &applicationClientProxy{http: httpClient}
}

const applicationServicePath = "/application"

func (p *applicationClientProxy) getApplicationDetails(ctx context.Context) (*ApplicationDetails, error) {
	var result ApplicationDetails
	if err := p.http.Get(ctx, applicationServicePath, &result); err != nil {
		return nil, fmt.Errorf("get application details: %w", err)
	}
	return &result, nil
}
