package server

import (
	"context"
	"encoding/json"
	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/onepanelio/core/pkg/util"
	"github.com/onepanelio/core/pkg/util/collection"
	"github.com/onepanelio/core/pkg/util/request"
	"github.com/onepanelio/core/pkg/util/request/pagination"
	"github.com/onepanelio/core/pkg/util/router"
	"github.com/onepanelio/core/server/converter"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	api "github.com/onepanelio/core/api/gen"
	v1 "github.com/onepanelio/core/pkg"
	"github.com/onepanelio/core/pkg/util/ptr"
	"github.com/onepanelio/core/server/auth"

	requestSort "github.com/onepanelio/core/pkg/util/request/sort"
)

// WorkflowServer is an implementation of the grpc WorkflowServer
type WorkflowServer struct {
	api.UnimplementedWorkflowServiceServer
}

// NewWorkflowServer creates a new WorkflowServer
func NewWorkflowServer() *WorkflowServer {
	return &WorkflowServer{}
}

// removedUnusedManifestFields removes any fields not necessary in a Workflow Manfiest.
// this is used to cut down the size for more efficient data transmission
func removedUnusedManifestFields(manifest string) (string, error) {
	if manifest == "" {
		return "", nil
	}

	result := make(map[string]map[string]interface{})
	if err := json.Unmarshal([]byte(manifest), &result); err != nil {
		return "", err
	}

	for key := range result {
		collection.RemoveBlanks(result[key])
	}

	delete(result["metadata"], "managedFields")
	delete(result["status"], "resourcesDuration")

	templatesRaw, ok := result["spec"]["templates"]
	if ok {
		templatesArray := templatesRaw.([]interface{})

		for _, template := range templatesArray {
			templateMap := template.(map[string]interface{})
			delete(templateMap, "metadata")
		}
	}

	nodeStatusRaw, ok := result["status"]["nodes"]
	if ok {
		nodeStatusArray := nodeStatusRaw.(map[string]interface{})

		for _, nodeStatus := range nodeStatusArray {
			nodeStatusMap := nodeStatus.(map[string]interface{})
			delete(nodeStatusMap, "resourcesDuration")
		}
	}

	finalManifestBytes, err := json.Marshal(result)
	if err != nil {
		return "", err
	}

	return string(finalManifestBytes), nil
}

