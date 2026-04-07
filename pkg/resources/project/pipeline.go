// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: FSL-1.1-ALv2

package project

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/goccy/go-yaml"
	gitlab "gitlab.com/gitlab-org/api/client-go"

	"github.com/platform-engineering-labs/formae-plugin-gitlab/pkg/config"
	"github.com/platform-engineering-labs/formae-plugin-gitlab/pkg/provisioner"
	"github.com/platform-engineering-labs/formae/pkg/plugin/resource"
)

const PipelineResourceType = "GitLab::Project::Pipeline"

// pipelineKeyOrder defines the conventional key ordering for GitLab CI YAML.
var pipelineKeyOrder = []string{"stages", "workflow", "include", "default", "variables"}

func init() {
	provisioner.Register(PipelineResourceType, func(client *gitlab.Client, cfg *config.Config) provisioner.Provisioner {
		return &pipelineProvisioner{client: client, cfg: cfg}
	})
}

type pipelineProvisioner struct {
	client *gitlab.Client
	cfg    *config.Config
}

func (p *pipelineProvisioner) Create(ctx context.Context, req *resource.CreateRequest) (*resource.CreateResult, error) {
	var props map[string]interface{}
	if err := json.Unmarshal(req.Properties, &props); err != nil {
		return provisioner.CreateFailure(resource.OperationErrorCodeInvalidRequest, fmt.Sprintf("invalid properties: %v", err)), nil
	}

	path, _ := props["path"].(string)
	if path == "" {
		path = ".gitlab-ci.yml"
	}

	yamlContent, err := marshalPipelineYAML(props)
	if err != nil {
		return provisioner.CreateFailure(resource.OperationErrorCodeInternalFailure, fmt.Sprintf("failed to marshal YAML: %v", err)), nil
	}

	opts := &gitlab.CreateFileOptions{
		Branch:        gitlab.Ptr("main"),
		Content:       gitlab.Ptr(string(yamlContent)),
		CommitMessage: gitlab.Ptr("managed by formae"),
	}

	_, _, err = p.client.RepositoryFiles.CreateFile(p.cfg.ProjectPath(), path, opts)
	if err != nil {
		return provisioner.CreateFailure(provisioner.ClassifyError(err), err.Error()), nil
	}

	return provisioner.CreateSuccess(path, req.Properties), nil
}

func (p *pipelineProvisioner) Read(ctx context.Context, req *resource.ReadRequest) (*resource.ReadResult, error) {
	file, resp, err := p.client.RepositoryFiles.GetFile(p.cfg.ProjectPath(), req.NativeID, &gitlab.GetFileOptions{})
	if err = provisioner.WrapError(err, resp); err != nil {
		if provisioner.IsNotFound(err) {
			return provisioner.ReadNotFound(PipelineResourceType), nil
		}
		return &resource.ReadResult{ResourceType: PipelineResourceType, ErrorCode: provisioner.ClassifyError(err)}, nil
	}

	content := file.Content
	if file.Encoding == "base64" {
		decoded, err := base64.StdEncoding.DecodeString(file.Content)
		if err != nil {
			return &resource.ReadResult{ResourceType: PipelineResourceType, ErrorCode: resource.OperationErrorCodeInternalFailure}, nil
		}
		content = string(decoded)
	}

	var pipelineMap map[string]interface{}
	if err := yaml.Unmarshal([]byte(content), &pipelineMap); err != nil {
		return &resource.ReadResult{ResourceType: PipelineResourceType, ErrorCode: resource.OperationErrorCodeInternalFailure}, nil
	}

	// Separate jobs from top-level keys — anything not in the reserved set is a job
	jobs := make(map[string]interface{})
	reserved := map[string]bool{
		"stages": true, "workflow": true, "include": true,
		"default": true, "variables": true, "path": true,
	}
	for k, v := range pipelineMap {
		if !reserved[k] {
			jobs[k] = v
			delete(pipelineMap, k)
		}
	}
	if len(jobs) > 0 {
		pipelineMap["jobs"] = jobs
	}

	pipelineMap["path"] = req.NativeID

	return provisioner.ReadSuccess(PipelineResourceType, provisioner.MustMarshal(pipelineMap)), nil
}

