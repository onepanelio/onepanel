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
		Schedule:                   cwf.Schedule,
		Timezone:                   cwf.Timezone,
		Suspend:                    cwf.Suspend,
		ConcurrencyPolicy:          cwf.ConcurrencyPolicy,
		StartingDeadlineSeconds:    *cwf.StartingDeadlineSeconds,
		SuccessfulJobsHistoryLimit: *cwf.SuccessfulJobsHistoryLimit,
		FailedJobsHistoryLimit:     *cwf.FailedJobsHistoryLimit,
		WorkflowExecution:          GenApiWorkflowExecution(cwf.WorkflowExecution),
	}
	return
}

func (c *CronWorkflowServer) CreateCronWorkflow(ctx context.Context, req *api.CreateWorkflowRequest) (*api.CronWorkflow, error) {
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

	cronWorkflow := &v1.CronWorkflow{
		WorkflowExecution: workflow,
	}

	cwf, err := client.CreateCronWorkflow(req.Namespace, cronWorkflow)
	if err != nil {
		return nil, err
	}
	return apiCronWorkflow(cwf), nil
}
