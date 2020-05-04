package v1

import (
	"encoding/json"
	"fmt"
	sq "github.com/Masterminds/squirrel"
	"github.com/asaskevich/govalidator"
	"github.com/onepanelio/core/pkg/util"
	"github.com/onepanelio/core/pkg/util/ptr"
	"google.golang.org/grpc/codes"
)

func injectWorkspaceSystemParameters(namespace string, workspace *Workspace, workspaceAction, resourceAction string, config map[string]string) (err error) {
	host := fmt.Sprintf("%v--%v.%v", workspace.Name, namespace, config["ONEPANEL_DOMAIN"])
	workspace.Parameters = append(workspace.Parameters,
		Parameter{
			Name:  "sys-name",
			Value: ptr.String(workspace.Name),
		},
		Parameter{
			Name:  "sys-workspace-action",
			Value: ptr.String(workspaceAction),
		}, Parameter{
			Name:  "sys-resource-action",
			Value: ptr.String(resourceAction),
		}, Parameter{
			Name:  "sys-host",
			Value: ptr.String(host),
		})

	return
}

func (c *Client) createWorkspace(namespace string, parameters []byte, workspace *Workspace) (*Workspace, error) {
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
			"uid":                        workspace.UID,
			"name":                       workspace.Name,
			"namespace":                  namespace,
			"parameters":                 parameters,
			"workspace_template_id":      workspace.WorkspaceTemplate.ID,
			"workspace_template_version": workspace.WorkspaceTemplate.Version,
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
	valid, err := govalidator.ValidateStruct(workspace)
	if err != nil || !valid {
		return nil, util.NewUserError(codes.InvalidArgument, err.Error())
	}

	config, err := c.GetSystemConfig()
	if err != nil {
		return nil, err
	}

	parameters, err := json.Marshal(workspace.Parameters)
	if err != nil {
		return nil, err
	}

	if err := injectWorkspaceSystemParameters(namespace, workspace, "create", "apply", config); err != nil {
		return nil, err
	}

	workspaceTemplate, err := c.GetWorkspaceTemplate(namespace,
		workspace.WorkspaceTemplate.UID, workspace.WorkspaceTemplate.Version)
	if err != nil {
		return nil, util.NewUserError(codes.NotFound, "Workspace template not found.")
	}

	workspace.WorkspaceTemplate.ID = workspaceTemplate.ID
	workspace.WorkspaceTemplate = workspaceTemplate

	workspace, err = c.createWorkspace(namespace, parameters, workspace)
	if err != nil {
		return nil, util.NewUserError(codes.Unknown, "Could not create workspace.")
	}

	return workspace, nil
}
