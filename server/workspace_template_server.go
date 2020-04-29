package server

import (
	"context"
	"github.com/onepanelio/core/api"
	v1 "github.com/onepanelio/core/pkg"
	"github.com/onepanelio/core/server/auth"
	"time"
)

type WorkspaceTemplateServer struct{}

func apiWorkspaceTemplate(wt *v1.WorkspaceTemplate) *api.WorkspaceTemplate {
	res := &api.WorkspaceTemplate{
		Uid:       wt.UID,
		Name:      wt.Name,
		Version:   wt.Version,
		Manifest:  wt.Manifest,
		IsLatest:  wt.IsLatest,
		CreatedAt: wt.CreatedAt.UTC().Format(time.RFC3339),
	}

	if wt.WorkflowTemplate != nil {
		res.WorkflowTemplate = apiWorkflowTemplate(wt.WorkflowTemplate)
	}

	return res
}

func NewWorkspaceTemplateServer() *WorkspaceTemplateServer {
	return &WorkspaceTemplateServer{}
}

func (s *WorkspaceTemplateServer) CreateWorkspaceTemplate(ctx context.Context, req *api.CreateWorkspaceTemplateRequest) (*api.WorkspaceTemplate, error) {
	client := ctx.Value("kubeClient").(*v1.Client)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "create", "argoproj.io", "workflowtemplates", "")
	if err != nil || !allowed {
		return nil, err
	}

	workspaceTemplate := &v1.WorkspaceTemplate{
		Name:     req.WorkspaceTemplate.Name,
		Manifest: req.WorkspaceTemplate.Manifest,
	}
	workspaceTemplate, err = client.CreateWorkspaceTemplate(req.Namespace, workspaceTemplate)
	if err != nil {
		return nil, err
	}

	req.WorkspaceTemplate = apiWorkspaceTemplate(workspaceTemplate)

	return req.WorkspaceTemplate, nil
}

func (s WorkspaceTemplateServer) GenerateWorkspaceTemplateWorkflowTemplate(ctx context.Context, req *api.GenerateWorkspaceTemplateWorkflowTemplateRequest) (*api.WorkflowTemplate, error) {
	client := ctx.Value("kubeClient").(*v1.Client)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "get", "argoproj.io", "workflowtemplates", "")
	if err != nil || !allowed {
		return nil, err
	}

	workspaceTemplate := &v1.WorkspaceTemplate{
		Manifest: req.WorkspaceTemplateManifest,
	}
	workflowTemplate, err := client.GenerateWorkspaceTemplateWorkflowTemplate(workspaceTemplate)

	return apiWorkflowTemplate(workflowTemplate), nil
}
