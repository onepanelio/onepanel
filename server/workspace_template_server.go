package server

import (
	"context"
	api "github.com/onepanelio/core/api/gen"
	v1 "github.com/onepanelio/core/pkg"
	"github.com/onepanelio/core/pkg/util"
	"github.com/onepanelio/core/pkg/util/request"
	"github.com/onepanelio/core/pkg/util/request/pagination"
	requestSort "github.com/onepanelio/core/pkg/util/request/sort"
	"github.com/onepanelio/core/server/auth"
	"github.com/onepanelio/core/server/converter"
	"google.golang.org/grpc/codes"
	"time"
)

// WorkspaceTemplateServer is an implementation of the grpc WorkspaceTemplateServer
type WorkspaceTemplateServer struct {
	api.UnimplementedWorkspaceTemplateServiceServer
}

// NewWorkspaceTemplateServer creates a new WorkspaceTemplateServer
func NewWorkspaceTemplateServer() *WorkspaceTemplateServer {
	return &WorkspaceTemplateServer{}
}

func apiWorkspaceTemplate(wt *v1.WorkspaceTemplate) *api.WorkspaceTemplate {
	res := &api.WorkspaceTemplate{
		Uid:         wt.UID,
		Name:        wt.Name,
		Description: wt.Description,
		Version:     wt.Version,
		Manifest:    wt.Manifest,
		IsLatest:    wt.IsLatest,
		CreatedAt:   wt.CreatedAt.UTC().Format(time.RFC3339),
		Labels:      converter.MappingToKeyValue(wt.Labels),
	}

	if wt.WorkflowTemplate != nil {
		res.WorkflowTemplate = apiWorkflowTemplate(wt.WorkflowTemplate)
	}

	return res
}

func (s WorkspaceTemplateServer) GenerateWorkspaceTemplateWorkflowTemplate(ctx context.Context, req *api.GenerateWorkspaceTemplateWorkflowTemplateRequest) (*api.WorkflowTemplate, error) {
	client := getClient(ctx)
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
		Namespace: req.Namespace,
		Manifest:  req.WorkspaceTemplate.Manifest,
	}
	workflowTemplate, err := client.GenerateWorkspaceTemplateWorkflowTemplate(workspaceTemplate)

	if workflowTemplate == nil {
		return &api.WorkflowTemplate{
			Manifest: "",
		}, err
	}

	return apiWorkflowTemplate(workflowTemplate), nil
}

func (s *WorkspaceTemplateServer) CreateWorkspaceTemplate(ctx context.Context, req *api.CreateWorkspaceTemplateRequest) (*api.WorkspaceTemplate, error) {
	client := getClient(ctx)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "create", "argoproj.io", "workflowtemplates", "")
	if err != nil || !allowed {
		return nil, err
	}

	if IsNameReservedForSystem(req.WorkspaceTemplate.Name) {
		return nil, v1.NameReservedForSystemError()
	}

	workspaceTemplate := &v1.WorkspaceTemplate{
		Namespace:   req.Namespace,
		Name:        req.WorkspaceTemplate.Name,
		Manifest:    req.WorkspaceTemplate.Manifest,
		Description: req.WorkspaceTemplate.Description,
		Labels:      converter.APIKeyValueToLabel(req.WorkspaceTemplate.Labels),
	}
	workspaceTemplate, err = client.CreateWorkspaceTemplate(req.Namespace, workspaceTemplate)
	if err != nil {
		return nil, err
	}

	req.WorkspaceTemplate = apiWorkspaceTemplate(workspaceTemplate)

	return req.WorkspaceTemplate, nil
}

