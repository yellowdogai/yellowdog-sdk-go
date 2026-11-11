package account

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/yellowdogai/yellowdog-sdk-go/internal"
	"github.com/yellowdogai/yellowdog-sdk-go/model"
)

// accountClientProxy makes the requests for the generated AccountClient, which delegates each
// method to the method here of the same name with its first letter lowercased.
type accountClientProxy struct {
	http *internal.HTTPClient
}

func newAccountClientProxy(httpClient *internal.HTTPClient) *accountClientProxy {
	return &accountClientProxy{http: httpClient}
}

const (
	accountServicePath      = "/account"
	usersPathSegment        = "users"
	applicationsPathSegment = "applications"
	rolesPathSegment        = "roles"
	groupsPathSegment       = "groups"
)

// --- Users ------------------------------------------------------------------------------------

func (p *accountClientProxy) getUser(ctx context.Context, userId string) (model.User, error) {
	var raw json.RawMessage
	path := fmt.Sprintf("%s/%s/%s", accountServicePath, usersPathSegment, userId)
	if err := p.http.Get(ctx, path, &raw); err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	return model.UnmarshalUser(raw)
}

func (p *accountClientProxy) getUsers(search *UserSearch) *internal.SearchClient[model.User] {
	path := fmt.Sprintf("%s/%s", accountServicePath, usersPathSegment)
	return internal.NewPolymorphicSearchClient[model.User](p.http, path, search, model.UnmarshalUser)
}

func (p *accountClientProxy) getUserGroups(userId string) *internal.SearchClient[GroupSummary] {
	path := fmt.Sprintf("%s/%s/%s/groups", accountServicePath, usersPathSegment, userId)
	return internal.NewSearchClient[GroupSummary](p.http, path, nil)
}

// --- Applications -----------------------------------------------------------------------------

func (p *accountClientProxy) getApplication(ctx context.Context, applicationId string) (*model.Application, error) {
	var result model.Application
	path := fmt.Sprintf("%s/%s/%s", accountServicePath, applicationsPathSegment, applicationId)
	if err := p.http.Get(ctx, path, &result); err != nil {
		return nil, fmt.Errorf("get application: %w", err)
	}
	return &result, nil
}

func (p *accountClientProxy) updateApplication(ctx context.Context, applicationId string, request *UpdateApplicationRequest) (*model.Application, error) {
	var result model.Application
	path := fmt.Sprintf("%s/%s/%s", accountServicePath, applicationsPathSegment, applicationId)
	if err := p.http.Put(ctx, path, request, &result); err != nil {
		return nil, fmt.Errorf("update application: %w", err)
	}
	return &result, nil
}

func (p *accountClientProxy) addApplication(ctx context.Context, request *AddApplicationRequest) (*AddApplicationResponse, error) {
	var result AddApplicationResponse
	path := fmt.Sprintf("%s/%s", accountServicePath, applicationsPathSegment)
	if err := p.http.Post(ctx, path, request, &result); err != nil {
		return nil, fmt.Errorf("add application: %w", err)
	}
	return &result, nil
}

func (p *accountClientProxy) regenerateApplicationApiKey(ctx context.Context, applicationId string) (*model.ApiKey, error) {
	var result model.ApiKey
	path := fmt.Sprintf("%s/%s/%s/key", accountServicePath, applicationsPathSegment, applicationId)
	if err := p.http.Post(ctx, path, nil, &result); err != nil {
		return nil, fmt.Errorf("regenerate application api key: %w", err)
	}
	return &result, nil
}

func (p *accountClientProxy) getApplications(search *ApplicationSearch) *internal.SearchClient[model.Application] {
	path := fmt.Sprintf("%s/%s", accountServicePath, applicationsPathSegment)
	return internal.NewSearchClient[model.Application](p.http, path, search)
}

func (p *accountClientProxy) getApplicationGroups(applicationId string) *internal.SearchClient[GroupSummary] {
	path := fmt.Sprintf("%s/%s/%s/groups", accountServicePath, applicationsPathSegment, applicationId)
	return internal.NewSearchClient[GroupSummary](p.http, path, nil)
}

func (p *accountClientProxy) deleteApplication(ctx context.Context, applicationId string) error {
	path := fmt.Sprintf("%s/%s/%s", accountServicePath, applicationsPathSegment, applicationId)
	if err := p.http.Delete(ctx, path); err != nil {
		return fmt.Errorf("delete application: %w", err)
	}
	return nil
}

// --- Permissions and roles ----------------------------------------------------------------------

func (p *accountClientProxy) listPermissions(ctx context.Context) ([]*PermissionDetail, error) {
	var result []*PermissionDetail
	if err := p.http.Get(ctx, accountServicePath+"/permissions", &result); err != nil {
		return nil, fmt.Errorf("list permissions: %w", err)
	}
	return result, nil
}

func (p *accountClientProxy) getRole(ctx context.Context, roleId string) (*model.Role, error) {
	var result model.Role
	path := fmt.Sprintf("%s/%s/%s", accountServicePath, rolesPathSegment, roleId)
	if err := p.http.Get(ctx, path, &result); err != nil {
		return nil, fmt.Errorf("get role: %w", err)
	}
	return &result, nil
}

func (p *accountClientProxy) getRoles(search *RoleSearch) *internal.SearchClient[model.Role] {
	path := fmt.Sprintf("%s/%s", accountServicePath, rolesPathSegment)
	return internal.NewSearchClient[model.Role](p.http, path, search)
}

