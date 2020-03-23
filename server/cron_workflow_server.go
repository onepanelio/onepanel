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
}
