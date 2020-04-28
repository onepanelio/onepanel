package server

import (
	"context"
	"github.com/onepanelio/core/pkg/util"
	"github.com/onepanelio/core/server/converter"
	"google.golang.org/grpc/codes"
	"math"
	"sort"
	"strings"
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

func GenApiWorkflowExecution(wf *v1.WorkflowExecution) (workflow *api.WorkflowExecution) {
	return apiWorkflowExecution(wf)
}

func apiWorkflowExecution(wf *v1.WorkflowExecution) (workflow *api.WorkflowExecution) {
	workflow = &api.WorkflowExecution{
		CreatedAt: wf.CreatedAt.Format(time.RFC3339),
		Name:      wf.Name,
		Uid:       wf.UID,
		Phase:     string(wf.Phase),
		Manifest:  wf.Manifest,
	}

	if !wf.StartedAt.IsZero() {
		workflow.StartedAt = wf.StartedAt.Format(time.RFC3339)
	}
	if !wf.FinishedAt.IsZero() {
		workflow.FinishedAt = wf.FinishedAt.Format(time.RFC3339)
	}

	if wf.WorkflowTemplate != nil {
		workflow.WorkflowTemplate = apiWorkflowTemplate(wf.WorkflowTemplate)
	}

	return
}

func (s *WorkflowServer) CreateWorkflowExecution(ctx context.Context, req *api.CreateWorkflowExecutionRequest) (*api.WorkflowExecution, error) {
	client := ctx.Value("kubeClient").(*v1.Client)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "create", "argoproj.io", "workflows", "")
	if err != nil || !allowed {
		return nil, err
	}

	workflow := &v1.WorkflowExecution{
		Labels: converter.APIKeyValueToLabel(req.WorkflowExecution.Labels),
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
		return nil, err
	}

	return apiWorkflowExecution(wf), nil
}

func (s *WorkflowServer) AddWorkflowExecutionStatistics(ctx context.Context, req *api.AddWorkflowExecutionStatisticRequest) (*empty.Empty, error) {
	client := ctx.Value("kubeClient").(*v1.Client)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "create", "argoproj.io", "workflows", req.Name)
	if err != nil || !allowed {
		return &empty.Empty{}, err
	}

	workflowOutcomeIsSuccess := false
	if req.Statistics.WorkflowStatus == "Succeeded" {
		workflowOutcomeIsSuccess = true
	}

	err = client.FinishWorkflowExecutionStatisticViaExitHandler(req.Namespace, req.Name,
		req.Statistics.WorkflowTemplateId, workflowOutcomeIsSuccess)
	if err != nil {
		return &empty.Empty{}, err
	}
	return &empty.Empty{}, nil
}

func (s *WorkflowServer) CronStartWorkflowExecutionStatistic(ctx context.Context, req *api.CronStartWorkflowExecutionStatisticRequest) (*empty.Empty, error) {
	client := ctx.Value("kubeClient").(*v1.Client)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "get", "argoproj.io", "workflows", req.Name)
	if err != nil || !allowed {
		return &empty.Empty{}, err
	}

	err = client.CronStartWorkflowExecutionStatisticInsert(req.Namespace, req.Name, req.WorkflowTemplateId)
	if err != nil {
		return &empty.Empty{}, err
	}

	return &empty.Empty{}, nil
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
	for {
		le = <-watcher
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
	if len(apiMetrics) == 0 {
		return nil, util.NewUserError(codes.NotFound, "Metrics were not found.")
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

	files, err := client.ListFiles(req.Namespace, req.Path)
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

	sort.Slice(apiFiles, func(i, j int) bool {
		fileI := apiFiles[i]
		fileJ := apiFiles[j]

		if fileI.Directory && !fileJ.Directory {
			return true
		}

		return strings.Compare(fileI.Path, fileJ.Path) < 0
	})

	parentPath := v1.FilePathToParentPath(req.Path)

	return &api.ListFilesResponse{
		Files:      apiFiles,
		ParentPath: parentPath,
	}, nil
}

func (s *WorkflowServer) GetWorkflowExecutionLabels(ctx context.Context, req *api.GetLabelsRequest) (*api.GetLabelsResponse, error) {
	client := ctx.Value("kubeClient").(*v1.Client)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "get", "argoproj.io", "workflows", "")
	if err != nil || !allowed {
		return nil, err
	}

	labels, err := client.GetWorkflowExecutionLabels(req.Namespace, req.Name, "tags.onepanel.io/")
	if err != nil {
		return nil, err
	}

	resp := &api.GetLabelsResponse{
		Labels: mapToKeyValue(labels),
	}

	return resp, nil
}

// Adds any labels that are not yet associated to the workflow execution.
// If the label already exists, overwrites it.
func (s *WorkflowServer) AddWorkflowExecutionLabels(ctx context.Context, req *api.AddLabelsRequest) (*api.GetLabelsResponse, error) {
	client := ctx.Value("kubeClient").(*v1.Client)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "update", "argoproj.io", "workflows", "")
	if err != nil || !allowed {
		return nil, err
	}

	keyValues := make(map[string]string)
	for _, item := range req.Labels.Items {
		keyValues[item.Key] = item.Value
	}

	labels, err := client.SetWorkflowExecutionLabels(req.Namespace, req.Name, "tags.onepanel.io/", keyValues, false)
	if err != nil {
		return nil, err
	}

	resp := &api.GetLabelsResponse{
		Labels: mapToKeyValue(labels),
	}

	return resp, nil
}

// Deletes all of the old labels and adds the new ones.
func (s *WorkflowServer) ReplaceWorkflowExecutionLabels(ctx context.Context, req *api.ReplaceLabelsRequest) (*api.GetLabelsResponse, error) {
	client := ctx.Value("kubeClient").(*v1.Client)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "update", "argoproj.io", "workflows", "")
	if err != nil || !allowed {
		return nil, err
	}

	keyValues := make(map[string]string)
	for _, item := range req.Labels.Items {
		keyValues[item.Key] = item.Value
	}

	labels, err := client.SetWorkflowExecutionLabels(req.Namespace, req.Name, "tags.onepanel.io/", keyValues, true)
	if err != nil {
		return nil, err
	}

	resp := &api.GetLabelsResponse{
		Labels: mapToKeyValue(labels),
	}

	return resp, nil
}

func (s *WorkflowServer) DeleteWorkflowExecutionLabel(ctx context.Context, req *api.DeleteLabelRequest) (*api.GetLabelsResponse, error) {
	client := ctx.Value("kubeClient").(*v1.Client)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "delete", "argoproj.io", "workflows", "")
	if err != nil || !allowed {
		return nil, err
	}

	keyToDelete := "tags.onepanel.io/" + req.Key
	labels, err := client.DeleteWorkflowExecutionLabel(req.Namespace, req.Name, keyToDelete)
	if err != nil {
		return nil, err
	}

	keyValues := make(map[string]string)
	for key, val := range labels {
		keyValues[key] = val
	}

	labels, err = client.SetWorkflowExecutionLabels(req.Namespace, req.Name, "", keyValues, true)
	if err != nil {
		return nil, err
	}

	resp := &api.GetLabelsResponse{
		Labels: mapToKeyValue(labels),
	}

	return resp, nil
}
