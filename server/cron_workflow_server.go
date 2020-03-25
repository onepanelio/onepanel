package server

import (
	"github.com/onepanelio/core/api"
	"context"
)

type CronWorkflowServer struct{}

func NewCronWorkflowServer() *CronWorkflowServer {
	return &CronWorkflowServer{}
}

func (c CronWorkflowServer) CreateCronWorkflow(ctx context.Context, req *api.CreateWorkflowRequest) (*api.CronWorkflow, error) {
	panic("implement me")
func apiCronWorkflow(cwf *v1.CronWorkflow) (cronWorkflow *api.CronWorkflow) {
	cronWorkflow = &api.CronWorkflow{
		Schedule:                   cwf.Schedule,
		Timezone:                   cwf.Timezone,
		Suspend:                    cwf.Suspend,
		ConcurrencyPolicy:          cwf.ConcurrencyPolicy,
		StartingDeadlineSeconds:    cwf.StartingDeadlineSeconds,
		SuccessfulJobsHistoryLimit: cwf.SuccessfulJobsHistoryLimit,
		FailedJobsHistoryLimit:     cwf.FailedJobsHistoryLimit,
		WorkflowExecution:          GenApiWorkflowExecution(cwf.WorkflowExecution),
	}
	return
}
}
