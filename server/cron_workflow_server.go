package server

import (
	"context"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/onepanelio/core/api"
	v1 "github.com/onepanelio/core/pkg"
	"github.com/onepanelio/core/pkg/util/ptr"
	"github.com/onepanelio/core/pkg/util/request/pagination"
	"github.com/onepanelio/core/server/auth"
	"github.com/onepanelio/core/server/converter"
)

type CronWorkflowServer struct{}

func NewCronWorkflowServer() *CronWorkflowServer {
	return &CronWorkflowServer{}
}

func apiCronWorkflow(cwf *v1.CronWorkflow) (cronWorkflow *api.CronWorkflow) {
	if cwf == nil {
		return nil
	}

	cronWorkflow = &api.CronWorkflow{
		Name:      cwf.Name,
		Uid:       cwf.UID,
		Labels:    converter.MappingToKeyValue(cwf.Labels),
		Manifest:  cwf.Manifest,
		Namespace: cwf.Namespace,
	}

	if cwf.WorkflowExecution != nil {
		cronWorkflow.WorkflowExecution = apiWorkflowExecution(cwf.WorkflowExecution, nil)
		for _, param := range cwf.WorkflowExecution.Parameters {
			convertedParam := &api.Parameter{
				Name:  param.Name,
				Value: *param.Value,
			}
			cronWorkflow.WorkflowExecution.Parameters = append(cronWorkflow.WorkflowExecution.Parameters, convertedParam)
		}
	}

	return
}

func (c *CronWorkflowServer) CreateCronWorkflow(ctx context.Context, req *api.CreateCronWorkflowRequest) (*api.CronWorkflow, error) {
	client := getClient(ctx)
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
		options := make([]*v1.ParameterOption, 0)
		for _, option := range param.Options {
			options = append(options, &v1.ParameterOption{
				Name:  option.Name,
				Value: option.Value,
			})
		}

		workflow.Parameters = append(workflow.Parameters, v1.Parameter{
			Name:        param.Name,
			Value:       ptr.String(param.Value),
			Type:        param.Type,
			DisplayName: &param.DisplayName,
			Hint:        &param.Hint,
			Options:     options,
			Required:    param.Required,
		})
	}

	cronWorkflow := v1.CronWorkflow{
		WorkflowExecution: workflow,
		Manifest:          req.CronWorkflow.Manifest,
		Labels:            converter.APIKeyValueToLabel(req.CronWorkflow.Labels),
		Namespace:         req.Namespace,
	}

	cwf, err := client.CreateCronWorkflow(req.Namespace, &cronWorkflow)
	if err != nil {
		return nil, err
	}

	return apiCronWorkflow(cwf), nil
}

func (c *CronWorkflowServer) UpdateCronWorkflow(ctx context.Context, req *api.UpdateCronWorkflowRequest) (*api.CronWorkflow, error) {
	client := getClient(ctx)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "update", "argoproj.io", "cronworkflows", "")
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
		options := make([]*v1.ParameterOption, 0)
		for _, option := range param.Options {
			options = append(options, &v1.ParameterOption{
				Name:  option.Name,
				Value: option.Value,
			})
		}

		workflow.Parameters = append(workflow.Parameters, v1.Parameter{
			Name:        param.Name,
			Value:       ptr.String(param.Value),
			Type:        param.Type,
			DisplayName: &param.DisplayName,
			Hint:        &param.Hint,
			Options:     options,
			Required:    param.Required,
		})
	}

	cronWorkflow := v1.CronWorkflow{
		WorkflowExecution: workflow,
		Manifest:          req.CronWorkflow.Manifest,
		Labels:            converter.APIKeyValueToLabel(req.CronWorkflow.Labels),
		Namespace:         req.Namespace,
	}

	cwf, err := client.UpdateCronWorkflow(req.Namespace, req.Uid, &cronWorkflow)
	if err != nil {
		return nil, err
	}
	if cwf == nil {
		return nil, nil
	}
	return apiCronWorkflow(cwf), nil
}

func (c *CronWorkflowServer) GetCronWorkflow(ctx context.Context, req *api.GetCronWorkflowRequest) (*api.CronWorkflow, error) {
	client := getClient(ctx)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "get", "argoproj.io", "cronworkflows", req.Uid)
	if err != nil || !allowed {
		return nil, err
	}
	cwf, err := client.GetCronWorkflow(req.Namespace, req.Uid)
	if err != nil {
		return nil, err
	}
	return apiCronWorkflow(cwf), nil
}

func (c *CronWorkflowServer) ListCronWorkflows(ctx context.Context, req *api.ListCronWorkflowRequest) (*api.ListCronWorkflowsResponse, error) {
	client := getClient(ctx)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "list", "argoproj.io", "cronworkflows", "")
	if err != nil || !allowed {
		return nil, err
	}

	paginator := pagination.NewRequest(req.Page, req.PageSize)
	cronWorkflows, err := client.ListCronWorkflows(req.Namespace, req.WorkflowTemplateName, &paginator)
	if err != nil {
		return nil, err
	}
	var apiCronWorkflows []*api.CronWorkflow
	for _, cwf := range cronWorkflows {
		apiCronWorkflows = append(apiCronWorkflows, apiCronWorkflow(cwf))
	}

	count, err := client.CountCronWorkflows(req.Namespace, req.WorkflowTemplateName)
	if err != nil {
		return nil, err
	}

	return &api.ListCronWorkflowsResponse{
		Count:         int32(len(apiCronWorkflows)),
		CronWorkflows: apiCronWorkflows,
		Page:          int32(paginator.Page),
		Pages:         paginator.CalculatePages(count),
		TotalCount:    int32(count),
	}, nil
}

func (c *CronWorkflowServer) DeleteCronWorkflow(ctx context.Context, req *api.DeleteCronWorkflowRequest) (*empty.Empty, error) {
	client := getClient(ctx)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "delete", "argoproj.io", "cronworkflows", "")
	if err != nil || !allowed {
		return nil, err
	}

	err = client.TerminateCronWorkflow(req.Namespace, req.Uid)
	if err != nil {
		return nil, err
	}

	return &empty.Empty{}, nil
}
