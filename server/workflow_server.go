package server

import (
	"context"

	"github.com/onepanelio/core/api"
	"github.com/onepanelio/core/manager"
)

type WorkflowServer struct {
	resourceManager *manager.ResourceManager
}

func NewWorkflowServer(resourceManager *manager.ResourceManager) *WorkflowServer {
	return &WorkflowServer{resourceManager: resourceManager}
}

func (w *WorkflowServer) Create(ctx context.Context, req *api.CreateWorkflowRequest) (*api.Workflow, error) {
	return req.Workflow, nil
}
