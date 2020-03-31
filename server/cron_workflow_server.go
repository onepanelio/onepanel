package server

import (
	"context"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/onepanelio/core/api"
	v1 "github.com/onepanelio/core/pkg"
	"github.com/onepanelio/core/pkg/util/ptr"
	"github.com/onepanelio/core/server/auth"
	"math"
)

type CronWorkflowServer struct{}

func NewCronWorkflowServer() *CronWorkflowServer {
	return &CronWorkflowServer{}
}

func apiCronWorkflow(cwf *v1.CronWorkflow) (cronWorkflow *api.CronWorkflow) {
	cronWorkflow = &api.CronWorkflow{
		Name:              cwf.Name,
		Schedule:          cwf.Schedule,
		Timezone:          cwf.Timezone,
		Suspend:           cwf.Suspend,
		ConcurrencyPolicy: cwf.ConcurrencyPolicy,
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
		workflow.Parameters = append(workflow.Parameters, v1.WorkflowExecutionParameter{
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
	}

	cwf, err := client.CreateCronWorkflow(req.Namespace, &cronWorkflow)
	if err != nil {
		return nil, err
	}
	if cwf == nil {
		return nil, nil
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
		workflow.Parameters = append(workflow.Parameters, v1.WorkflowExecutionParameter{
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

	if req.PageSize <= 0 {
		req.PageSize = 15
	}

	cronWorkflows, err := client.ListCronWorkflows(req.Namespace)
	if err != nil {
		return nil, err
	}
	var apiCronWorkflows []*api.CronWorkflow
	for _, cwf := range cronWorkflows {
		apiCronWorkflows = append(apiCronWorkflows, apiCronWorkflow(cwf))
	}

	pages := int32(math.Ceil(float64(len(apiCronWorkflows)) / float64(req.PageSize)))
	if req.Page > pages {
		req.Page = pages
	}

	if req.Page <= 0 {
		req.Page = 1
	}

	start := (req.Page - 1) * req.PageSize
	end := start + req.PageSize
	if end >= int32(len(apiCronWorkflows)) {
		end = int32(len(apiCronWorkflows))
	}

	return &api.ListCronWorkflowsResponse{
		Count:         end - start,
		CronWorkflows: apiCronWorkflows[start:end],
		Page:          req.Page,
		Pages:         pages,
		TotalCount:    int32(len(apiCronWorkflows)),
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
