package server

import (
	"context"

	"github.com/onepanelio/core/api"
	"github.com/onepanelio/core/manager"
	"github.com/onepanelio/core/model"
)

type WorkflowServer struct {
	resourceManager *manager.ResourceManager
}

func NewWorkflowServer(resourceManager *manager.ResourceManager) *WorkflowServer {
	return &WorkflowServer{resourceManager: resourceManager}
}

func (w *WorkflowServer) Create(ctx context.Context, req *api.CreateWorkflowRequest) (*api.Workflow, error) {
	workflow := &model.Workflow{
		WorkflowTemplate: model.WorkflowTemplate{
			Manifest: req.Workflow.WorkflowTemplate.Manifest,
		},
	}

	createdWorkflow, err := w.resourceManager.CreateWorkflow(req.Namespace, workflow)
	if err != nil {
		return nil, err
	}
	req.Workflow.Name = createdWorkflow.Name

	return req.Workflow, nil
}
