package server

import (
	"context"
	"github.com/onepanelio/core/api"
	v1 "github.com/onepanelio/core/pkg"
	"github.com/onepanelio/core/pkg/util/ptr"
	"github.com/onepanelio/core/server/auth"
)

type CronWorkflowServer struct{}

func NewCronWorkflowServer() *CronWorkflowServer {
	return &CronWorkflowServer{}
}

func apiCronWorkflow(cwf *v1.CronWorkflow) (cronWorkflow *api.CronWorkflow) {
	cronWorkflow = &api.CronWorkflow{
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