func (s *WorkspaceTemplateServer) UpdateWorkspaceTemplate(ctx context.Context, req *api.UpdateWorkspaceTemplateRequest) (*api.WorkspaceTemplate, error) {
	client := getClient(ctx)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "update", "argoproj.io", "workflowtemplates", req.Uid)
	if err != nil || !allowed {
		return nil, err
	}

	workspaceTemplate := &v1.WorkspaceTemplate{
		Namespace:   req.Namespace,
		UID:         req.Uid,
		Manifest:    req.WorkspaceTemplate.Manifest,
		Description: req.WorkspaceTemplate.Description,
		Labels:      converter.APIKeyValueToLabel(req.WorkspaceTemplate.Labels),
	}
	workspaceTemplate, err = client.UpdateWorkspaceTemplate(req.Namespace, workspaceTemplate)
	if err != nil {
		return nil, err
	}

	req.WorkspaceTemplate = apiWorkspaceTemplate(workspaceTemplate)

	return req.WorkspaceTemplate, nil
}

func (s *WorkspaceTemplateServer) GetWorkspaceTemplate(ctx context.Context, req *api.GetWorkspaceTemplateRequest) (*api.WorkspaceTemplate, error) {
	client := getClient(ctx)
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
	client := getClient(ctx)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "create", "argoproj.io", "workflowtemplates", "")
	if err != nil || !allowed {
		return nil, err
	}

	labelFilter, err := v1.LabelsFromString(req.Labels)
	if err != nil {
		return nil, err
	}
	reqSort, err := requestSort.New(req.Order)
	if err != nil {
		return nil, err
	}

	resourceRequest := &request.Request{
		Pagination: pagination.New(req.Page, req.PageSize),
		Filter: v1.WorkspaceTemplateFilter{
			Labels: labelFilter,
			UID:    req.Uid,
		},
		Sort: reqSort,
	}

	workspaceTemplates, err := client.ListWorkspaceTemplates(req.Namespace, resourceRequest)
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

	paginator := resourceRequest.Pagination
	return &api.ListWorkspaceTemplatesResponse{
		Count:              int32(len(apiWorkspaceTemplates)),
		WorkspaceTemplates: apiWorkspaceTemplates,
		Page:               int32(paginator.Page),
		Pages:              paginator.CalculatePages(count),
		TotalCount:         int32(count),
	}, nil
}

func (s *WorkspaceTemplateServer) ListWorkspaceTemplateVersions(ctx context.Context, req *api.ListWorkspaceTemplateVersionsRequest) (*api.ListWorkspaceTemplateVersionsResponse, error) {
	client := getClient(ctx)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "list", "argoproj.io", "workflowtemplates", "")
	if err != nil || !allowed {
		return nil, err
	}

	workspaceTemplateVersions, err := client.ListWorkspaceTemplateVersions(req.Namespace, req.Uid)
	if err != nil {
		return nil, err
	}

	var workspaceTemplates []*api.WorkspaceTemplate
	for _, wtv := range workspaceTemplateVersions {
		workspaceTemplates = append(workspaceTemplates, apiWorkspaceTemplate(wtv))
	}

	return &api.ListWorkspaceTemplateVersionsResponse{
		Count:              int32(len(workspaceTemplateVersions)),
		WorkspaceTemplates: workspaceTemplates,
	}, nil
}

func (s *WorkspaceTemplateServer) ArchiveWorkspaceTemplate(ctx context.Context, req *api.ArchiveWorkspaceTemplateRequest) (*api.WorkspaceTemplate, error) {
	client := getClient(ctx)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "delete", "argoproj.io", "workflowtemplates", "")
	if err != nil || !allowed {
		return nil, err
	}

	hasRunning, err := client.WorkspaceTemplateHasRunningWorkspaces(req.Namespace, req.Uid)
	if err != nil {
		return nil, util.NewUserError(codes.Unknown, "Unable to get check running workspaces")
	}
	if hasRunning {
		return nil, util.NewUserError(codes.FailedPrecondition, "Unable to archive workspace template. There are running workspaces that use it.")
	}

	archived, err := client.ArchiveWorkspaceTemplate(req.Namespace, req.Uid)
	if err != nil {
		return nil, err
	}

	return &api.WorkspaceTemplate{
		IsArchived: archived,
	}, nil
}
