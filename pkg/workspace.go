package v1

import (
	"database/sql"
	"encoding/json"
	"fmt"
	sq "github.com/Masterminds/squirrel"
	"github.com/asaskevich/govalidator"
	"github.com/lib/pq"
	"github.com/onepanelio/core/pkg/util"
	"github.com/onepanelio/core/pkg/util/pagination"
	"github.com/onepanelio/core/pkg/util/ptr"
	"google.golang.org/grpc/codes"
	"time"
)

func (c *Client) workspacesSelectBuilder(namespace string) sq.SelectBuilder {
	sb := sb.Select("w.id", "w.uid", "w.name", "w.parameters", "wt.id \"workspace_template.id\"", "wt.uid \"workspace_template.uid\"", "wtv.version \"workspace_template.version\"").
		Columns(getWorkspaceStatusColumns("w", "status")...).
		From("workspaces w").
		Join("workspace_templates wt ON wt.id = w.workspace_template_id").
		Join("workspace_template_versions wtv ON wtv.workspace_template_id = wt.id AND wtv.version = w.workspace_template_version").
		Where(sq.Eq{
			"w.namespace": namespace,
		})

	return sb
}

// Injects parameters into the workspace.Parameters.
// If the parameter already exists, it's value is updated.
// The parameters injected are:
// sys-name
// sys-workspace-action
// sys-resource-action
// sys-host
func injectWorkspaceSystemParameters(namespace string, workspace *Workspace, workspaceAction, resourceAction string, config map[string]string) (parameterMap map[string]Parameter, err error) {
	host := fmt.Sprintf("%v--%v.%v", workspace.Name, namespace, config["ONEPANEL_DOMAIN"])
	if _, err = workspace.GenerateUID(); err != nil {
		return
	}

	insertionMap := map[string]Parameter{
		"sys-workspace-action": {
			Name:  "sys-workspace-action",
			Value: ptr.String(workspaceAction),
		},
		"sys-resource-action": {
			Name:  "sys-resource-action",
			Value: ptr.String(resourceAction),
		},
		"sys-host": {
			Name:  "sys-host",
			Value: ptr.String(host),
		},
	}

	for i := range workspace.Parameters {
		parameter := &workspace.Parameters[i]
		existingParam, ok := insertionMap[parameter.Name]
		if ok {
			parameter.Value = existingParam.Value
			delete(insertionMap, parameter.Name)
		}
	}

	for _, parameter := range insertionMap {
		workspace.Parameters = append(workspace.Parameters, parameter)
	}

	parameterMap = make(map[string]Parameter)
	for _, parameter := range workspace.Parameters {
		parameterMap[parameter.Name] = parameter
	}

	return
}

func (c *Client) createWorkspace(namespace string, parameters []byte, workspace *Workspace) (*Workspace, error) {
	_, err := c.CreateWorkflowExecution(namespace, &WorkflowExecution{
		Parameters:       workspace.Parameters,
		WorkflowTemplate: workspace.WorkspaceTemplate.WorkflowTemplate,
	})
	if err != nil {
		return nil, err
	}

	err = sb.Insert("workspaces").
		SetMap(sq.Eq{
			"uid":                        workspace.UID,
			"name":                       workspace.Name,
			"namespace":                  namespace,
			"parameters":                 parameters,
			"phase":                      WorkspaceStarted,
			"started_at":                 time.Now().UTC(),
			"workspace_template_id":      workspace.WorkspaceTemplate.ID,
			"workspace_template_version": workspace.WorkspaceTemplate.Version,
			"url":                        workspace.URL,
		}).
		Suffix("RETURNING id, created_at").
		RunWith(c.DB).
		QueryRow().Scan(&workspace.ID, &workspace.CreatedAt)
	if err != nil {
		return nil, util.NewUserErrorWrap(err, "Workspace")
	}

	return workspace, nil
}

// CreateWorkspace creates a workspace by triggering the corresponding workflow
func (c *Client) CreateWorkspace(namespace string, workspace *Workspace) (*Workspace, error) {
	config, err := c.GetSystemConfig()
	if err != nil {
		return nil, err
	}

	parameters, err := json.Marshal(workspace.Parameters)
	if err != nil {
		return nil, err
	}

	parameterMap, err := injectWorkspaceSystemParameters(namespace, workspace, "create", "apply", config)
	if err != nil {
		return nil, err
	}
	workspace.Parameters = append(workspace.Parameters, Parameter{
		Name:  "sys-uid",
		Value: ptr.String(workspace.UID),
	})

	sysHost, ok := parameterMap["sys-host"]
	if !ok {
		return nil, fmt.Errorf("sys-host parameter not found")
	}
	workspace.URL = "http://" + *sysHost.Value

	existingWorkspace, err := c.GetWorkspace(namespace, workspace.UID)
	if err != nil {
		return nil, err
	}
	if existingWorkspace != nil {
		return nil, util.NewUserError(codes.AlreadyExists, "Workspace already exists.")
	}

	// Validate workspace fields
	valid, err := govalidator.ValidateStruct(workspace)
	if err != nil || !valid {
		return nil, util.NewUserError(codes.InvalidArgument, err.Error())
	}

	workspaceTemplate, err := c.GetWorkspaceTemplate(namespace,
		workspace.WorkspaceTemplate.UID, workspace.WorkspaceTemplate.Version)
	if err != nil {
		return nil, util.NewUserError(codes.NotFound, "Workspace template not found.")
	}
	workspace.WorkspaceTemplate = workspaceTemplate

	workspace, err = c.createWorkspace(namespace, parameters, workspace)
	if err != nil {
		return nil, err
	}

	return workspace, nil
}