func (p *accountClientProxy) getRoleGroups(roleId string) *internal.SearchClient[GroupSummary] {
	path := fmt.Sprintf("%s/%s/%s/groups", accountServicePath, rolesPathSegment, roleId)
	return internal.NewSearchClient[GroupSummary](p.http, path, nil)
}

// --- Groups -----------------------------------------------------------------------------------

func (p *accountClientProxy) getGroup(ctx context.Context, groupId string) (*model.Group, error) {
	var result model.Group
	path := fmt.Sprintf("%s/%s/%s", accountServicePath, groupsPathSegment, groupId)
	if err := p.http.Get(ctx, path, &result); err != nil {
		return nil, fmt.Errorf("get group: %w", err)
	}
	return &result, nil
}

func (p *accountClientProxy) addGroup(ctx context.Context, request *AddGroupRequest) (*model.Group, error) {
	var result model.Group
	path := fmt.Sprintf("%s/%s", accountServicePath, groupsPathSegment)
	if err := p.http.Post(ctx, path, request, &result); err != nil {
		return nil, fmt.Errorf("add group: %w", err)
	}
	return &result, nil
}

func (p *accountClientProxy) updateGroup(ctx context.Context, groupId string, request *UpdateGroupRequest) (*model.Group, error) {
	var result model.Group
	path := fmt.Sprintf("%s/%s/%s", accountServicePath, groupsPathSegment, groupId)
	if err := p.http.Put(ctx, path, request, &result); err != nil {
		return nil, fmt.Errorf("update group: %w", err)
	}
	return &result, nil
}

func (p *accountClientProxy) getGroups(search *GroupSearch) *internal.SearchClient[GroupSummary] {
	path := fmt.Sprintf("%s/%s", accountServicePath, groupsPathSegment)
	return internal.NewSearchClient[GroupSummary](p.http, path, search)
}

func (p *accountClientProxy) getGroupUsers(groupId string) *internal.SearchClient[model.User] {
	path := fmt.Sprintf("%s/%s/%s/users", accountServicePath, groupsPathSegment, groupId)
	return internal.NewPolymorphicSearchClient[model.User](p.http, path, nil, model.UnmarshalUser)
}

func (p *accountClientProxy) getGroupApplications(groupId string) *internal.SearchClient[model.Application] {
	path := fmt.Sprintf("%s/%s/%s/applications", accountServicePath, groupsPathSegment, groupId)
	return internal.NewSearchClient[model.Application](p.http, path, nil)
}

func (p *accountClientProxy) getGroupRoles(groupId string) *internal.SearchClient[model.Role] {
	path := fmt.Sprintf("%s/%s/%s/roles", accountServicePath, groupsPathSegment, groupId)
	return internal.NewSearchClient[model.Role](p.http, path, nil)
}

// --- Group membership ---------------------------------------------------------------------------

func (p *accountClientProxy) addUserToGroup(ctx context.Context, groupId string, userId string) error {
	path := fmt.Sprintf("%s/%s/%s/%s/%s", accountServicePath, groupsPathSegment, groupId, usersPathSegment, userId)
	if err := p.http.Put(ctx, path, nil, nil); err != nil {
		return fmt.Errorf("add user to group: %w", err)
	}
	return nil
}

func (p *accountClientProxy) addApplicationToGroup(ctx context.Context, groupId string, applicationId string) error {
	path := fmt.Sprintf("%s/%s/%s/%s/%s", accountServicePath, groupsPathSegment, groupId, applicationsPathSegment, applicationId)
	if err := p.http.Put(ctx, path, nil, nil); err != nil {
		return fmt.Errorf("add application to group: %w", err)
	}
	return nil
}

func (p *accountClientProxy) addRoleToGroup(ctx context.Context, groupId string, roleId string, scope *RoleScope) error {
	path := fmt.Sprintf("%s/%s/%s/%s/%s", accountServicePath, groupsPathSegment, groupId, rolesPathSegment, roleId)
	if err := p.http.Put(ctx, path, scope, nil); err != nil {
		return fmt.Errorf("add role to group: %w", err)
	}
	return nil
}

func (p *accountClientProxy) removeUserFromGroup(ctx context.Context, groupId string, userId string) error {
	path := fmt.Sprintf("%s/%s/%s/%s/%s", accountServicePath, groupsPathSegment, groupId, usersPathSegment, userId)
	if err := p.http.Delete(ctx, path); err != nil {
		return fmt.Errorf("remove user from group: %w", err)
	}
	return nil
}

func (p *accountClientProxy) removeApplicationFromGroup(ctx context.Context, groupId string, applicationId string) error {
	path := fmt.Sprintf("%s/%s/%s/%s/%s", accountServicePath, groupsPathSegment, groupId, applicationsPathSegment, applicationId)
	if err := p.http.Delete(ctx, path); err != nil {
		return fmt.Errorf("remove application from group: %w", err)
	}
	return nil
}

func (p *accountClientProxy) removeRoleFromGroup(ctx context.Context, groupId string, roleId string) error {
	path := fmt.Sprintf("%s/%s/%s/%s/%s", accountServicePath, groupsPathSegment, groupId, rolesPathSegment, roleId)
	if err := p.http.Delete(ctx, path); err != nil {
		return fmt.Errorf("remove role from group: %w", err)
	}
	return nil
}

func (p *accountClientProxy) deleteGroup(ctx context.Context, groupId string) error {
	path := fmt.Sprintf("%s/%s/%s", accountServicePath, groupsPathSegment, groupId)
	if err := p.http.Delete(ctx, path); err != nil {
		return fmt.Errorf("delete group: %w", err)
	}
	return nil
}
