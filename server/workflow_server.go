package server

import (
	"context"
	"errors"

	"github.com/onepanelio/core/api"
	"github.com/onepanelio/core/manager"
	"github.com/onepanelio/core/model"
	"github.com/onepanelio/core/util"
	"github.com/onepanelio/core/util/ptr"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var userError *util.UserError

type WorkflowServer struct {
	resourceManager *manager.ResourceManager
}

func NewWorkflowServer(resourceManager *manager.ResourceManager) *WorkflowServer {
	return &WorkflowServer{resourceManager: resourceManager}
}

func (s *WorkflowServer) CreateWorkflow(ctx context.Context, req *api.CreateWorkflowRequest) (*api.Workflow, error) {
	workflow := &model.Workflow{
		WorkflowTemplate: model.WorkflowTemplate{
			Manifest: req.Workflow.WorkflowTemplate.Manifest,
		},
	}
	for _, param := range req.Workflow.Parameters {
		workflow.Parameters = append(workflow.Parameters, model.Parameter{
			Name:  param.Name,
			Value: ptr.String(param.Value),
		})
	}

	createdWorkflow, err := s.resourceManager.CreateWorkflow(req.Namespace, workflow)
	if err != nil {
		return nil, err
	}
	req.Workflow = &api.Workflow{
		Name: createdWorkflow.Name,
		Uid:  createdWorkflow.UID,
	}

	return req.Workflow, nil
}

func (s *WorkflowServer) CreateWorkflowTemplate(ctx context.Context, req *api.CreateWorkflowTemplateRequest) (*api.WorkflowTemplate, error) {
	workflowTemplate := &model.WorkflowTemplate{
		Name:     req.WorkflowTemplate.Name,
		Manifest: req.WorkflowTemplate.Manifest,
	}
	workflowTemplate, err := s.resourceManager.CreateWorkflowTemplate(req.Namespace, workflowTemplate)
	if errors.As(err, &userError) {
		if userError.Code == 409 {
			return nil, status.Errorf(codes.Aborted, err.Error())
		}
		return nil, status.Errorf(codes.Unknown, err.Error())
	}
	req.WorkflowTemplate.Uid = workflowTemplate.UID
	req.WorkflowTemplate.Version = workflowTemplate.Version.String()

	return req.WorkflowTemplate, nil
}
