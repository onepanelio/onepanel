package server

import (
	"context"
	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/onepanelio/core/pkg/util"
	"github.com/onepanelio/core/pkg/util/request"
	"github.com/onepanelio/core/pkg/util/request/pagination"
	"github.com/onepanelio/core/pkg/util/router"
	"github.com/onepanelio/core/server/converter"
	"google.golang.org/grpc/codes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sort"
	"strings"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/onepanelio/core/api"
	v1 "github.com/onepanelio/core/pkg"
	"github.com/onepanelio/core/pkg/util/ptr"
	"github.com/onepanelio/core/server/auth"

	requestSort "github.com/onepanelio/core/pkg/util/request/sort"
)

type WorkflowServer struct{}

func NewWorkflowServer() *WorkflowServer {
	return &WorkflowServer{}
}

// apiWorkflowExecution converts a package workflow execution to the api version
// router is optional
func apiWorkflowExecution(wf *v1.WorkflowExecution, router router.Web) (workflow *api.WorkflowExecution) {
	workflow = &api.WorkflowExecution{
		CreatedAt: wf.CreatedAt.Format(time.RFC3339),
		Uid:       wf.UID,
		Name:      wf.Name,
		Phase:     string(wf.Phase),
		Manifest:  wf.Manifest,
		Labels:    converter.MappingToKeyValue(wf.Labels),
	}

	if wf.StartedAt != nil && !wf.StartedAt.IsZero() {
		workflow.StartedAt = wf.StartedAt.Format(time.RFC3339)
	}
	if wf.FinishedAt != nil && !wf.FinishedAt.IsZero() {
		workflow.FinishedAt = wf.FinishedAt.Format(time.RFC3339)
	}
	if wf.WorkflowTemplate != nil {
		workflow.WorkflowTemplate = apiWorkflowTemplate(wf.WorkflowTemplate)
	}
	if wf.ParametersBytes != nil {
		parameters, err := wf.LoadParametersFromBytes()
		if err != nil {
			return nil
		}

		workflow.Parameters = converter.ParametersToAPI(parameters)
	}

	if router != nil {
		workflow.Metadata = &api.WorkflowExecutionMetadata{
			Url: router.WorkflowExecution(wf.Namespace, wf.UID),
		}
	}

	return
}

func (s *WorkflowServer) CreateWorkflowExecution(ctx context.Context, req *api.CreateWorkflowExecutionRequest) (*api.WorkflowExecution, error) {
	client := getClient(ctx)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "create", "argoproj.io", "workflows", "")
	if err != nil || !allowed {
		return nil, err
	}

	workflow := &v1.WorkflowExecution{
		Labels: converter.APIKeyValueToLabel(req.Body.Labels),
		WorkflowTemplate: &v1.WorkflowTemplate{
			UID:     req.Body.WorkflowTemplateUid,
			Version: req.Body.WorkflowTemplateVersion,
		},
	}
	for _, param := range req.Body.Parameters {
		workflow.Parameters = append(workflow.Parameters, v1.Parameter{
			Name:  param.Name,
			Value: ptr.String(param.Value),
		})
	}

	workflowTemplate, err := client.GetWorkflowTemplate(req.Namespace, req.Body.WorkflowTemplateUid, req.Body.WorkflowTemplateVersion)
	if err != nil {
		return nil, err
	}

	wf, err := client.CreateWorkflowExecution(req.Namespace, workflow, workflowTemplate)
	if err != nil {
		return nil, err
	}
	wf.Namespace = req.Namespace

	webRouter, err := client.GetWebRouter()
	if err != nil {
		return nil, err
	}

	return apiWorkflowExecution(wf, webRouter), nil
}

func (s *WorkflowServer) CloneWorkflowExecution(ctx context.Context, req *api.CloneWorkflowExecutionRequest) (*api.WorkflowExecution, error) {
	client := getClient(ctx)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "create", "argoproj.io", "workflows", "")
	if err != nil || !allowed {
		return nil, err
	}

	wf, err := client.CloneWorkflowExecution(req.Namespace, req.Uid)
	if err != nil {
		return nil, err
	}
	wf.Namespace = req.Namespace

	webRouter, err := client.GetWebRouter()
	if err != nil {
		return nil, err
	}

	return apiWorkflowExecution(wf, webRouter), nil
}