func (p *pipelineProvisioner) Update(ctx context.Context, req *resource.UpdateRequest) (*resource.UpdateResult, error) {
	var props map[string]interface{}
	if err := json.Unmarshal(req.DesiredProperties, &props); err != nil {
		return provisioner.UpdateFailure(req.NativeID, resource.OperationErrorCodeInvalidRequest, fmt.Sprintf("invalid properties: %v", err)), nil
	}

	yamlContent, err := marshalPipelineYAML(props)
	if err != nil {
		return provisioner.UpdateFailure(req.NativeID, resource.OperationErrorCodeInternalFailure, fmt.Sprintf("failed to marshal YAML: %v", err)), nil
	}

	opts := &gitlab.UpdateFileOptions{
		Branch:        gitlab.Ptr("main"),
		Content:       gitlab.Ptr(string(yamlContent)),
		CommitMessage: gitlab.Ptr("managed by formae"),
	}

	_, _, err = p.client.RepositoryFiles.UpdateFile(p.cfg.ProjectPath(), req.NativeID, opts)
	if err != nil {
		return provisioner.UpdateFailure(req.NativeID, provisioner.ClassifyError(err), err.Error()), nil
	}

	return provisioner.UpdateSuccess(req.NativeID, req.DesiredProperties), nil
}

func (p *pipelineProvisioner) Delete(ctx context.Context, req *resource.DeleteRequest) (*resource.DeleteResult, error) {
	opts := &gitlab.DeleteFileOptions{
		Branch:        gitlab.Ptr("main"),
		CommitMessage: gitlab.Ptr("removed by formae"),
	}

	resp, err := p.client.RepositoryFiles.DeleteFile(p.cfg.ProjectPath(), req.NativeID, opts)
	if err = provisioner.WrapError(err, resp); err != nil {
		if provisioner.IsNotFound(err) {
			return provisioner.DeleteSuccess(req.NativeID), nil
		}
		return provisioner.DeleteFailure(req.NativeID, provisioner.ClassifyError(err), err.Error()), nil
	}

	return provisioner.DeleteSuccess(req.NativeID), nil
}

func (p *pipelineProvisioner) List(ctx context.Context, req *resource.ListRequest) (*resource.ListResult, error) {
	// Check if .gitlab-ci.yml exists
	_, resp, err := p.client.RepositoryFiles.GetFile(p.cfg.ProjectPath(), ".gitlab-ci.yml", &gitlab.GetFileOptions{})
	if err = provisioner.WrapError(err, resp); err != nil {
		if provisioner.IsNotFound(err) {
			return &resource.ListResult{NativeIDs: []string{}}, nil
		}
		return nil, fmt.Errorf("failed to check pipeline file: %w", err)
	}

	return &resource.ListResult{NativeIDs: []string{".gitlab-ci.yml"}}, nil
}

func (p *pipelineProvisioner) Status(_ context.Context, req *resource.StatusRequest) (*resource.StatusResult, error) {
	return provisioner.StatusSuccess(req.NativeID), nil
}

// marshalPipelineYAML converts pipeline properties to YAML with conventional
// key ordering (stages, workflow, include, default, variables, then jobs).
func marshalPipelineYAML(data map[string]interface{}) ([]byte, error) {
	// Remove path — it's metadata, not part of the YAML content
	delete(data, "path")

	var ms yaml.MapSlice

	// Add top-level keys in conventional order
	for _, key := range pipelineKeyOrder {
		val, ok := data[key]
		if !ok {
			continue
		}
		ms = append(ms, yaml.MapItem{Key: key, Value: val})
	}

	// Add jobs — flatten from "jobs" map to top-level keys
	if jobs, ok := data["jobs"].(map[string]interface{}); ok {
		for name, job := range jobs {
			ms = append(ms, yaml.MapItem{Key: name, Value: job})
		}
	}

	// Add any remaining non-job, non-ordered keys
	for key, val := range data {
		if key == "jobs" {
			continue
		}
		inOrder := false
		for _, k := range pipelineKeyOrder {
			if key == k {
				inOrder = true
				break
			}
		}
		if inOrder {
			continue
		}
		ms = append(ms, yaml.MapItem{Key: key, Value: val})
	}

	return yaml.Marshal(ms)
}
