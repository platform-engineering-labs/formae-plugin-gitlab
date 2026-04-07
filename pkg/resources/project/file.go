// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: FSL-1.1-ALv2

package project

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	gitlab "gitlab.com/gitlab-org/api/client-go"

	"github.com/platform-engineering-labs/formae-plugin-gitlab/pkg/config"
	"github.com/platform-engineering-labs/formae-plugin-gitlab/pkg/provisioner"
	"github.com/platform-engineering-labs/formae/pkg/plugin/resource"
)

const FileResourceType = "GitLab::Project::File"

func init() {
	provisioner.Register(FileResourceType, func(client *gitlab.Client, cfg *config.Config) provisioner.Provisioner {
		return &fileProvisioner{client: client, cfg: cfg}
	})
}

type fileProvisioner struct {
	client *gitlab.Client
	cfg    *config.Config
}

func (p *fileProvisioner) Create(ctx context.Context, req *resource.CreateRequest) (*resource.CreateResult, error) {
	var props map[string]interface{}
	if err := json.Unmarshal(req.Properties, &props); err != nil {
		return provisioner.CreateFailure(resource.OperationErrorCodeInvalidRequest, fmt.Sprintf("invalid properties: %v", err)), nil
	}

	path, _ := props["path"].(string)
	if path == "" {
		return provisioner.CreateFailure(resource.OperationErrorCodeInvalidRequest, "path is required"), nil
	}

	content, _ := props["content"].(string)
	commitMsg, _ := props["commitMessage"].(string)
	if commitMsg == "" {
		commitMsg = "managed by formae"
	}
	branch, _ := props["branch"].(string)

	if branch == "" {
		branch = "main"
	}

	opts := &gitlab.CreateFileOptions{
		Branch:        gitlab.Ptr(branch),
		Content:       gitlab.Ptr(content),
		CommitMessage: gitlab.Ptr(commitMsg),
	}

	_, _, err := p.client.RepositoryFiles.CreateFile(p.cfg.ProjectPath(), path, opts)
	if err != nil {
		return provisioner.CreateFailure(provisioner.ClassifyError(err), err.Error()), nil
	}

	responseProps := map[string]interface{}{
		"path":          path,
		"content":       content,
		"commitMessage": commitMsg,
	}
	if branch != "" {
		responseProps["branch"] = branch
	}

	return provisioner.CreateSuccess(path, provisioner.MustMarshal(responseProps)), nil
}

func (p *fileProvisioner) Read(ctx context.Context, req *resource.ReadRequest) (*resource.ReadResult, error) {
	opts := &gitlab.GetFileOptions{}

	file, resp, err := p.client.RepositoryFiles.GetFile(p.cfg.ProjectPath(), req.NativeID, opts)
	if err = provisioner.WrapError(err, resp); err != nil {
		if provisioner.IsNotFound(err) {
			return provisioner.ReadNotFound(FileResourceType), nil
		}
		return &resource.ReadResult{ResourceType: FileResourceType, ErrorCode: provisioner.ClassifyError(err)}, nil
	}

	content := file.Content
	if file.Encoding == "base64" {
		decoded, err := base64.StdEncoding.DecodeString(file.Content)
		if err != nil {
			return &resource.ReadResult{ResourceType: FileResourceType, ErrorCode: resource.OperationErrorCodeInternalFailure}, nil
		}
		content = string(decoded)
	}

	responseProps := map[string]interface{}{
		"path":    req.NativeID,
		"content": content,
	}

	return provisioner.ReadSuccess(FileResourceType, provisioner.MustMarshal(responseProps)), nil
}

func (p *fileProvisioner) Update(ctx context.Context, req *resource.UpdateRequest) (*resource.UpdateResult, error) {
	var props map[string]interface{}
	if err := json.Unmarshal(req.DesiredProperties, &props); err != nil {
		return provisioner.UpdateFailure(req.NativeID, resource.OperationErrorCodeInvalidRequest, fmt.Sprintf("invalid properties: %v", err)), nil
	}

	content, _ := props["content"].(string)
	commitMsg, _ := props["commitMessage"].(string)
	if commitMsg == "" {
		commitMsg = "managed by formae"
	}
	branch, _ := props["branch"].(string)

	if branch == "" {
		branch = "main"
	}

	opts := &gitlab.UpdateFileOptions{
		Branch:        gitlab.Ptr(branch),
		Content:       gitlab.Ptr(content),
		CommitMessage: gitlab.Ptr(commitMsg),
	}

	_, _, err := p.client.RepositoryFiles.UpdateFile(p.cfg.ProjectPath(), req.NativeID, opts)
	if err != nil {
		return provisioner.UpdateFailure(req.NativeID, provisioner.ClassifyError(err), err.Error()), nil
	}

	responseProps := map[string]interface{}{
		"path":          req.NativeID,
		"content":       content,
		"commitMessage": commitMsg,
	}
	if branch != "" {
		responseProps["branch"] = branch
	}

	return provisioner.UpdateSuccess(req.NativeID, provisioner.MustMarshal(responseProps)), nil
}

func (p *fileProvisioner) Delete(ctx context.Context, req *resource.DeleteRequest) (*resource.DeleteResult, error) {
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

func (p *fileProvisioner) List(ctx context.Context, req *resource.ListRequest) (*resource.ListResult, error) {
	opts := &gitlab.ListTreeOptions{
		ListOptions: gitlab.ListOptions{PerPage: 100},
		Recursive:   gitlab.Ptr(true),
	}

	var allIDs []string

	for {
		nodes, resp, err := p.client.Repositories.ListTree(p.cfg.ProjectPath(), opts)
		if err != nil {
			if provisioner.IsNotFound(err) {
				return &resource.ListResult{NativeIDs: []string{}}, nil
			}
			return nil, fmt.Errorf("failed to list repository files: %w", err)
		}
		for _, node := range nodes {
			if node.Type == "blob" {
				allIDs = append(allIDs, node.Path)
			}
		}
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return &resource.ListResult{NativeIDs: allIDs}, nil
}

func (p *fileProvisioner) Status(_ context.Context, req *resource.StatusRequest) (*resource.StatusResult, error) {
	return provisioner.StatusSuccess(req.NativeID), nil
}
