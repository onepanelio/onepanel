package server

import (
	"context"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/onepanelio/core/api"
	v1 "github.com/onepanelio/core/pkg"
	"github.com/onepanelio/core/pkg/util/pagination"
	"github.com/onepanelio/core/pkg/util/ptr"
	"github.com/onepanelio/core/server/auth"
	"github.com/onepanelio/core/server/converter"
)

type CronWorkflowServer struct{}

func NewCronWorkflowServer() *CronWorkflowServer {
	return &CronWorkflowServer{}
}

func apiCronWorkflow(cwf *v1.CronWorkflow) (cronWorkflow *api.CronWorkflow) {
	if cwf == nil {
		return nil
	}

	cronWorkflow = &api.CronWorkflow{
		Name:              cwf.Name,
		Schedule:          cwf.Schedule,
		Timezone:          cwf.Timezone,
		Suspend:           cwf.Suspend,
		ConcurrencyPolicy: cwf.ConcurrencyPolicy,
		Labels:            converter.MappingToKeyValue(cwf.Labels),
	}

	if cwf.StartingDeadlineSeconds != nil {
		cronWorkflow.StartingDeadlineSeconds = *cwf.StartingDeadlineSeconds
	}

	if cwf.SuccessfulJobsHistoryLimit != nil {
		cronWorkflow.SuccessfulJobsHistoryLimit = *cwf.SuccessfulJobsHistoryLimit
	}

	if cwf.FailedJobsHistoryLimit != nil {
		cronWorkflow.FailedJobsHistoryLimit = *cwf.FailedJobsHistoryLimit
	}

	if cwf.WorkflowExecution != nil {
		cronWorkflow.WorkflowExecution = GenApiWorkflowExecution(cwf.WorkflowExecution)
		for _, param := range cwf.WorkflowExecution.Parameters {
			convertedParam := &api.Parameter{
				Name:  param.Name,
				Value: *param.Value,
			}
			cronWorkflow.WorkflowExecution.Parameters = append(cronWorkflow.WorkflowExecution.Parameters, convertedParam)
		}
	}

	return
}

func (c *CronWorkflowServer) CreateCronWorkflow(ctx context.Context, req *api.CreateCronWorkflowRequest) (*api.CronWorkflow, error) {
	client := ctx.Value("kubeClient").(*v1.Client)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "create", "argoproj.io", "cronworkflows", "")
	if err != nil || !allowed {
		return nil, err
	}

	workflow := &v1.WorkflowExecution{
		WorkflowTemplate: &v1.WorkflowTemplate{
			UID:     req.CronWorkflow.WorkflowExecution.WorkflowTemplate.Uid,
			Version: req.CronWorkflow.WorkflowExecution.WorkflowTemplate.Version,
		},
	}
	for _, param := range req.CronWorkflow.WorkflowExecution.Parameters {
		workflow.Parameters = append(workflow.Parameters, v1.Parameter{
			Name:  param.Name,
			Value: ptr.String(param.Value),
		})
	}

	cronWorkflow := v1.CronWorkflow{
		Schedule:                   req.CronWorkflow.Schedule,
		Timezone:                   req.CronWorkflow.Timezone,
		Suspend:                    req.CronWorkflow.Suspend,
		ConcurrencyPolicy:          req.CronWorkflow.ConcurrencyPolicy,
		StartingDeadlineSeconds:    &req.CronWorkflow.StartingDeadlineSeconds,
		SuccessfulJobsHistoryLimit: &req.CronWorkflow.SuccessfulJobsHistoryLimit,
		FailedJobsHistoryLimit:     &req.CronWorkflow.FailedJobsHistoryLimit,
		WorkflowExecution:          workflow,
		Labels:                     converter.APIKeyValueToLabel(req.CronWorkflow.Labels),
	}

	cwf, err := client.CreateCronWorkflow(req.Namespace, &cronWorkflow)
	if err != nil {
		return nil, err
	}

	return apiCronWorkflow(cwf), nil
}

