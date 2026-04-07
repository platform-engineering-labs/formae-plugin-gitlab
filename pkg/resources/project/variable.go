// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: Apache-2.0

package project

import (
	"context"
	"encoding/json"
	"fmt"

	gitlab "gitlab.com/gitlab-org/api/client-go"

	"github.com/platform-engineering-labs/formae-plugin-gitlab/pkg/config"
	"github.com/platform-engineering-labs/formae-plugin-gitlab/pkg/provisioner"
	"github.com/platform-engineering-labs/formae/pkg/plugin/resource"
)

const VariableResourceType = "GitLab::Project::Variable"

func init() {
	provisioner.Register(VariableResourceType, func(client *gitlab.Client, cfg *config.Config) provisioner.Provisioner {
		return &variableProvisioner{client: client, cfg: cfg}
	})
}

type variableProvisioner struct {
	client *gitlab.Client
	cfg    *config.Config
}

func (p *variableProvisioner) Create(ctx context.Context, req *resource.CreateRequest) (*resource.CreateResult, error) {
	var props map[string]interface{}
	if err := json.Unmarshal(req.Properties, &props); err != nil {
		return provisioner.CreateFailure(resource.OperationErrorCodeInvalidRequest, fmt.Sprintf("invalid properties: %v", err)), nil
	}

	key, _ := props["key"].(string)
	if key == "" {
		return provisioner.CreateFailure(resource.OperationErrorCodeInvalidRequest, "key is required"), nil
	}

	value, _ := props["value"].(string)
	protected, _ := props["protected"].(bool)
	masked, _ := props["masked"].(bool)
	envScope, _ := props["environmentScope"].(string)
	if envScope == "" {
		envScope = "*"
	}

	opts := &gitlab.CreateProjectVariableOptions{
		Key:              gitlab.Ptr(key),
		Value:            gitlab.Ptr(value),
		Protected:        gitlab.Ptr(protected),
		Masked:           gitlab.Ptr(masked),
		EnvironmentScope: gitlab.Ptr(envScope),
	}

	result, _, err := p.client.ProjectVariables.CreateVariable(p.cfg.ProjectPath(), opts)
	if err != nil {
		return provisioner.CreateFailure(provisioner.ClassifyError(err), err.Error()), nil
	}

	nativeID := variableNativeID(result.Key, result.EnvironmentScope)
	responseProps := variableToProps(result)

	return provisioner.CreateSuccess(nativeID, provisioner.MustMarshal(responseProps)), nil
}

func (p *variableProvisioner) Read(ctx context.Context, req *resource.ReadRequest) (*resource.ReadResult, error) {
	key, scope := parseVariableNativeID(req.NativeID)

	filter := &gitlab.GetProjectVariableOptions{}
	if scope != "*" {
		filter.Filter = &gitlab.VariableFilter{EnvironmentScope: scope}
	}

	result, _, err := p.client.ProjectVariables.GetVariable(p.cfg.ProjectPath(), key, filter)
	if err != nil {
		if provisioner.IsNotFound(err) {
			return provisioner.ReadNotFound(VariableResourceType), nil
		}
		return &resource.ReadResult{ResourceType: VariableResourceType, ErrorCode: provisioner.ClassifyError(err)}, nil
	}

	responseProps := variableToProps(result)
	return provisioner.ReadSuccess(VariableResourceType, provisioner.MustMarshal(responseProps)), nil
}

func (p *variableProvisioner) Update(ctx context.Context, req *resource.UpdateRequest) (*resource.UpdateResult, error) {
	key, scope := parseVariableNativeID(req.NativeID)

	var props map[string]interface{}
	if err := json.Unmarshal(req.DesiredProperties, &props); err != nil {
		return provisioner.UpdateFailure(req.NativeID, resource.OperationErrorCodeInvalidRequest, fmt.Sprintf("invalid properties: %v", err)), nil
	}

	value, _ := props["value"].(string)
	protected, _ := props["protected"].(bool)
	masked, _ := props["masked"].(bool)

	opts := &gitlab.UpdateProjectVariableOptions{
		Value:     gitlab.Ptr(value),
		Protected: gitlab.Ptr(protected),
		Masked:    gitlab.Ptr(masked),
	}
	if scope != "*" {
		opts.Filter = &gitlab.VariableFilter{EnvironmentScope: scope}
	}

	result, _, err := p.client.ProjectVariables.UpdateVariable(p.cfg.ProjectPath(), key, opts)
	if err != nil {
		return provisioner.UpdateFailure(req.NativeID, provisioner.ClassifyError(err), err.Error()), nil
	}

	responseProps := variableToProps(result)
	return provisioner.UpdateSuccess(req.NativeID, provisioner.MustMarshal(responseProps)), nil
}

func (p *variableProvisioner) Delete(ctx context.Context, req *resource.DeleteRequest) (*resource.DeleteResult, error) {
	key, scope := parseVariableNativeID(req.NativeID)

	opts := &gitlab.RemoveProjectVariableOptions{}
	if scope != "*" {
		opts.Filter = &gitlab.VariableFilter{EnvironmentScope: scope}
	}

	_, err := p.client.ProjectVariables.RemoveVariable(p.cfg.ProjectPath(), key, opts)
	if err != nil {
		if provisioner.IsNotFound(err) {
			return provisioner.DeleteSuccess(req.NativeID), nil
		}
		return provisioner.DeleteFailure(req.NativeID, provisioner.ClassifyError(err), err.Error()), nil
	}

	return provisioner.DeleteSuccess(req.NativeID), nil
}

func (p *variableProvisioner) List(ctx context.Context, req *resource.ListRequest) (*resource.ListResult, error) {
	var allIDs []string
	opts := &gitlab.ListProjectVariablesOptions{
		ListOptions: gitlab.ListOptions{PerPage: 100},
	}

	for {
		vars, resp, err := p.client.ProjectVariables.ListVariables(p.cfg.ProjectPath(), opts)
		if err != nil {
			return nil, fmt.Errorf("failed to list project variables: %w", err)
		}
		for _, v := range vars {
			allIDs = append(allIDs, variableNativeID(v.Key, v.EnvironmentScope))
		}
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return &resource.ListResult{NativeIDs: allIDs}, nil
}

func (p *variableProvisioner) Status(_ context.Context, req *resource.StatusRequest) (*resource.StatusResult, error) {
	return provisioner.StatusSuccess(req.NativeID), nil
}

// variableNativeID builds the native ID from key and scope.
// Uses "key" for scope "*", and "key:scope" for scoped variables.
func variableNativeID(key, scope string) string {
	if scope == "" || scope == "*" {
		return key
	}
	return key + ":" + scope
}

// parseVariableNativeID extracts the key and scope from a native ID.
func parseVariableNativeID(nativeID string) (key, scope string) {
	for i := len(nativeID) - 1; i >= 0; i-- {
		if nativeID[i] == ':' {
			return nativeID[:i], nativeID[i+1:]
		}
	}
	return nativeID, "*"
}

func variableToProps(v *gitlab.ProjectVariable) map[string]interface{} {
	return map[string]interface{}{
		"key":              v.Key,
		"value":            v.Value,
		"protected":        v.Protected,
		"masked":           v.Masked,
		"environmentScope": v.EnvironmentScope,
	}
}
