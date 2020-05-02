package v1

import (
	"github.com/onepanelio/core/pkg/util"
	"github.com/onepanelio/core/pkg/util/ptr"
	"github.com/onepanelio/core/pkg/util/validate"
	"google.golang.org/grpc/codes"
)

func injectWorkspaceParameterValues(workspace *Workspace, workspaceAction, resourceAction string) (err error) {
	for _, p := range workspace.Parameters {
		if p.Name == "sys-name" {
			// TODO: These if statements can be removed when we have validation on param level
			if p.Value == nil {
				return util.NewUserError(codes.InvalidArgument, "Workspace name is required.")
			}
			if !validate.IsDNSHost(*p.Value) {
				return util.NewUserError(codes.InvalidArgument, "Workspace name is not valid.")
			}
			workspace.Name = *p.Value
		}
	}
	workspace.Parameters = append(workspace.Parameters, Parameter{
		Name:  "sys-workspace-action",
		Value: ptr.String(workspaceAction),
	}, Parameter{
		Name:  "sys-resource-action",
		Value: ptr.String(resourceAction),
	})

	return
}

func (c *Client) CreateWorkspace(namespace string, workspace *Workspace) (*Workspace, error) {
	if err := injectWorkspaceParameterValues(workspace, "create", "apply"); err != nil {
		return nil, err
	}

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