func (s *WorkflowServer) AddWorkflowExecutionStatistics(ctx context.Context, req *api.AddWorkflowExecutionStatisticRequest) (*empty.Empty, error) {
	client := getClient(ctx)
	phase := v1alpha1.NodeFailed
	if req.Statistics.WorkflowStatus == "Succeeded" {
		phase = v1alpha1.NodeSucceeded
	}

	// TODO: This needs to be moved to pkg
	workflow, err := client.ArgoprojV1alpha1().Workflows(req.Namespace).Get(req.Uid, metav1.GetOptions{})
	if err != nil {
		return &empty.Empty{}, err
	}

	err = client.FinishWorkflowExecutionStatisticViaExitHandler(req.Namespace, req.Uid,
		req.Statistics.WorkflowTemplateId, phase, workflow.Status.StartedAt.UTC())

	if err != nil {
		return &empty.Empty{}, err
	}
	return &empty.Empty{}, nil
}

// @todo we should not pass in an id into the request.
// instead pass in the cron workflow uid, we can load the cron workflow from db that way and get
// all required data.
func (s *WorkflowServer) CronStartWorkflowExecutionStatistic(ctx context.Context, req *api.CronStartWorkflowExecutionStatisticRequest) (*empty.Empty, error) {
	client := getClient(ctx)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "get", "argoproj.io", "workflows", req.Uid)
	if err != nil || !allowed {
		return &empty.Empty{}, err
	}

	err = client.CronStartWorkflowExecutionStatisticInsert(req.Namespace, req.Uid, req.Statistics.WorkflowTemplateId)
	if err != nil {
		return &empty.Empty{}, err
	}

	return &empty.Empty{}, nil
}

func (s *WorkflowServer) GetWorkflowExecution(ctx context.Context, req *api.GetWorkflowExecutionRequest) (*api.WorkflowExecution, error) {
	client := getClient(ctx)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "get", "argoproj.io", "workflows", req.Uid)
	if err != nil || !allowed {
		return nil, err
	}

	wf, err := client.GetWorkflowExecution(req.Namespace, req.Uid)
	if err != nil {
		return nil, err
	}
	if wf == nil {
		return nil, util.NewUserError(codes.NotFound, "Workflow not found")
	}

	wf.Namespace = req.Namespace

	webRouter, err := client.GetWebRouter()
	if err != nil {
		return nil, err
	}
	return apiWorkflowExecution(wf, webRouter), nil
}

func (s *WorkflowServer) WatchWorkflowExecution(req *api.WatchWorkflowExecutionRequest, stream api.WorkflowService_WatchWorkflowExecutionServer) error {
	client := getClient(stream.Context())
	allowed, err := auth.IsAuthorized(client, req.Namespace, "get", "argoproj.io", "workflows", req.Uid)
	if err != nil || !allowed {
		return err
	}

	watcher, err := client.WatchWorkflowExecution(req.Namespace, req.Uid)
	if err != nil {
		return err
	}

	webRouter, err := client.GetWebRouter()
	if err != nil {
		return err
	}

	for wf := range watcher {
		if wf == nil {
			break
		}
		wf.Namespace = req.Namespace
		if err := stream.Send(apiWorkflowExecution(wf, webRouter)); err != nil {
			return err
		}
	}

	return nil
}

func (s *WorkflowServer) GetWorkflowExecutionLogs(req *api.GetWorkflowExecutionLogsRequest, stream api.WorkflowService_GetWorkflowExecutionLogsServer) error {
	client := getClient(stream.Context())
	allowed, err := auth.IsAuthorized(client, req.Namespace, "get", "argoproj.io", "workflows", req.Uid)
	if err != nil || !allowed {
		return err
	}

	watcher, err := client.GetWorkflowExecutionLogs(req.Namespace, req.Uid, req.PodName, req.ContainerName)
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
	client := getClient(ctx)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "get", "argoproj.io", "workflows", req.Uid)
	if err != nil || !allowed {
		return nil, err
	}

	metrics, err := client.GetWorkflowExecutionMetrics(req.Namespace, req.Uid, req.PodName)
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

