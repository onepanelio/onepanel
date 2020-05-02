package v1

import (
	"github.com/onepanelio/core/pkg/util"
	"github.com/onepanelio/core/pkg/util/ptr"
	"github.com/onepanelio/core/pkg/util/validate"
	"google.golang.org/grpc/codes"
)

func injectWorkspaceParameterValues(parameters []Parameter, workspaceAction, resourceAction string) (workspace *Workspace, err error) {
	workspace = &Workspace{}
	for _, p := range parameters {
		// TODO: This can be removed when we have validation on param level
		if p.Name == "sys-name" {
			if p.Value == nil {
				return nil, util.NewUserError(codes.InvalidArgument, "Workspace name is required.")
			}
			if !validate.IsDNSHost(*p.Value) {
				return nil, util.NewUserError(codes.InvalidArgument, "Workspace name is not valid.")
			}
			workspace.Name = *p.Value
		}
	}
	parameters = append(parameters, Parameter{
		Name:  "sys-workspace-action",
		Value: ptr.String(workspaceAction),
	}, Parameter{
		Name:  "sys-resource-action",
		Value: ptr.String(resourceAction),
	})
	workspace.Parameters = parameters

	return
}

func (c *Client) CreateWorkspace(namespace string, workspace *Workspace) (*Workspace, error) {
	ws, err := injectWorkspaceParameterValues(workspace.Parameters, "create", "apply")
	if err != nil {
		return nil, err
	}

	workspace.Name = ws.Name
	workspace.Parameters = ws.Parameters

	workflowTemplate, err := c.getWorkspaceTemplateWorkflowTemplate(namespace,
		workspace.WorkspaceTemplate.UID, workspace.WorkspaceTemplate.Version)
	if err != nil {
		return nil, util.NewUserError(codes.NotFound, "Workspace template not found.")
	}

	workflowExecution, err := c.CreateWorkflowExecution(namespace, &WorkflowExecution{
		Parameters:       workspace.Parameters,
		WorkflowTemplate: workflowTemplate,
	})

	workspace.UID = workflowExecution.UID

	return workspace, nil
}
