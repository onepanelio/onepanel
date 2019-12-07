package server

import (
	"context"

	"github.com/onepanelio/core/api"
	"github.com/onepanelio/core/repository"
)

type WorkflowServer struct {
	workflowRepository *repository.WorkflowRepository
}

func NewWorkflowServer(workflowRepository *repository.WorkflowRepository) *WorkflowServer {
	return &WorkflowServer{workflowRepository: workflowRepository}
}

func (w *WorkflowServer) Create(ctx context.Context, req *api.CreateWorkflowRequest) (*api.Workflow, error) {
	return req.Workflow, nil
}
