package server

import (
	"context"
	"errors"
	"github.com/onepanelio/core/pkg/util"
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
	res := &api.WorkflowTemplate{
		Uid:        wft.UID,
		CreatedAt:  wft.CreatedAt.UTC().Format(time.RFC3339),
		Name:       wft.Name,
		Version:    wft.Version,
		Manifest:   wft.Manifest,
		IsLatest:   wft.IsLatest,
		IsArchived: wft.IsArchived,
	}

	if wft.WorkflowExecutionStatisticReport != nil {
		res.Stats = &api.WorkflowExecutionStatisticReport{
			Total:        wft.WorkflowExecutionStatisticReport.Total,
			LastExecuted: wft.WorkflowExecutionStatisticReport.LastExecuted.String(),
			Running:      wft.WorkflowExecutionStatisticReport.Running,
			Completed:    wft.WorkflowExecutionStatisticReport.Completed,
			Failed:       wft.WorkflowExecutionStatisticReport.Failed,
		}
	}

	return res
}

func mapToKeyValue(input map[string]string) []*api.KeyValue {
	var result []*api.KeyValue
	for key, value := range input {
		keyValue := &api.KeyValue{
			Key:   key,
			Value: value,
		}

		result = append(result, keyValue)
	}

	return result
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
		return nil, err
	}

	return apiWorkflowExecution(wf), nil
}

func (s *WorkflowServer) AddWorkflowExecutionStatistics(ctx context.Context, request *api.AddWorkflowExecutionStatisticRequest) (*empty.Empty, error) {
	client := ctx.Value("kubeClient").(*v1.Client)
	workflowOutcomeIsSuccess := false
	if request.Statistics.WorkflowStatus == "Success" {
		workflowOutcomeIsSuccess = true
	}

	/*
	 The format from Argo needs to be parsed.
	 It's not RFC3339
	*/
	layout := "2006-01-02 15:04:05 -0700 MST"
	createdAt, err := time.Parse(layout, request.Statistics.CreatedAt)
	if err != nil {
		return &empty.Empty{}, err
	}
	err = client.AddWorkflowExecutionStatistic(request.Namespace, request.Name,
		request.Statistics.WorkflowTemplateId, createdAt, workflowOutcomeIsSuccess)
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

func (s *WorkflowServer) CloneWorkflowTemplate(ctx context.Context, req *api.CloneWorkflowTemplateRequest) (*api.WorkflowTemplate, error) {
	client := ctx.Value("kubeClient").(*v1.Client)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "get", "argoproj.io", "workflows", "")
	if err != nil || !allowed {
		return nil, err
	}

	allowed, err = auth.IsAuthorized(client, req.Namespace, "create", "argoproj.io", "workflows", "")
	if err != nil || !allowed {
		return nil, err
	}

	//Verify the template exists
	workflowTemplate, err := client.GetWorkflowTemplate(req.Namespace, req.Uid, req.Version)
	if err != nil {
		return nil, err
	}

	//Verify the cloned template name doesn't exist already
	workflowTemplateByName, err := client.GetWorkflowTemplateByName(req.Namespace, req.Name, req.Version)
	if err != nil {
		if !strings.Contains(err.Error(), "not found") {
			return nil, err
		}
	}
	if workflowTemplateByName != nil {
		return nil, errors.New("Cannot clone, WorkflowTemplate name already taken.")
	}

	workflowTemplateClone := &v1.WorkflowTemplate{
		Name:     req.Name,
		Manifest: workflowTemplate.Manifest,
		IsLatest: true,
	}
	workflowTemplateCloned, err := client.CreateWorkflowTemplate(req.Namespace, workflowTemplateClone)
	if err != nil {
		return nil, err
	}

	return apiWorkflowTemplate(workflowTemplateCloned), nil
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
	allowed, err := auth.IsAuthorized(client, req.Namespace, "create", "argoproj.io", "workflows", "")
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
	allowed, err := auth.IsAuthorized(client, req.Namespace, "create", "argoproj.io", "workflows", "")
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
	allowed, err := auth.IsAuthorized(client, req.Namespace, "create", "argoproj.io", "workflows", "")
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

func (s *WorkflowServer) GetWorkflowTemplateLabels(ctx context.Context, req *api.GetLabelsRequest) (*api.GetLabelsResponse, error) {
	client := ctx.Value("kubeClient").(*v1.Client)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "create", "argoproj.io", "workflows", "")
	if err != nil || !allowed {
		return nil, err
	}

	labels, err := client.GetWorkflowTemplateLabels(req.Namespace, req.Name, "tags.onepanel.io/")
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
func (s *WorkflowServer) AddWorkflowTemplateLabels(ctx context.Context, req *api.AddLabelsRequest) (*api.GetLabelsResponse, error) {
	client := ctx.Value("kubeClient").(*v1.Client)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "create", "argoproj.io", "workflows", "")
	if err != nil || !allowed {
		return nil, err
	}

	keyValues := make(map[string]string)
	for _, item := range req.Labels.Items {
		keyValues[item.Key] = item.Value
	}

	labels, err := client.SetWorkflowTemplateLabels(req.Namespace, req.Name, "tags.onepanel.io/", keyValues, false)
	if err != nil {
		return nil, err
	}

	resp := &api.GetLabelsResponse{
		Labels: mapToKeyValue(labels),
	}

	return resp, nil
}

// Deletes all of the old labels and adds the new ones.
func (s *WorkflowServer) ReplaceWorkflowTemplateLabels(ctx context.Context, req *api.ReplaceLabelsRequest) (*api.GetLabelsResponse, error) {
	client := ctx.Value("kubeClient").(*v1.Client)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "create", "argoproj.io", "workflows", "")
	if err != nil || !allowed {
		return nil, err
	}

	keyValues := make(map[string]string)
	for _, item := range req.Labels.Items {
		keyValues[item.Key] = item.Value
	}

	labels, err := client.SetWorkflowTemplateLabels(req.Namespace, req.Name, "tags.onepanel.io/", keyValues, true)
	if err != nil {
		return nil, err
	}

	resp := &api.GetLabelsResponse{
		Labels: mapToKeyValue(labels),
	}

	return resp, nil
}

func (s *WorkflowServer) DeleteWorkflowTemplateLabel(ctx context.Context, req *api.DeleteLabelRequest) (*api.GetLabelsResponse, error) {
	client := ctx.Value("kubeClient").(*v1.Client)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "delete", "argoproj.io", "workflows", "")
	if err != nil || !allowed {
		return nil, err
	}

	keyToDelete := "tags.onepanel.io/" + req.Key
	labels, err := client.DeleteWorkflowTemplateLabel(req.Namespace, req.Name, keyToDelete)
	if err != nil {
		return nil, err
	}

	resp := &api.GetLabelsResponse{
		Labels: mapToKeyValue(labels),
	}

	return resp, nil
}
