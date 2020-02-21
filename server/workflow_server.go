package server

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/onepanelio/core/api"
	v1 "github.com/onepanelio/core/pkg"
	"github.com/onepanelio/core/pkg/util/ptr"
	"github.com/onepanelio/core/server/auth"
)

type WorkflowServer struct{}

func NewWorkflowServer() *WorkflowServer {
	return &WorkflowServer{}
}

func apiWorkflow(wf *v1.Workflow) (workflow *api.Workflow) {
	workflow = &api.Workflow{
		CreatedAt:  wf.CreatedAt.Format(time.RFC3339),
		Name:       wf.Name,
		Uid:        wf.UID,
		Phase:      string(wf.Phase),
		StartedAt:  wf.CreatedAt.Format(time.RFC3339),
		FinishedAt: wf.FinishedAt.Format(time.RFC3339),
		Manifest:   wf.Manifest,
	}

	if wf.WorkflowTemplate != nil {
		workflow.WorkflowTemplate = &api.WorkflowTemplate{
			Uid:        wf.WorkflowTemplate.UID,
			CreatedAt:  wf.WorkflowTemplate.CreatedAt.UTC().Format(time.RFC3339),
			Name:       wf.WorkflowTemplate.Name,
			Version:    wf.WorkflowTemplate.Version,
			Manifest:   wf.WorkflowTemplate.Manifest,
			IsLatest:   wf.WorkflowTemplate.IsLatest,
			IsArchived: wf.WorkflowTemplate.IsArchived,
		}
	}

	return
}

func apiWorkflowTemplate(wft *v1.WorkflowTemplate) *api.WorkflowTemplate {
	return &api.WorkflowTemplate{
		Uid:        wft.UID,
		CreatedAt:  wft.CreatedAt.UTC().Format(time.RFC3339),
		Name:       wft.Name,
		Version:    wft.Version,
		Manifest:   wft.Manifest,
		IsLatest:   wft.IsLatest,
		IsArchived: wft.IsArchived,
	}
}

func (s *WorkflowServer) CreateWorkflow(ctx context.Context, req *api.CreateWorkflowRequest) (*api.Workflow, error) {
	client := ctx.Value("kubeClient").(*v1.Client)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "create", "argoproj.io", "workflows", "")
	if err != nil || !allowed {
		return nil, err
	}

	workflow := &v1.Workflow{
		WorkflowTemplate: &v1.WorkflowTemplate{
			UID:     req.Workflow.WorkflowTemplate.Uid,
			Version: req.Workflow.WorkflowTemplate.Version,
		},
	}
	for _, param := range req.Workflow.Parameters {
		workflow.Parameters = append(workflow.Parameters, v1.WorkflowParameter{
			Name:  param.Name,
			Value: ptr.String(param.Value),
		})
	}

	wf, err := client.CreateWorkflow(req.Namespace, workflow)
	if err != nil {
		if errors.As(err, &userError) {
			return nil, userError.GRPCError()
		}
	}

	return apiWorkflow(wf), nil
}

func (s *WorkflowServer) GetWorkflow(ctx context.Context, req *api.GetWorkflowRequest) (*api.Workflow, error) {
	client := ctx.Value("kubeClient").(*v1.Client)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "get", "argoproj.io", "workflows", req.Name)
	if err != nil || !allowed {
		return nil, err
	}

	wf, err := client.GetWorkflow(req.Namespace, req.Name)
	if errors.As(err, &userError) {
		return nil, userError.GRPCError()
	}

	return apiWorkflow(wf), nil
}

func (s *WorkflowServer) WatchWorkflow(req *api.WatchWorkflowRequest, stream api.WorkflowService_WatchWorkflowServer) error {
	client := stream.Context().Value("kubeClient").(*v1.Client)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "get", "argoproj.io", "workflows", req.Name)
	if err != nil || !allowed {
		return err
	}

	watcher, err := client.WatchWorkflow(req.Namespace, req.Name)
	if errors.As(err, &userError) {
		return userError.GRPCError()
	}

	wf := &v1.Workflow{}
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case wf = <-watcher:
		case <-ticker.C:
		}

		if wf == nil {
			break
		}
		if err := stream.Send(apiWorkflow(wf)); err != nil {
			return err
		}
	}

	return nil
}

func (s *WorkflowServer) GetWorkflowLogs(req *api.GetWorkflowLogsRequest, stream api.WorkflowService_GetWorkflowLogsServer) error {
	client := stream.Context().Value("kubeClient").(*v1.Client)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "get", "argoproj.io", "workflows", req.Name)
	if err != nil || !allowed {
		return err
	}

	watcher, err := client.GetWorkflowLogs(req.Namespace, req.Name, req.PodName, req.ContainerName)
	if errors.As(err, &userError) {
		return userError.GRPCError()
	}

	le := &v1.LogEntry{}
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case le = <-watcher:
		case <-ticker.C:
		}

		if le == nil {
			break
		}
		if err := stream.Send(&api.LogEntry{
			Timestamp: le.Timestamp.String(),
			Content:   le.Content,
		}); err != nil {
			return err
		}
	}

	return nil
}

