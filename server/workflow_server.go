package server

import (
	"context"
	"errors"

	"github.com/onepanelio/core/api"
	"github.com/onepanelio/core/manager"
	"github.com/onepanelio/core/model"
	"github.com/onepanelio/core/util"
	"github.com/onepanelio/core/util/ptr"
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
			UID:     req.Workflow.WorkflowTemplate.Uid,
			Version: req.Workflow.WorkflowTemplate.Version,
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

func (s *WorkflowServer) GetWorkflow(ctx context.Context, req *api.GetWorkflowRequest) (*api.Workflow, error) {
	wf, err := s.resourceManager.GetWorkflow(req.Namespace, req.Name)
	if errors.As(err, &userError) {
		return nil, userError.GRPCError()
	}

	workflow := &api.Workflow{
		Uid:  string(wf.UID),
		Name: wf.Name,
		WorkflowTemplate: &api.WorkflowTemplate{
			Uid:     wf.WorkflowTemplate.UID,
			Version: wf.WorkflowTemplate.Version,
		},
	}

	return workflow, nil
}

func (s *WorkflowServer) ListWorkflows(ctx context.Context, req *api.ListWorkflowsRequest) (*api.ListWorkflowsResponse, error) {
	return nil, nil
}

func (s *WorkflowServer) CreateWorkflowTemplate(ctx context.Context, req *api.CreateWorkflowTemplateRequest) (*api.WorkflowTemplate, error) {
	workflowTemplate := &model.WorkflowTemplate{
		Name:     req.WorkflowTemplate.Name,
		Manifest: req.WorkflowTemplate.Manifest,
	}
	workflowTemplate, err := s.resourceManager.CreateWorkflowTemplate(req.Namespace, workflowTemplate)
	if errors.As(err, &userError) {
		return nil, userError.GRPCError()
	}
	req.WorkflowTemplate.Uid = workflowTemplate.UID
	req.WorkflowTemplate.Version = workflowTemplate.Version

	return req.WorkflowTemplate, nil
}

func (s *WorkflowServer) GetWorkflowTemplate(ctx context.Context, req *api.GetWorkflowTemplateRequest) (*api.WorkflowTemplate, error) {
	workflowTemplate, err := s.resourceManager.GetWorkflowTemplate(req.Namespace, req.Uid, req.Version)
	if errors.As(err, &userError) {
		return nil, userError.GRPCError()
	}

	return &api.WorkflowTemplate{
		Uid:      workflowTemplate.UID,
		Version:  workflowTemplate.Version,
		Manifest: workflowTemplate.Manifest,
	}, nil
}

func (s *WorkflowServer) ListWorkflowTemplateVersions(ctx context.Context, req *api.ListWorkflowTemplateVersionsRequest) (*api.ListWorkflowTemplateVersionsResponse, error) {
	workflowTemplateVersions, err := s.resourceManager.ListWorkflowTemplateVersions(req.Namespace, req.Uid)
	if errors.As(err, &userError) {
		return nil, userError.GRPCError()
	}

	workflowTemplates := []*api.WorkflowTemplate{}
	for _, wtv := range workflowTemplateVersions {
		workflowTemplates = append(workflowTemplates, &api.WorkflowTemplate{
			Uid:      wtv.UID,
			Name:     wtv.Name,
			Version:  wtv.Version,
			Manifest: wtv.Manifest,
		})
	}

	return &api.ListWorkflowTemplateVersionsResponse{
		Count:             int32(len(workflowTemplateVersions)),
		WorkflowTemplates: workflowTemplates,
	}, nil
}
