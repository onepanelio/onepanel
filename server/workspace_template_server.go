package server

import (
	"context"
	"github.com/onepanelio/core/api"
	v1 "github.com/onepanelio/core/pkg"
	"github.com/onepanelio/core/pkg/util/pagination"
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

func (s WorkspaceTemplateServer) GenerateWorkspaceTemplateWorkflowTemplate(ctx context.Context, req *api.GenerateWorkspaceTemplateWorkflowTemplateRequest) (*api.WorkflowTemplate, error) {
	client := ctx.Value("kubeClient").(*v1.Client)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "get", "argoproj.io", "workflowtemplates", "")
	if err != nil || !allowed {
		return nil, err
	}

	if req.WorkspaceTemplate.Manifest == "" {
		return &api.WorkflowTemplate{
			Manifest: "",
		}, nil
	}

	workspaceTemplate := &v1.WorkspaceTemplate{
		Manifest: req.WorkspaceTemplate.Manifest,
	}
	workflowTemplate, err := client.GenerateWorkspaceTemplateWorkflowTemplate(workspaceTemplate)

	if workflowTemplate == nil {
		return &api.WorkflowTemplate{
			Manifest: "",
		}, nil
	}

	return apiWorkflowTemplate(workflowTemplate), nil
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

func (s *WorkspaceTemplateServer) GetWorkspaceTemplate(ctx context.Context, req *api.GetWorkspaceTemplateRequest) (*api.WorkspaceTemplate, error) {
	client := ctx.Value("kubeClient").(*v1.Client)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "get", "argoproj.io", "workflowtemplates", "")
	if err != nil || !allowed {
		return nil, err
	}

	workspaceTemplate, err := client.GetWorkspaceTemplate(req.Namespace, req.Uid, req.Version)
	if err != nil {
		return nil, err
	}

	return apiWorkspaceTemplate(workspaceTemplate), nil
}

func (s *WorkspaceTemplateServer) ListWorkspaceTemplates(ctx context.Context, req *api.ListWorkspaceTemplatesRequest) (*api.ListWorkspaceTemplatesResponse, error) {
	client := ctx.Value("kubeClient").(*v1.Client)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "create", "argoproj.io", "workflowtemplates", "")
	if err != nil || !allowed {
		return nil, err
	}

	paginator := pagination.NewRequest(req.Page, req.PageSize)
	workspaceTemplates, err := client.ListWorkspaceTemplates(req.Namespace, &paginator)
	if err != nil {
		return nil, err
	}

	apiWorkspaceTemplates := []*api.WorkspaceTemplate{}
	for _, wtv := range workspaceTemplates {
		apiWorkspaceTemplates = append(apiWorkspaceTemplates, apiWorkspaceTemplate(wtv))
	}

	count, err := client.CountWorkspaceTemplates(req.Namespace)
	if err != nil {
		return nil, err
	}

	return &api.ListWorkspaceTemplatesResponse{
		Count:              int32(len(apiWorkspaceTemplates)),
		WorkspaceTemplates: apiWorkspaceTemplates,
		Page:               int32(paginator.Page),
		Pages:              paginator.CalculatePages(count),
		TotalCount:         int32(count),
	}, nil
}

func (s *WorkspaceTemplateServer) ListWorkspaceTemplateVersions(ctx context.Context, req *api.ListWorkspaceTemplateVersionsRequest) (*api.ListWorkflowTemplateVersionsResponse, error) {
	client := ctx.Value("kubeClient").(*v1.Client)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "list", "argoproj.io", "workflowtemplates", "")
	if err != nil || !allowed {
		return nil, err
	}

	workflowTemplateVersions, err := client.ListWorkspaceWorkflowTemplateVersions(req.Namespace, req.Name)
	if err != nil {
		return nil, err
	}

	var workflowTemplates []*api.WorkflowTemplate
	for _, wtv := range workflowTemplateVersions {
		workflowTemplates = append(workflowTemplates, apiWorkflowTemplate(wtv))
	}

	return &api.ListWorkflowTemplateVersionsResponse{
		Count:             int32(len(workflowTemplateVersions)),
		WorkflowTemplates: workflowTemplates,
	}, nil
}
