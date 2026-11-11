package account

import (
	"context"
	"fmt"

	"github.com/yellowdogai/yellowdog-sdk-go/internal"
	"github.com/yellowdogai/yellowdog-sdk-go/model"
)

// namespacesClientProxy makes the requests for the generated NamespacesClient, which delegates
// each method to the method here of the same name with its first letter lowercased.
type namespacesClientProxy struct {
	http *internal.HTTPClient
}

func newNamespacesClientProxy(httpClient *internal.HTTPClient) *namespacesClientProxy {
	return &namespacesClientProxy{http: httpClient}
}

const namespacesServicePath = "/namespaces"

func (p *namespacesClientProxy) getNamespace(ctx context.Context, namespaceId string) (*model.Namespace, error) {
	var result model.Namespace
	path := fmt.Sprintf("%s/%s", namespacesServicePath, namespaceId)
	if err := p.http.Get(ctx, path, &result); err != nil {
		return nil, fmt.Errorf("get namespace: %w", err)
	}
	return &result, nil
}

func (p *namespacesClientProxy) getNamespaces(search *NamespaceSearch) *internal.SearchClient[model.Namespace] {
	return internal.NewSearchClient[model.Namespace](p.http, namespacesServicePath, search)
}

func (p *namespacesClientProxy) createNamespace(ctx context.Context, request *CreateNamespaceRequest) (string, error) {
	var result string
	if err := p.http.Post(ctx, namespacesServicePath, request, &result); err != nil {
		return result, fmt.Errorf("create namespace: %w", err)
	}
	return result, nil
}

func (p *namespacesClientProxy) deleteNamespace(ctx context.Context, namespaceId string) error {
	path := fmt.Sprintf("%s/%s", namespacesServicePath, namespaceId)
	if err := p.http.Delete(ctx, path); err != nil {
		return fmt.Errorf("delete namespace: %w", err)
	}
	return nil
}

func (p *namespacesClientProxy) saveNamespacePolicy(ctx context.Context, namespacePolicy *model.NamespacePolicy) error {
	path := fmt.Sprintf("%s/%s/policy", namespacesServicePath, namespacePolicy.Namespace)
	if err := p.http.Put(ctx, path, namespacePolicy, nil); err != nil {
		return fmt.Errorf("save namespace policy: %w", err)
	}
	return nil
}

func (p *namespacesClientProxy) getNamespacePolicy(ctx context.Context, namespace string) (*model.NamespacePolicy, error) {
	var result model.NamespacePolicy
	path := fmt.Sprintf("%s/%s/policy", namespacesServicePath, namespace)
	if err := p.http.Get(ctx, path, &result); err != nil {
		return nil, fmt.Errorf("get namespace policy: %w", err)
	}
	return &result, nil
}

func (p *namespacesClientProxy) deleteNamespacePolicy(ctx context.Context, namespace string) error {
	path := fmt.Sprintf("%s/%s/policy", namespacesServicePath, namespace)
	if err := p.http.Delete(ctx, path); err != nil {
		return fmt.Errorf("delete namespace policy: %w", err)
	}
	return nil
}

func (p *namespacesClientProxy) getNamespacePolicies(search *NamespacePolicySearch) *internal.SearchClient[model.NamespacePolicy] {
	return internal.NewSearchClient[model.NamespacePolicy](p.http, namespacesServicePath+"/policies", search)
}
