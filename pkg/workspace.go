package v1

import (
	"github.com/onepanelio/core/pkg/util"
	"github.com/onepanelio/core/pkg/util/ptr"
	"github.com/onepanelio/core/pkg/util/validate"
	"google.golang.org/grpc/codes"
)

func injectWorkspaceParameterValues(workspace *Workspace, workspaceAction, resourceAction string) (err error) {
	for i, p := range workspace.Parameters {
		// TODO: This can be removed when we have validation on param level
		if p.Name == "sys-name" {
			if p.Value == nil || !validate.IsDNSHost(*p.Value) {
				return util.NewUserError(codes.InvalidArgument, "Workspace name is not valid.")
			}
		}

		if p.Name == "sys-workspace-action" {
			workspace.Parameters[i].Value = ptr.String(workspaceAction)
		}

		if p.Name == "sys-resource-action" {
			workspace.Parameters[i].Value = ptr.String(resourceAction)
		}
	}

	return
}

func (c *Client) CreateWorkspace(namespace string, workspace *Workspace) (err error) {
	if err = injectWorkspaceParameterValues(workspace, "create", "apply"); err != nil {
		return
	}

	workflowTemplate, err := c.getWorkspaceTemplateWorkflowTemplate(namespace,
		workspace.WorkspaceTemplate.UID, workspace.WorkspaceTemplate.Version)
	if err != nil {
		return util.NewUserError(codes.NotFound, "Workspace template not found.")
	}

	_, err = c.CreateWorkflowExecution(namespace, &WorkflowExecution{
		Parameters:       workspace.Parameters,
		WorkflowTemplate: workflowTemplate,
	})

	return
}
