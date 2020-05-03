package v1

import (
	sq "github.com/Masterminds/squirrel"
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

func (c *Client) createWorkspace(namespace string, workspace *Workspace) (*Workspace, error) {
	workflowExecution, err := c.CreateWorkflowExecution(namespace, &WorkflowExecution{
		Parameters:       workspace.Parameters,
		WorkflowTemplate: workspace.WorkspaceTemplate.WorkflowTemplate,
	})
	if err != nil {
		return nil, err
	}

	workspace.UID = workflowExecution.UID

	err = sb.Insert("workspaces").
		SetMap(sq.Eq{
			"uid":                   workspace.UID,
			"name":                  workspace.Name,
			"namespace":             namespace,
			"workspace_template_id": workspace.WorkspaceTemplate.ID,
			"workflow_execution_id": workflowExecution.ID,
		}).
		Suffix("RETURNING id, created_at").
		RunWith(c.DB).
		QueryRow().Scan(&workspace.ID, &workspace.CreatedAt)
	if err != nil {
		return nil, err
	}

	return workspace, nil
}

// CreateWorkspace creates a workspace by triggering the corresponding workflow
func (c *Client) CreateWorkspace(namespace string, workspace *Workspace) (*Workspace, error) {
	if err := injectWorkspaceParameterValues(workspace, "create", "apply"); err != nil {
		return nil, err
	}

	workspaceTemplate, err := c.getWorkspaceTemplate(namespace,
		workspace.WorkspaceTemplate.UID, workspace.WorkspaceTemplate.Version)
	if err != nil {
		return nil, util.NewUserError(codes.NotFound, "Workspace template not found.")
	}

	workspace.WorkspaceTemplate.ID = workspaceTemplate.ID
	workspace.WorkspaceTemplate = workspaceTemplate

	workspace, err = c.createWorkspace(namespace, workspace)
	if err != nil {
		return nil, util.NewUserError(codes.Unknown, "Could not create workspace.")
	}

	return workspace, nil
}
