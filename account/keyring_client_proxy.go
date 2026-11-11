package account

import (
	"context"
	"errors"
	"fmt"

	"github.com/yellowdogai/yellowdog-sdk-go/internal"
	"github.com/yellowdogai/yellowdog-sdk-go/model"
	"github.com/yellowdogai/yellowdog-sdk-go/optional"
)

// keyringClientProxy makes the requests for the generated KeyringClient, which delegates each
// method to the method here of the same name with its first letter lowercased.
type keyringClientProxy struct {
	http *internal.HTTPClient
}

func newKeyringClientProxy(httpClient *internal.HTTPClient) *keyringClientProxy {
	return &keyringClientProxy{http: httpClient}
}

const keyringServicePath = "/keyrings"

func (p *keyringClientProxy) getKeyring(ctx context.Context, keyringNameOrId string) (*model.Keyring, error) {
	var result model.Keyring
	path := fmt.Sprintf("%s/%s", keyringServicePath, keyringNameOrId)
	if err := p.http.Get(ctx, path, &result); err != nil {
		return nil, fmt.Errorf("get keyring: %w", err)
	}
	return &result, nil
}

func (p *keyringClientProxy) updateKeyring(ctx context.Context, keyringId string, request *UpdateKeyringRequest) (*model.Keyring, error) {
	var result model.Keyring
	path := fmt.Sprintf("%s/%s", keyringServicePath, keyringId)
	if err := p.http.Put(ctx, path, request, &result); err != nil {
		return nil, fmt.Errorf("update keyring: %w", err)
	}
	return &result, nil
}

// createKeyring returns the keyring from the create response, discarding the generated secret;
// callers that need the secret use addKeyring.
func (p *keyringClientProxy) createKeyring(ctx context.Context, name string, description string) (*model.Keyring, error) {
	response, err := p.addKeyring(ctx, name, description)
	if err != nil {
		return nil, err
	}
	keyring, ok := response.Keyring.Get()
	if !ok {
		return nil, errors.New("account: keyring creation returned no keyring")
	}
	return &keyring, nil
}

func (p *keyringClientProxy) addKeyring(ctx context.Context, name string, description string) (*CreateKeyringResponse, error) {
	request := &CreateKeyringRequest{Name: name}
	if description != "" {
		request.Description = optional.Of(description)
	}

	var result CreateKeyringResponse
	if err := p.http.Post(ctx, keyringServicePath, request, &result); err != nil {
		return nil, fmt.Errorf("add keyring: %w", err)
	}
	return &result, nil
}

func (p *keyringClientProxy) getKeyrings(search *KeyringSearch) *internal.SearchClient[KeyringSummary] {
	return internal.NewSearchClient[KeyringSummary](p.http, keyringServicePath, search, internal.Param{Name: "sliced", Value: "true"})
}

func (p *keyringClientProxy) deleteKeyring(ctx context.Context, keyring *model.Keyring) error {
	return p.deleteKeyringByName(ctx, keyring.Name.Or(""))
}

func (p *keyringClientProxy) deleteKeyringByName(ctx context.Context, keyringName string) error {
	path := fmt.Sprintf("%s/%s", keyringServicePath, keyringName)
	if err := p.http.Delete(ctx, path); err != nil {
		return fmt.Errorf("delete keyring: %w", err)
	}
	return nil
}

// grantApplicationAccessToKeyring wraps the API key in a GrantApplicationAccessRequest because the
// endpoint rejects a bare ApiKey.
func (p *keyringClientProxy) grantApplicationAccessToKeyring(ctx context.Context, keyringName string, applicationId string, applicationApiKey *model.ApiKey) (*model.Keyring, error) {
	request := &GrantApplicationAccessRequest{ApplicationApiKey: *applicationApiKey}

	var result model.Keyring
	path := fmt.Sprintf("%s/%s/access/applications/%s", keyringServicePath, keyringName, applicationId)
	if err := p.http.Put(ctx, path, request, &result); err != nil {
		return nil, fmt.Errorf("grant application access to keyring: %w", err)
	}
	return &result, nil
}

func (p *keyringClientProxy) putCredential(ctx context.Context, keyring *model.Keyring, credential model.Credential) (*model.Keyring, error) {
	return p.putCredentialByName(ctx, keyring.Name.Or(""), credential)
}

func (p *keyringClientProxy) putCredentialByName(ctx context.Context, keyringName string, credential model.Credential) (*model.Keyring, error) {
	var result model.Keyring
	path := fmt.Sprintf("%s/%s/credentials", keyringServicePath, keyringName)
	if err := p.http.Put(ctx, path, credential, &result); err != nil {
		return nil, fmt.Errorf("put credential: %w", err)
	}
	return &result, nil
}

func (p *keyringClientProxy) deleteCredential(ctx context.Context, keyring *model.Keyring, credentialName string) (*model.Keyring, error) {
	return p.deleteCredentialByName(ctx, keyring.Name.Or(""), credentialName)
}

func (p *keyringClientProxy) deleteCredentialByName(ctx context.Context, keyringName string, credentialName string) (*model.Keyring, error) {
	var result model.Keyring
	path := fmt.Sprintf("%s/%s/credentials/%s", keyringServicePath, keyringName, credentialName)
	if err := p.http.DeleteWithResult(ctx, path, &result); err != nil {
		return nil, fmt.Errorf("delete credential: %w", err)
	}
	return &result, nil
}
