// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: FSL-1.1-ALv2

package project

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	gitlab "gitlab.com/gitlab-org/api/client-go"

	"github.com/platform-engineering-labs/formae-plugin-gitlab/pkg/config"
	"github.com/platform-engineering-labs/formae-plugin-gitlab/pkg/provisioner"
	"github.com/platform-engineering-labs/formae/pkg/plugin/resource"
)

const EnvironmentResourceType = "GitLab::Project::Environment"

func init() {
	provisioner.Register(EnvironmentResourceType, func(client *gitlab.Client, cfg *config.Config) provisioner.Provisioner {
		return &environmentProvisioner{client: client, cfg: cfg}
	})
}

type environmentProvisioner struct {
	client *gitlab.Client
	cfg    *config.Config
}

func (p *environmentProvisioner) Create(ctx context.Context, req *resource.CreateRequest) (*resource.CreateResult, error) {
	var props map[string]interface{}
	if err := json.Unmarshal(req.Properties, &props); err != nil {
		return provisioner.CreateFailure(resource.OperationErrorCodeInvalidRequest, fmt.Sprintf("invalid properties: %v", err)), nil
	}

	name, _ := props["name"].(string)
	if name == "" {
		return provisioner.CreateFailure(resource.OperationErrorCodeInvalidRequest, "name is required"), nil
	}

	opts := &gitlab.CreateEnvironmentOptions{
		Name: gitlab.Ptr(name),
	}
	if externalURL, ok := props["externalUrl"].(string); ok && externalURL != "" {
		opts.ExternalURL = gitlab.Ptr(externalURL)
	}
	if tier, ok := props["tier"].(string); ok && tier != "" {
		opts.Tier = gitlab.Ptr(tier)
	}

	env, _, err := p.client.Environments.CreateEnvironment(p.cfg.ProjectPath(), opts)
	if err != nil {
		return provisioner.CreateFailure(provisioner.ClassifyError(err), err.Error()), nil
	}

	nativeID := strconv.FormatInt(env.ID, 10)
	responseProps := envToProps(env)

	return provisioner.CreateSuccess(nativeID, provisioner.MustMarshal(responseProps)), nil
}

func (p *environmentProvisioner) Read(ctx context.Context, req *resource.ReadRequest) (*resource.ReadResult, error) {
	envID, err := strconv.ParseInt(req.NativeID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid environment NativeID %q: %w", req.NativeID, err)
	}

	env, resp, err := p.client.Environments.GetEnvironment(p.cfg.ProjectPath(), envID)
	if err = provisioner.WrapError(err, resp); err != nil {
		if provisioner.IsNotFound(err) {
			return provisioner.ReadNotFound(EnvironmentResourceType), nil
		}
		return &resource.ReadResult{ResourceType: EnvironmentResourceType, ErrorCode: provisioner.ClassifyError(err)}, nil
	}

	responseProps := envToProps(env)
	return provisioner.ReadSuccess(EnvironmentResourceType, provisioner.MustMarshal(responseProps)), nil
}

func (p *environmentProvisioner) Update(ctx context.Context, req *resource.UpdateRequest) (*resource.UpdateResult, error) {
	envID, err := strconv.ParseInt(req.NativeID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid environment NativeID %q: %w", req.NativeID, err)
	}

	var props map[string]interface{}
	if err := json.Unmarshal(req.DesiredProperties, &props); err != nil {
		return provisioner.UpdateFailure(req.NativeID, resource.OperationErrorCodeInvalidRequest, fmt.Sprintf("invalid properties: %v", err)), nil
	}

	opts := &gitlab.EditEnvironmentOptions{}
	if externalURL, ok := props["externalUrl"].(string); ok {
		opts.ExternalURL = gitlab.Ptr(externalURL)
	}
	if tier, ok := props["tier"].(string); ok {
		opts.Tier = gitlab.Ptr(tier)
	}

	env, _, err := p.client.Environments.EditEnvironment(p.cfg.ProjectPath(), envID, opts)
	if err != nil {
		return provisioner.UpdateFailure(req.NativeID, provisioner.ClassifyError(err), err.Error()), nil
	}

	responseProps := envToProps(env)
	return provisioner.UpdateSuccess(req.NativeID, provisioner.MustMarshal(responseProps)), nil
}

func (p *environmentProvisioner) Delete(ctx context.Context, req *resource.DeleteRequest) (*resource.DeleteResult, error) {
	envID, err := strconv.ParseInt(req.NativeID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid environment NativeID %q: %w", req.NativeID, err)
	}

	resp, err := p.client.Environments.DeleteEnvironment(p.cfg.ProjectPath(), envID)
	if err = provisioner.WrapError(err, resp); err != nil {
		if provisioner.IsNotFound(err) {
			return provisioner.DeleteSuccess(req.NativeID), nil
		}
		return provisioner.DeleteFailure(req.NativeID, provisioner.ClassifyError(err), err.Error()), nil
	}

	return provisioner.DeleteSuccess(req.NativeID), nil
}

func (p *environmentProvisioner) List(ctx context.Context, req *resource.ListRequest) (*resource.ListResult, error) {
	opts := &gitlab.ListEnvironmentsOptions{
		ListOptions: gitlab.ListOptions{PerPage: 100},
	}

	var allIDs []string

	for {
		envs, resp, err := p.client.Environments.ListEnvironments(p.cfg.ProjectPath(), opts)
		if err != nil {
			return nil, fmt.Errorf("failed to list environments: %w", err)
		}
		for _, env := range envs {
			allIDs = append(allIDs, strconv.FormatInt(env.ID, 10))
		}
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return &resource.ListResult{NativeIDs: allIDs}, nil
}

func (p *environmentProvisioner) Status(_ context.Context, req *resource.StatusRequest) (*resource.StatusResult, error) {
	return provisioner.StatusSuccess(req.NativeID), nil
}

func envToProps(env *gitlab.Environment) map[string]interface{} {
	props := map[string]interface{}{
		"name": env.Name,
	}
	if env.ExternalURL != "" {
		props["externalUrl"] = env.ExternalURL
	}
	if env.Tier != "" {
		props["tier"] = env.Tier
	}
	return props
}