// apiWorkflowExecution converts a package workflow execution to the api version
// router is optional
func apiWorkflowExecution(wf *v1.WorkflowExecution, router router.Web) (workflow *api.WorkflowExecution) {
	manifest, err := removedUnusedManifestFields(wf.Manifest)
	if err != nil {
		log.Printf("error trimming manifest %v", err)
		return nil
	}

	workflow = &api.WorkflowExecution{
		CreatedAt: wf.CreatedAt.Format(time.RFC3339),
		Uid:       wf.UID,
		Name:      wf.Name,
		Phase:     string(wf.Phase),
		Manifest:  manifest,
		Labels:    converter.MappingToKeyValue(wf.Labels),
		Metrics:   converter.MetricsToAPI(wf.Metrics),
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

	err = client.FinishWorkflowExecutionStatisticViaExitHandler(req.Namespace, req.Uid, phase, workflow.Status.StartedAt.UTC())

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
			log.Printf("Stream Send failed: %v\n", err)
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

	le := make([]*v1.LogEntry, 0)
	for {
		le = <-watcher
		if le == nil {
			break
		}

		apiLogEntries := make([]*api.LogEntry, len(le))
		for i, item := range le {
			apiLogEntries[i] = &api.LogEntry{
				Content: item.Content,
			}

			if item.Timestamp.After(time.Time{}) {
				apiLogEntries[i].Timestamp = item.Timestamp.Format(time.RFC3339)
			}
		}

		if err := stream.Send(&api.LogStreamResponse{
			LogEntries: apiLogEntries,
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
			Phase:  req.Phase,
		},
		Sort: reqSort,
	}

	workflows, err := client.ListWorkflowExecutions(req.Namespace, req.WorkflowTemplateUid, req.WorkflowTemplateVersion, req.IncludeSystem, resourceRequest)
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

	count, err := client.CountWorkflowExecutions(req.Namespace, req.WorkflowTemplateUid, req.WorkflowTemplateVersion, req.IncludeSystem, resourceRequest)
	if err != nil {
		return nil, err
	}

	totalCount, err := client.CountWorkflowExecutions(req.Namespace, "", "", req.IncludeSystem, nil)
	if err != nil {
		return nil, err
	}

	paginator := resourceRequest.Pagination
	return &api.ListWorkflowExecutionsResponse{
		Count:               int32(len(apiWorkflowExecutions)),
		WorkflowExecutions:  apiWorkflowExecutions,
		Page:                int32(paginator.Page),
		Pages:               paginator.CalculatePages(count),
		TotalCount:          int32(count),
		TotalAvailableCount: int32(totalCount),
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

// GetWorkflowExecutionStatisticsForNamespace returns statistics on workflow executions for a given namespace
func (s *WorkflowServer) GetWorkflowExecutionStatisticsForNamespace(ctx context.Context, req *api.GetWorkflowExecutionStatisticsForNamespaceRequest) (*api.GetWorkflowExecutionStatisticsForNamespaceResponse, error) {
	client := getClient(ctx)

	allowed, err := auth.IsAuthorized(client, req.Namespace, "list", "argoproj.io", "workflows", "")
	if err != nil || !allowed {
		return nil, err
	}

	report, err := client.GetWorkflowExecutionStatisticsForNamespace(req.Namespace)
	if err != nil {
		return nil, err
	}

	return &api.GetWorkflowExecutionStatisticsForNamespaceResponse{
		Stats: converter.WorkflowExecutionStatisticsReportToAPI(report),
	}, nil
}

// AddWorkflowExecutionMetrics merges the input metrics for the workflow execution identified by (namespace,uid)
func (s *WorkflowServer) AddWorkflowExecutionMetrics(ctx context.Context, req *api.AddWorkflowExecutionsMetricsRequest) (*api.WorkflowExecutionsMetricsResponse, error) {
	client := getClient(ctx)

	metrics := converter.APIMetricsToCore(req.Metrics)
	workflowExecution, err := client.AddWorkflowExecutionMetrics(req.Namespace, req.Uid, metrics, req.Override)
	if err != nil {
		return nil, err
	}

	resp := &api.WorkflowExecutionsMetricsResponse{
		Metrics: converter.MetricsToAPI(workflowExecution.Metrics),
	}

	return resp, nil
}

// UpdateWorkflowExecutionMetrics replaces the metrics with the input metrics for the workflow identified by (namespace, uid)
func (s *WorkflowServer) UpdateWorkflowExecutionMetrics(ctx context.Context, req *api.UpdateWorkflowExecutionsMetricsRequest) (*api.WorkflowExecutionsMetricsResponse, error) {
	client := getClient(ctx)

	metrics := converter.APIMetricsToCore(req.Metrics)
	workflowExecution, err := client.UpdateWorkflowExecutionMetrics(req.Namespace, req.Uid, metrics)
	if err != nil {
		return nil, err
	}

	resp := &api.WorkflowExecutionsMetricsResponse{
		Metrics: converter.MetricsToAPI(workflowExecution.Metrics),
	}

	return resp, nil
}

// ListWorkflowExecutionsField returns a list of all the distinct values of a field from WorkflowExecutions
func (s *WorkflowServer) ListWorkflowExecutionsField(ctx context.Context, req *api.ListWorkflowExecutionsFieldRequest) (*api.ListWorkflowExecutionsFieldResponse, error) {
	client := getClient(ctx)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "list", "onepanel.io", "workspaces", "")
	if err != nil || !allowed {
		return nil, err
	}

	values, err := client.ListWorkflowExecutionsField(req.Namespace, req.FieldName)
	if err != nil {
		return nil, err
	}

	return &api.ListWorkflowExecutionsFieldResponse{
		Values: values,
	}, nil
}