func (c *Client) GetWorkspace(namespace, uid string) (workspace *Workspace, err error) {
	query, args, err := c.workspacesSelectBuilder(namespace).
		Where(sq.And{
			sq.Eq{"w.uid": uid},
			sq.NotEq{"w.phase": WorkspaceTerminated},
		}).ToSql()
	if err != nil {
		return
	}

	workspace = &Workspace{}
	if err = c.DB.Get(workspace, query, args...); err == sql.ErrNoRows {
		err = nil
		workspace = nil

		return
	}
	if err != nil {
		return nil, err
	}

	if err = json.Unmarshal(workspace.ParametersBytes, &workspace.Parameters); err != nil {
		return
	}

	labelsMap, err := c.GetDbLabelsMapped(TypeWorkspace, workspace.ID)
	if err != nil {
		return nil, err
	}

	workspace.Labels = labelsMap[workspace.ID]

	return
}

// UpdateWorkspaceStatus updates workspace status and times based on phase
func (c *Client) UpdateWorkspaceStatus(namespace, uid string, status *WorkspaceStatus) (err error) {
	fieldMap := sq.Eq{
		"phase": status.Phase,
	}
	switch status.Phase {
	case WorkspaceStarted:
		fieldMap["paused_at"] = pq.NullTime{}
		fieldMap["started_at"] = time.Now().UTC()
		break
	case WorkspacePausing:
		fieldMap["started_at"] = pq.NullTime{}
		fieldMap["paused_at"] = time.Now().UTC()
		break
	case WorkspaceTerminating:
		fieldMap["started_at"] = pq.NullTime{}
		fieldMap["paused_at"] = pq.NullTime{}
		fieldMap["terminated_at"] = time.Now().UTC()
		break
	}
	_, err = sb.Update("workspaces").
		SetMap(fieldMap).
		Where(sq.And{
			sq.Eq{
				"namespace": namespace,
				"uid":       uid,
			}, sq.NotEq{
				"phase": WorkspaceTerminated,
			},
		}).
		RunWith(c.DB).Exec()
	if err != nil {
		return util.NewUserError(codes.NotFound, "Workspace not found.")
	}

	return
}

func (c *Client) ListWorkspaces(namespace string, paginator *pagination.PaginationRequest) (workspaces []*Workspace, err error) {
	sb := sb.Select(getWorkspaceColumns("w", "")...).
		Columns(getWorkspaceStatusColumns("w", "status")...).
		Columns(getWorkspaceTemplateColumns("wt", "workspace_template")...).
		From("workspaces w").
		Join("workspace_templates wt ON wt.id = w.workspace_template_id").
		OrderBy("w.created_at DESC").
		Where(sq.Eq{
			"w.namespace": namespace,
		})
	sb = *paginator.ApplyToSelect(&sb)

	query, args, err := sb.ToSql()
	if err != nil {
		return nil, err
	}

	if err := c.DB.Select(&workspaces, query, args...); err != nil {
		return nil, err
	}

	return
}

func (c *Client) CountWorkspaces(namespace string) (count int, err error) {
	err = sb.Select("COUNT( DISTINCT( w.id ))").
		From("workspaces w").
		Join("workspace_templates wt ON w.workspace_template_id = wt.id").
		Where(sq.Eq{
			"wt.namespace":   namespace,
			"wt.is_archived": false,
		}).
		RunWith(c.DB.DB).
		QueryRow().
		Scan(&count)

	return
}

func (c *Client) updateWorkspace(namespace, uid, workspaceAction, resourceAction string, status *WorkspaceStatus) (err error) {
	workspace, err := c.GetWorkspace(namespace, uid)
	if err != nil {
		return util.NewUserError(codes.NotFound, "Workspace not found.")
	}

	config, err := c.GetSystemConfig()
	if err != nil {
		return
	}

	workspace.Parameters = append(workspace.Parameters, Parameter{
		Name:  "sys-uid",
		Value: ptr.String(uid),
	})
	_, err = injectWorkspaceSystemParameters(namespace, workspace, workspaceAction, resourceAction, config)
	if err != nil {
		return
	}

	if err = c.UpdateWorkspaceStatus(namespace, uid, status); err != nil {
		return
	}

	workspaceTemplate, err := c.GetWorkspaceTemplate(namespace,
		workspace.WorkspaceTemplate.UID, workspace.WorkspaceTemplate.Version)
	if err != nil {
		return util.NewUserError(codes.NotFound, "Workspace template not found.")
	}
	workspace.WorkspaceTemplate = workspaceTemplate

	_, err = c.CreateWorkflowExecution(namespace, &WorkflowExecution{
		Parameters:       workspace.Parameters,
		WorkflowTemplate: workspace.WorkspaceTemplate.WorkflowTemplate,
	})
	if err != nil {
		return
	}

	return
}

func (c *Client) PauseWorkspace(namespace, uid string) (err error) {
	return c.updateWorkspace(namespace, uid, "pause", "delete", &WorkspaceStatus{Phase: WorkspacePausing})
}

func (c *Client) ResumeWorkspace(namespace, uid string) (err error) {
	return c.updateWorkspace(namespace, uid, "create", "apply", &WorkspaceStatus{Phase: WorkspaceStarted})
}

func (c *Client) DeleteWorkspace(namespace, uid string) (err error) {
	return c.updateWorkspace(namespace, uid, "delete", "delete", &WorkspaceStatus{Phase: WorkspaceTerminating})
}
