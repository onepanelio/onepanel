package server

import (
	"context"
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

func apiWorkflowExecution(wf *v1.WorkflowExecution) (workflow *api.WorkflowExecution) {
	workflow = &api.WorkflowExecution{
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

func (s *WorkflowServer) CreateWorkflowExecution(ctx context.Context, req *api.CreateWorkflowExecutionRequest) (*api.WorkflowExecution, error) {
	client := ctx.Value("kubeClient").(*v1.Client)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "create", "argoproj.io", "workflows", "")
	if err != nil || !allowed {
		return nil, err
	}

	workflow := &v1.WorkflowExecution{
		WorkflowTemplate: &v1.WorkflowTemplate{
			UID:     req.WorkflowExecution.WorkflowTemplate.Uid,
			Version: req.WorkflowExecution.WorkflowTemplate.Version,
		},
	}
	for _, param := range req.WorkflowExecution.Parameters {
		workflow.Parameters = append(workflow.Parameters, v1.WorkflowExecutionParameter{
			Name:  param.Name,
			Value: ptr.String(param.Value),
		})
	}

	wf, err := client.CreateWorkflowExecution(req.Namespace, workflow)
	if err != nil {
		if err != nil {
			return nil, err
		}
	}

	return apiWorkflowExecution(wf), nil
}

func (s *WorkflowServer) GetWorkflowExecution(ctx context.Context, req *api.GetWorkflowExecutionRequest) (*api.WorkflowExecution, error) {
	client := ctx.Value("kubeClient").(*v1.Client)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "get", "argoproj.io", "workflows", req.Name)
	if err != nil || !allowed {
		return nil, err
	}

	wf, err := client.GetWorkflowExecution(req.Namespace, req.Name)
	if err != nil {
		return nil, err
	}

	return apiWorkflowExecution(wf), nil
}

func (s *WorkflowServer) WatchWorkflowExecution(req *api.WatchWorkflowExecutionRequest, stream api.WorkflowService_WatchWorkflowExecutionServer) error {
	client := stream.Context().Value("kubeClient").(*v1.Client)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "get", "argoproj.io", "workflows", req.Name)
	if err != nil || !allowed {
		return err
	}

	watcher, err := client.WatchWorkflowExecution(req.Namespace, req.Name)
	if err != nil {
		return err
	}

	wf := &v1.WorkflowExecution{}
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case wf = <-watcher:
		case <-ticker.C:
		}

		if wf == nil {
			break
		}
		if err := stream.Send(apiWorkflowExecution(wf)); err != nil {
			return err
		}
	}

	return nil
}

func (s *WorkflowServer) GetWorkflowExecutionLogs(req *api.GetWorkflowExecutionLogsRequest, stream api.WorkflowService_GetWorkflowExecutionLogsServer) error {
	client := stream.Context().Value("kubeClient").(*v1.Client)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "get", "argoproj.io", "workflows", req.Name)
	if err != nil || !allowed {
		return err
	}

	watcher, err := client.GetWorkflowExecutionLogs(req.Namespace, req.Name, req.PodName, req.ContainerName)
	if err != nil {
		return err
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

func (s *WorkflowServer) GetWorkflowExecutionMetrics(ctx context.Context, req *api.GetWorkflowExecutionMetricsRequest) (*api.GetWorkflowExecutionMetricsResponse, error) {
	client := ctx.Value("kubeClient").(*v1.Client)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "get", "argoproj.io", "workflows", req.Name)
	if err != nil || !allowed {
		return nil, err
	}

	metrics, err := client.GetWorkflowExecutionMetrics(req.Namespace, req.Name, req.PodName)
	if err != nil {
		return nil, err
	}

	var apiMetrics []*api.Metric
	for _, m := range metrics {
		apiMetrics = append(apiMetrics, &api.Metric{
			Name:   m.Name,
			Value:  m.Value,
			Format: m.Format,
		})
	}

	return &api.GetWorkflowExecutionMetricsResponse{Metrics: apiMetrics}, nil
}