// ListWorkflowExecutions returns a list of workflow executions that are specified by the criteria in the ListWorkflowExecutionsRequest
func (s *WorkflowServer) ListWorkflowExecutions(ctx context.Context, req *api.ListWorkflowExecutionsRequest) (*api.ListWorkflowExecutionsResponse, error) {
	client := getClient(ctx)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "list", "argoproj.io", "workflows", "")
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
		Filter: v1.WorkflowExecutionFilter{
			Labels: labelFilter,
		},
		Sort: reqSort,
	}

	workflows, err := client.ListWorkflowExecutions(req.Namespace, req.WorkflowTemplateUid, req.WorkflowTemplateVersion, resourceRequest)
	if err != nil {
		return nil, err
	}

	webRouter, err := client.GetWebRouter()
	if err != nil {
		return nil, err
	}

	var apiWorkflowExecutions []*api.WorkflowExecution
	for _, wf := range workflows {
		wf.Namespace = req.Namespace
		apiWorkflowExecutions = append(apiWorkflowExecutions, apiWorkflowExecution(wf, webRouter))
	}

	count, err := client.CountWorkflowExecutions(req.Namespace, req.WorkflowTemplateUid, req.WorkflowTemplateVersion, resourceRequest)
	if err != nil {
		return nil, err
	}

	paginator := resourceRequest.Pagination
	return &api.ListWorkflowExecutionsResponse{
		Count:              int32(len(apiWorkflowExecutions)),
		WorkflowExecutions: apiWorkflowExecutions,
		Page:               int32(paginator.Page),
		Pages:              paginator.CalculatePages(count),
		TotalCount:         int32(count),
	}, nil
}

func (s *WorkflowServer) ResubmitWorkflowExecution(ctx context.Context, req *api.ResubmitWorkflowExecutionRequest) (*api.WorkflowExecution, error) {
	client := getClient(ctx)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "create", "argoproj.io", "workflows", req.Uid)
	if err != nil || !allowed {
		return nil, err
	}

	wf, err := client.ResubmitWorkflowExecution(req.Namespace, req.Uid)
	if err != nil {
		return nil, err
	}

	wf.Namespace = req.Namespace
	webRouter, err := client.GetWebRouter()
	if err != nil {
		return nil, err
	}

	return apiWorkflowExecution(wf, webRouter), nil
}

func (s *WorkflowServer) TerminateWorkflowExecution(ctx context.Context, req *api.TerminateWorkflowExecutionRequest) (*empty.Empty, error) {
	client := getClient(ctx)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "update", "argoproj.io", "workflows", "")
	if err != nil || !allowed {
		return nil, err
	}

	err = client.TerminateWorkflowExecution(req.Namespace, req.Uid)
	if err != nil {
		return nil, err
	}

	return &empty.Empty{}, nil
}

func (s *WorkflowServer) GetArtifact(ctx context.Context, req *api.GetArtifactRequest) (*api.ArtifactResponse, error) {
	client := getClient(ctx)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "get", "argoproj.io", "workflows", req.Uid)
	if err != nil || !allowed {
		return nil, err
	}

	data, err := client.GetArtifact(req.Namespace, req.Uid, req.Key)
	if err != nil {
		return nil, err
	}

	return &api.ArtifactResponse{
		Data: data,
	}, nil
}

func (s *WorkflowServer) ListFiles(ctx context.Context, req *api.ListFilesRequest) (*api.ListFilesResponse, error) {
	client := getClient(ctx)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "get", "argoproj.io", "workflows", req.Uid)
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

func (s *WorkflowServer) UpdateWorkflowExecutionStatus(ctx context.Context, req *api.UpdateWorkflowExecutionStatusRequest) (*empty.Empty, error) {
	client := getClient(ctx)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "update", "argoproj.io", "workflows", req.Uid)
	if err != nil || !allowed {
		return &empty.Empty{}, err
	}

	status := &v1.WorkflowExecutionStatus{
		Phase: wfv1.NodePhase(req.Status.Phase),
	}
	err = client.UpdateWorkflowExecutionStatus(req.Namespace, req.Uid, status)

	return &empty.Empty{}, err
}