func (c *CronWorkflowServer) UpdateCronWorkflow(ctx context.Context, req *api.UpdateCronWorkflowRequest) (*api.CronWorkflow, error) {
	client := ctx.Value("kubeClient").(*v1.Client)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "update", "argoproj.io", "cronworkflows", "")
	if err != nil || !allowed {
		return nil, err
	}
	workflow := &v1.WorkflowExecution{
		WorkflowTemplate: &v1.WorkflowTemplate{
			UID:     req.CronWorkflow.WorkflowExecution.WorkflowTemplate.Uid,
			Version: req.CronWorkflow.WorkflowExecution.WorkflowTemplate.Version,
		},
	}
	for _, param := range req.CronWorkflow.WorkflowExecution.Parameters {
		workflow.Parameters = append(workflow.Parameters, v1.Parameter{
			Name:  param.Name,
			Value: ptr.String(param.Value),
		})
	}

	cronWorkflow := v1.CronWorkflow{
		Schedule:                   req.CronWorkflow.Schedule,
		Timezone:                   req.CronWorkflow.Timezone,
		Suspend:                    req.CronWorkflow.Suspend,
		ConcurrencyPolicy:          req.CronWorkflow.ConcurrencyPolicy,
		StartingDeadlineSeconds:    &req.CronWorkflow.StartingDeadlineSeconds,
		SuccessfulJobsHistoryLimit: &req.CronWorkflow.SuccessfulJobsHistoryLimit,
		FailedJobsHistoryLimit:     &req.CronWorkflow.FailedJobsHistoryLimit,
		WorkflowExecution:          workflow,
		Labels:                     converter.APIKeyValueToLabel(req.CronWorkflow.Labels),
	}

	cwf, err := client.UpdateCronWorkflow(req.Namespace, req.Name, &cronWorkflow)
	if err != nil {
		return nil, err
	}
	if cwf == nil {
		return nil, nil
	}
	return apiCronWorkflow(cwf), nil
}

func (c *CronWorkflowServer) GetCronWorkflow(ctx context.Context, req *api.GetCronWorkflowRequest) (*api.CronWorkflow, error) {
	client := ctx.Value("kubeClient").(*v1.Client)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "get", "argoproj.io", "cronworkflows", req.Name)
	if err != nil || !allowed {
		return nil, err
	}
	cwf, err := client.GetCronWorkflow(req.Namespace, req.Name)
	if err != nil {
		return nil, err
	}
	return apiCronWorkflow(cwf), nil
}

func (c *CronWorkflowServer) ListCronWorkflows(ctx context.Context, req *api.ListCronWorkflowRequest) (*api.ListCronWorkflowsResponse, error) {
	client := ctx.Value("kubeClient").(*v1.Client)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "list", "argoproj.io", "cronworkflows", "")
	if err != nil || !allowed {
		return nil, err
	}

	paginator := pagination.NewRequest(req.Page, req.PageSize)
	cronWorkflows, err := client.ListCronWorkflows(req.Namespace, req.WorkflowTemplateUid, &paginator)
	if err != nil {
		return nil, err
	}
	var apiCronWorkflows []*api.CronWorkflow
	for _, cwf := range cronWorkflows {
		apiCronWorkflows = append(apiCronWorkflows, apiCronWorkflow(cwf))
	}

	count, err := client.CountCronWorkflows(req.Namespace, req.WorkflowTemplateUid)
	if err != nil {
		return nil, err
	}

	return &api.ListCronWorkflowsResponse{
		Count:         int32(len(apiCronWorkflows)),
		CronWorkflows: apiCronWorkflows,
		Page:          int32(paginator.Page),
		Pages:         paginator.CalculatePages(count),
		TotalCount:    int32(count),
	}, nil
}

func (c *CronWorkflowServer) TerminateCronWorkflow(ctx context.Context, req *api.TerminateCronWorkflowRequest) (*empty.Empty, error) {
	client := ctx.Value("kubeClient").(*v1.Client)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "delete", "argoproj.io", "cronworkflows", "")
	if err != nil || !allowed {
		return nil, err
	}

	err = client.TerminateCronWorkflow(req.Namespace, req.Name)
	if err != nil {
		return nil, err
	}

	return &empty.Empty{}, nil
}