func (s *WorkflowServer) ListWorkflowExecutions(ctx context.Context, req *api.ListWorkflowExecutionsRequest) (*api.ListWorkflowExecutionsResponse, error) {
	client := ctx.Value("kubeClient").(*v1.Client)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "list", "argoproj.io", "workflows", "")
	if err != nil || !allowed {
		return nil, err
	}

	if req.PageSize <= 0 {
		req.PageSize = 15
	}

	workflows, err := client.ListWorkflowExecutions(req.Namespace, req.WorkflowTemplateUid, req.WorkflowTemplateVersion)
	if err != nil {
		return nil, err
	}

	var apiWorkflowExecutions []*api.WorkflowExecution
	for _, wf := range workflows {
		apiWorkflowExecutions = append(apiWorkflowExecutions, apiWorkflowExecution(wf))
	}

	pages := int32(math.Ceil(float64(len(apiWorkflowExecutions)) / float64(req.PageSize)))
	if req.Page > pages {
		req.Page = pages
	}

	if req.Page <= 0 {
		req.Page = 1
	}

	start := (req.Page - 1) * req.PageSize
	end := start + req.PageSize
	if end >= int32(len(apiWorkflowExecutions)) {
		end = int32(len(apiWorkflowExecutions))
	}

	return &api.ListWorkflowExecutionsResponse{
		Count:              end - start,
		WorkflowExecutions: apiWorkflowExecutions[start:end],
		Page:               req.Page,
		Pages:              pages,
		TotalCount:         int32(len(apiWorkflowExecutions)),
	}, nil
}

func (s *WorkflowServer) ResubmitWorkflowExecution(ctx context.Context, req *api.ResubmitWorkflowExecutionRequest) (*api.WorkflowExecution, error) {
	client := ctx.Value("kubeClient").(*v1.Client)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "create", "argoproj.io", "workflows", req.Name)
	if err != nil || !allowed {
		return nil, err
	}

	wf, err := client.ResubmitWorkflowExecution(req.Namespace, req.Name)
	if err != nil {
		return nil, err
	}

	return apiWorkflowExecution(wf), nil
}

func (s *WorkflowServer) TerminateWorkflowExecution(ctx context.Context, req *api.TerminateWorkflowExecutionRequest) (*empty.Empty, error) {
	client := ctx.Value("kubeClient").(*v1.Client)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "update", "argoproj.io", "workflows", "")
	if err != nil || !allowed {
		return nil, err
	}

	err = client.TerminateWorkflowExecution(req.Namespace, req.Name)
	if err != nil {
		return nil, err
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
	if err != nil {
		return nil, err
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
	if err != nil {
		return nil, err
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
	if err != nil {
		return nil, err
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
	if err != nil {
		return nil, err
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
	if err != nil {
		return nil, err
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
	if err != nil {
		return nil, err
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
	if err != nil {
		return nil, err
	}

	return &api.ArchiveWorkflowTemplateResponse{
		WorkflowTemplate: &api.WorkflowTemplate{
			IsArchived: archived,
		},
	}, nil
}

func (s *WorkflowServer) GetArtifact(ctx context.Context, req *api.GetArtifactRequest) (*api.ArtifactResponse, error) {
	client := ctx.Value("kubeClient").(*v1.Client)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "get", "argoproj.io", "workflows", req.Name)
	if err != nil || !allowed {
		return nil, err
	}

	data, err := client.GetArtifact(req.Namespace, req.Name, req.Key)
	if err != nil {
		return nil, err
	}

	return &api.ArtifactResponse{
		Data: data,
	}, nil
}

func (s *WorkflowServer) ListFiles(ctx context.Context, req *api.ListFilesRequest) (*api.ListFilesResponse, error) {
	client := ctx.Value("kubeClient").(*v1.Client)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "get", "argoproj.io", "workflows", req.Name)
	if err != nil || !allowed {
		return nil, err
	}

	files, err := client.ListFiles(req.Namespace, req.Name, req.Path)
	if err != nil {
		return nil, err
	}

	apiFiles := make([]*api.File, len(files))
	for i, file := range files {
		apiFiles[i] = &api.File{
			Path:         file.Path,
			Name:         file.Name,
			Extension:    file.Extension,
			Directory:    file.Directory,
			Size:         file.Size,
			ContentType:  file.ContentType,
			LastModified: file.LastModified.UTC().Format(time.RFC3339),
		}
	}

	parentPath := v1.FilePathToParentPath(req.Path)

	return &api.ListFilesResponse{
		Files:      apiFiles,
		ParentPath: parentPath,
	}, nil
}
