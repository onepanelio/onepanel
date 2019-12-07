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

func (w *WorkflowServer) Create(c context.Context, request *api.CreateWorkflowRequest) (*api.Workflow, error) {
	return &api.Workflow{Uuid: "uuid", Name: "name"}, nil
}