func (s *WorkflowServer) GetWorkflowMetrics(ctx context.Context, req *api.GetWorkflowMetricsRequest) (*api.GetWorkflowMetricsResponse, error) {
	client := ctx.Value("kubeClient").(*v1.Client)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "get", "argoproj.io", "workflows", req.Name)
	if err != nil || !allowed {
		return nil, err
	}

	metrics, err := client.GetWorkflowMetrics(req.Namespace, req.Name, req.PodName)
	if errors.As(err, &userError) {
		return nil, userError.GRPCError()
	}

	var apiMetrics []*api.Metric
	for _, m := range metrics {
		apiMetrics = append(apiMetrics, &api.Metric{
			Name:   m.Name,
			Value:  m.Value,
			Format: m.Format,
		})
	}

	return &api.GetWorkflowMetricsResponse{Metrics: apiMetrics}, nil
}

func (s *WorkflowServer) ListWorkflows(ctx context.Context, req *api.ListWorkflowsRequest) (*api.ListWorkflowsResponse, error) {
	client := ctx.Value("kubeClient").(*v1.Client)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "list", "argoproj.io", "workflows", "")
	if err != nil || !allowed {
		return nil, err
	}

	if req.PageSize <= 0 {
		req.PageSize = 15
	}

	workflows, err := client.ListWorkflows(req.Namespace, req.WorkflowTemplateUid, req.WorkflowTemplateVersion)
	if errors.As(err, &userError) {
		return nil, userError.GRPCError()
	}

	var apiWorkflows []*api.Workflow
	for _, wf := range workflows {
		apiWorkflows = append(apiWorkflows, apiWorkflow(wf))
	}

	pages := int32(math.Ceil(float64(len(apiWorkflows)) / float64(req.PageSize)))
	if req.Page > pages {
		req.Page = pages
	}

	if req.Page <= 0 {
		req.Page = 1
	}

	start := (req.Page - 1) * req.PageSize
	end := start + req.PageSize
	if end >= int32(len(apiWorkflows)) {
		end = int32(len(apiWorkflows))
	}

	return &api.ListWorkflowsResponse{
		Count:      end - start,
		Workflows:  apiWorkflows[start:end],
		Page:       req.Page,
		Pages:      pages,
		TotalCount: int32(len(apiWorkflows)),
	}, nil
}

func (s *WorkflowServer) ResubmitWorkflow(ctx context.Context, req *api.ResubmitWorkflowRequest) (*api.Workflow, error) {
	client := ctx.Value("kubeClient").(*v1.Client)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "create", "argoproj.io", "workflows", req.Name)
	if err != nil || !allowed {
		return nil, err
	}

	wf, err := client.ResubmitWorkflow(req.Namespace, req.Name)
	if errors.As(err, &userError) {
		return nil, userError.GRPCError()
	}

	return apiWorkflow(wf), nil
}

func (s *WorkflowServer) TerminateWorkflow(ctx context.Context, req *api.TerminateWorkflowRequest) (*empty.Empty, error) {
	client := ctx.Value("kubeClient").(*v1.Client)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "update", "argoproj.io", "workflows", "")
	if err != nil || !allowed {
		return nil, err
	}

	err = client.TerminateWorkflow(req.Namespace, req.Name)
	if errors.As(err, &userError) {
		return nil, userError.GRPCError()
	}

	return &empty.Empty{}, nil
}

func (s *WorkflowServer) CreateWorkflowTemplate(ctx context.Context, req *api.CreateWorkflowTemplateRequest) (*api.WorkflowTemplate, error) {
	client := ctx.Value("kubeClient").(*v1.Client)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "create", "argoproj.io", "workflows", "")
	if err != nil || !allowed {
		return nil, err
	}

	workflowTemplate := &v1.WorkflowTemplate{
		Name:     req.WorkflowTemplate.Name,
		Manifest: req.WorkflowTemplate.Manifest,
	}
	workflowTemplate, err = client.CreateWorkflowTemplate(req.Namespace, workflowTemplate)
	if errors.As(err, &userError) {
		return nil, userError.GRPCError()
	}
	req.WorkflowTemplate.Uid = workflowTemplate.UID
	req.WorkflowTemplate.Version = workflowTemplate.Version

	return req.WorkflowTemplate, nil
}

func (s *WorkflowServer) CreateWorkflowTemplateVersion(ctx context.Context, req *api.CreateWorkflowTemplateRequest) (*api.WorkflowTemplate, error) {
	client := ctx.Value("kubeClient").(*v1.Client)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "create", "argoproj.io", "workflows", req.WorkflowTemplate.Name)
	if err != nil || !allowed {
		return nil, err
	}

	workflowTemplate := &v1.WorkflowTemplate{
		UID:      req.WorkflowTemplate.Uid,
		Name:     req.WorkflowTemplate.Name,
		Manifest: req.WorkflowTemplate.Manifest,
	}

	workflowTemplate, err = client.CreateWorkflowTemplateVersion(req.Namespace, workflowTemplate)
	if errors.As(err, &userError) {
		return nil, userError.GRPCError()
	}
	req.WorkflowTemplate.Uid = workflowTemplate.UID
	req.WorkflowTemplate.Name = workflowTemplate.Name
	req.WorkflowTemplate.Version = workflowTemplate.Version

	return req.WorkflowTemplate, nil
}

func (s *WorkflowServer) UpdateWorkflowTemplateVersion(ctx context.Context, req *api.UpdateWorkflowTemplateVersionRequest) (*api.WorkflowTemplate, error) {
	client := ctx.Value("kubeClient").(*v1.Client)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "update", "argoproj.io", "workflows", req.WorkflowTemplate.Name)
	if err != nil || !allowed {
		return nil, err
	}

	workflowTemplate := &v1.WorkflowTemplate{
		UID:      req.WorkflowTemplate.Uid,
		Name:     req.WorkflowTemplate.Name,
		Manifest: req.WorkflowTemplate.Manifest,
		Version:  req.WorkflowTemplate.Version,
	}
	workflowTemplate, err = client.UpdateWorkflowTemplateVersion(req.Namespace, workflowTemplate)
	if errors.As(err, &userError) {
		return nil, userError.GRPCError()
	}
	req.WorkflowTemplate.Uid = workflowTemplate.UID
	req.WorkflowTemplate.Name = workflowTemplate.Name
	req.WorkflowTemplate.Version = workflowTemplate.Version

	return req.WorkflowTemplate, nil
}

func (s *WorkflowServer) GetWorkflowTemplate(ctx context.Context, req *api.GetWorkflowTemplateRequest) (*api.WorkflowTemplate, error) {
	client := ctx.Value("kubeClient").(*v1.Client)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "get", "argoproj.io", "workflows", "")
	if err != nil || !allowed {
		return nil, err
	}

	workflowTemplate, err := client.GetWorkflowTemplate(req.Namespace, req.Uid, req.Version)
	if errors.As(err, &userError) {
		return nil, userError.GRPCError()
	}

	return apiWorkflowTemplate(workflowTemplate), nil
}

func (s *WorkflowServer) ListWorkflowTemplateVersions(ctx context.Context, req *api.ListWorkflowTemplateVersionsRequest) (*api.ListWorkflowTemplateVersionsResponse, error) {
	client := ctx.Value("kubeClient").(*v1.Client)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "list", "argoproj.io", "workflows", "")
	if err != nil || !allowed {
		return nil, err
	}

	workflowTemplateVersions, err := client.ListWorkflowTemplateVersions(req.Namespace, req.Uid)
	if errors.As(err, &userError) {
		return nil, userError.GRPCError()
	}

	workflowTemplates := []*api.WorkflowTemplate{}
	for _, wtv := range workflowTemplateVersions {
		workflowTemplates = append(workflowTemplates, apiWorkflowTemplate(wtv))
	}

	return &api.ListWorkflowTemplateVersionsResponse{
		Count:             int32(len(workflowTemplateVersions)),
		WorkflowTemplates: workflowTemplates,
	}, nil
}

func (s *WorkflowServer) ListWorkflowTemplates(ctx context.Context, req *api.ListWorkflowTemplatesRequest) (*api.ListWorkflowTemplatesResponse, error) {
	client := ctx.Value("kubeClient").(*v1.Client)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "list", "argoproj.io", "workflows", "")
	if err != nil || !allowed {
		return nil, err
	}

	workflowTemplates, err := client.ListWorkflowTemplates(req.Namespace)
	if errors.As(err, &userError) {
		return nil, userError.GRPCError()
	}

	apiWorkflowTemplates := []*api.WorkflowTemplate{}
	for _, wtv := range workflowTemplates {
		apiWorkflowTemplates = append(apiWorkflowTemplates, apiWorkflowTemplate(wtv))
	}

	return &api.ListWorkflowTemplatesResponse{
		Count:             int32(len(apiWorkflowTemplates)),
		WorkflowTemplates: apiWorkflowTemplates,
	}, nil
}

func (s *WorkflowServer) ArchiveWorkflowTemplate(ctx context.Context, req *api.ArchiveWorkflowTemplateRequest) (*api.ArchiveWorkflowTemplateResponse, error) {
	client := ctx.Value("kubeClient").(*v1.Client)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "delete", "argoproj.io", "workflows", "")
	if err != nil || !allowed {
		return nil, err
	}

	archived, err := client.ArchiveWorkflowTemplate(req.Namespace, req.Uid)
	if errors.As(err, &userError) {
		return nil, userError.GRPCError()
	}

	return &api.ArchiveWorkflowTemplateResponse{
		WorkflowTemplate: &api.WorkflowTemplate{
			IsArchived: archived,
		},
	}, nil
}
