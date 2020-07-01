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
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"strings"
	"time"
)

func (c *Client) workspacesSelectBuilder(namespace string) sq.SelectBuilder {
	sb := sb.Select(getWorkspaceColumns("w")...).
		Columns(getWorkspaceStatusColumns("w", "status")...).
		Columns(getWorkspaceTemplateColumns("wt", "workspace_template")...).
		Columns(getWorkflowTemplateVersionColumns("wftv", "workflow_template_version")...).
		Columns("wtv.version \"workspace_template.version\"", `wtv.manifest "workspace_template.manifest"`).
		From("workspaces w").
		Join("workspace_templates wt ON wt.id = w.workspace_template_id").
		Join("workspace_template_versions wtv ON wtv.workspace_template_id = wt.id AND wtv.version = w.workspace_template_version").
		Join("workflow_template_versions wftv ON wftv.workflow_template_id = wt.workflow_template_id AND wftv.version = w.workspace_template_version").
		Where(sq.Eq{
			"w.namespace": namespace,
		})

	return sb
}

// mergeWorkspaceParameters combines two parameter arrays. If a parameter in newParameters is not in
// the existing ones, it is added. If it is, it is ignored.
func mergeWorkspaceParameters(existingParameters, newParameters []Parameter) (parameters []Parameter) {
	parameterMap := make(map[string]*string, 0)
	for _, p := range newParameters {
		parameterMap[p.Name] = p.Value
		parameters = append(parameters, Parameter{
			Name:  p.Name,
			Value: p.Value,
		})
	}

	for _, p := range existingParameters {
		_, ok := parameterMap[p.Name]
		if !ok {
			parameters = append(parameters, Parameter{
				Name:  p.Name,
				Value: p.Value,
			})
		}
	}

	return parameters
}

// Injects parameters into the workspace.Parameters.
// If the parameter already exists, it's value is updated.
// The parameters injected are:
// sys-name
// sys-workspace-action
// sys-resource-action
// sys-host
// TODO why does this generate the UID?
func injectWorkspaceSystemParameters(namespace string, workspace *Workspace, workspaceAction, resourceAction string, config SystemConfig) (err error) {
	if err := workspace.GenerateUID(workspace.Name); err != nil {
		return err
	}
	host := fmt.Sprintf("%v--%v.%v", workspace.UID, namespace, *config.Domain())
	systemParameters := []Parameter{
		{
			Name:  "sys-workspace-action",
			Value: ptr.String(workspaceAction),
		},
		{
			Name:  "sys-resource-action",
			Value: ptr.String(resourceAction),
		},
		{
			Name:  "sys-host",
			Value: ptr.String(host),
		},
	}
	workspace.Parameters = mergeWorkspaceParameters(workspace.Parameters, systemParameters)

	return
}

// createWorkspace creates a workspace and related resources.
// The following are required on the workspace:
//   WorkspaceTemplate.WorkflowTemplate.UID
//   WorkspaceTemplate.WorkflowTemplate.Version
func (c *Client) createWorkspace(namespace string, parameters []byte, workspace *Workspace) (*Workspace, error) {
	systemConfig, err := c.GetSystemConfig()
	if err != nil {
		return nil, err
	}

	workflowTemplate, err := c.GetWorkflowTemplate(namespace, workspace.WorkspaceTemplate.WorkflowTemplate.UID, workspace.WorkspaceTemplate.WorkflowTemplate.Version)
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace": namespace,
			"Workspace": workspace,
			"Error":     err.Error(),
		}).Error("Error with getting workflow template.")
		return nil, util.NewUserError(codes.NotFound, "Error with getting workflow template.")
	}

	runtimeParameters, err := generateRuntimeParameters(systemConfig)
	if err != nil {
		return nil, err
	}

	runtimeParametersMap := make(map[string]*string)
	for _, p := range runtimeParameters {
		runtimeParametersMap[p.Name] = p.Value
	}

	argoTemplate := workflowTemplate.ArgoWorkflowTemplate
	for i, p := range argoTemplate.Spec.Arguments.Parameters {
		value := runtimeParametersMap[p.Name]
		if value != nil {
			argoTemplate.Spec.Arguments.Parameters[i].Value = value
		}
	}

	_, err = c.CreateWorkflowExecution(namespace, &WorkflowExecution{
		Parameters: workspace.Parameters,
	}, workflowTemplate)
	if err != nil {
		return nil, err
	}

	err = sb.Insert("workspaces").
		SetMap(sq.Eq{
			"uid":                        workspace.UID,
			"name":                       workspace.Name,
			"namespace":                  namespace,
			"parameters":                 parameters,
			"phase":                      WorkspaceLaunching,
			"started_at":                 time.Now().UTC(),
			"workspace_template_id":      workspace.WorkspaceTemplate.ID,
			"workspace_template_version": workspace.WorkspaceTemplate.Version,
		}).
		Suffix("RETURNING id, created_at").
		RunWith(c.DB).
		QueryRow().
		Scan(&workspace.ID, &workspace.CreatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "invalid input syntax for type json") {
			return nil, util.NewUserError(codes.InvalidArgument, err.Error())
		}

		return nil, util.NewUserError(codes.Unknown, err.Error())
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

	err = injectWorkspaceSystemParameters(namespace, workspace, "create", "apply", config)
	if err != nil {
		return nil, err
	}
	workspace.Parameters = append(workspace.Parameters, Parameter{
		Name:  "sys-uid",
		Value: ptr.String(workspace.UID),
	})

	sysHost := workspace.GetParameterValue("sys-host")
	if sysHost == nil {
		return nil, fmt.Errorf("sys-host parameter not found")
	}

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

	workspaceTemplate, err := c.GetWorkspaceTemplate(namespace, workspace.WorkspaceTemplate.UID, workspace.WorkspaceTemplate.Version)
	if err != nil {
		return nil, util.NewUserError(codes.NotFound, "Workspace template not found.")
	}
	workspace.WorkspaceTemplate = workspaceTemplate

	workspace, err = c.createWorkspace(namespace, parameters, workspace)
	if err != nil {
		return nil, err
	}

	if _, err := c.InsertLabels(TypeWorkspace, workspace.ID, workspace.Labels); err != nil {
		return nil, err
	}

	return workspace, nil
}

// GetWorkspace loads a workspace for a given namespace, uid. This loads database data
// injects any runtime data, and loads the labels
func (c *Client) GetWorkspace(namespace, uid string) (workspace *Workspace, err error) {
	sb := c.workspacesSelectBuilder(namespace).
		Where(sq.And{
			sq.Eq{"w.uid": uid},
			sq.NotEq{"w.phase": WorkspaceTerminated},
		})

	workspace = &Workspace{}
	if err = c.DB.Getx(workspace, sb); err != nil {
		if err == sql.ErrNoRows {
			err = nil
			workspace = nil
		}

		return
	}

	workspace.WorkspaceTemplate.WorkflowTemplate = &WorkflowTemplate{
		Manifest: workspace.WorkflowTemplateVersion.Manifest,
	}

	configMap, err := c.GetSystemConfig()
	if err != nil {
		return nil, err
	}
	if err := workspace.WorkspaceTemplate.InjectRuntimeParameters(configMap); err != nil {
		return nil, err
	}
	workspace.WorkflowTemplateVersion.Manifest = workspace.WorkspaceTemplate.WorkflowTemplate.Manifest

	if err = json.Unmarshal(workspace.ParametersBytes, &workspace.Parameters); err != nil {
		return
	}

	labelsMap, err := c.GetDBLabelsMapped(TypeWorkspace, workspace.ID)
	if err != nil {
		return nil, err
	}

	workspace.Labels = labelsMap[workspace.ID]

	return
}

// UpdateWorkspaceStatus updates workspace status and times based on phase
func (c *Client) UpdateWorkspaceStatus(namespace, uid string, status *WorkspaceStatus) (err error) {
	fieldMap := sq.Eq{
		"phase":       status.Phase,
		"modified_at": time.Now().UTC(),
	}
	switch status.Phase {
	case WorkspaceLaunching:
		fieldMap["paused_at"] = pq.NullTime{}
		fieldMap["started_at"] = time.Now().UTC()
		break
	case WorkspacePausing:
		fieldMap["started_at"] = pq.NullTime{}
		fieldMap["paused_at"] = time.Now().UTC()
		break
	case WorkspaceUpdating:
		fieldMap["paused_at"] = pq.NullTime{}
		fieldMap["updated_at"] = time.Now().UTC()
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

// ListWorkspacesByTemplateID will return all the workspaces for a given workspace template id.
// Sourced from database.
func (c *Client) ListWorkspacesByTemplateID(namespace string, templateID uint64) (workspaces []*Workspace, err error) {
	sb := sb.Select(getWorkspaceColumns("w")...).
		From("workspaces w").
		Where(sq.And{
			sq.Eq{
				"w.namespace":             namespace,
				"w.workspace_template_id": templateID,
			},
			sq.NotEq{
				"phase": WorkspaceTerminated,
			},
		})

	if err := c.DB.Selectx(&workspaces, sb); err != nil {
		return nil, err
	}

	labelMap, err := c.GetDBLabelsMapped(TypeWorkspace, WorkspacesToIDs(workspaces)...)
	if err != nil {
		return nil, err
	}

	for _, workspace := range workspaces {
		workspace.Labels = labelMap[workspace.ID]
	}
	return
}

func (c *Client) ListWorkspaces(namespace string, paginator *pagination.PaginationRequest) (workspaces []*Workspace, err error) {
	sb := sb.Select(getWorkspaceColumns("w")...).
		Columns(getWorkspaceStatusColumns("w", "status")...).
		Columns(getWorkspaceTemplateColumns("wt", "workspace_template")...).
		From("workspaces w").
		Join("workspace_templates wt ON wt.id = w.workspace_template_id").
		OrderBy("w.created_at DESC").
		Where(sq.And{
			sq.Eq{
				"w.namespace": namespace,
			},
			sq.NotEq{
				"phase": WorkspaceTerminated,
			},
		})
	sb = *paginator.ApplyToSelect(&sb)

	if err := c.DB.Selectx(&workspaces, sb); err != nil {
		return nil, err
	}

	labelMap, err := c.GetDBLabelsMapped(TypeWorkspace, WorkspacesToIDs(workspaces)...)
	if err != nil {
		return nil, err
	}

	for _, workspace := range workspaces {
		workspace.Labels = labelMap[workspace.ID]
	}

	return
}

// CountWorkspaces returns the total number of workspaces in the given namespace that are not terminated
func (c *Client) CountWorkspaces(namespace string) (count int, err error) {
	err = sb.Select("COUNT( DISTINCT( w.id ))").
		From("workspaces w").
		Join("workspace_templates wt ON w.workspace_template_id = wt.id").
		Where(sq.And{
			sq.Eq{
				"w.namespace": namespace,
			},
			sq.NotEq{
				"phase": WorkspaceTerminated,
			},
		}).
		RunWith(c.DB).
		QueryRow().
		Scan(&count)

	return
}

func (c *Client) updateWorkspace(namespace, uid, workspaceAction, resourceAction string, status *WorkspaceStatus, parameters ...Parameter) (err error) {
	workspace, err := c.GetWorkspace(namespace, uid)
	if err != nil {
		return util.NewUserError(codes.Unknown, err.Error())
	}
	if workspace == nil {
		return util.NewUserError(codes.NotFound, "Workspace not found.")
	}

	config, err := c.GetSystemConfig()
	if err != nil {
		return
	}

	workspace.Parameters = mergeWorkspaceParameters(workspace.Parameters, parameters)
	parametersJSON, err := json.Marshal(workspace.Parameters)
	if err != nil {
		return
	}
	workspace.Parameters = append(workspace.Parameters, Parameter{
		Name:  "sys-uid",
		Value: ptr.String(uid),
	})
	err = injectWorkspaceSystemParameters(namespace, workspace, workspaceAction, resourceAction, config)
	if err != nil {
		return
	}

	workspaceTemplate, err := c.GetWorkspaceTemplate(namespace,
		workspace.WorkspaceTemplate.UID, workspace.WorkspaceTemplate.Version)
	if err != nil {
		return util.NewUserError(codes.NotFound, "Workspace template not found.")
	}

	workflowTemplate, err := c.GetWorkflowTemplate(namespace, workspaceTemplate.WorkflowTemplate.UID, workspaceTemplate.WorkflowTemplate.Version)
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace": namespace,
			"Workspace": workspace,
			"Error":     err.Error(),
		}).Error("Error with getting workflow template.")
		return util.NewUserError(codes.NotFound, "Error with getting workflow template.")
	}
	workspaceTemplate.WorkflowTemplate = workflowTemplate
	workspace.WorkspaceTemplate = workspaceTemplate

	_, err = c.CreateWorkflowExecution(namespace, &WorkflowExecution{
		Parameters: workspace.Parameters,
	}, workspaceTemplate.WorkflowTemplate)
	if err != nil {
		return
	}

	if err = c.UpdateWorkspaceStatus(namespace, uid, status); err != nil {
		return
	}

	// Update parameters if they are passed
	if len(parameters) == 0 {
		return
	}

	_, err = sb.Update("workspaces").
		SetMap(sq.Eq{
			"parameters": parametersJSON,
		}).
		Where(sq.And{
			sq.Eq{
				"namespace": namespace,
				"uid":       uid,
			}, sq.NotEq{
				"phase": WorkspaceTerminated,
			},
		}).
		RunWith(c.DB).
		Exec()
	if err != nil {
		return util.NewUserError(codes.NotFound, "Workspace not found.")
	}

	return
}

func (c *Client) UpdateWorkspace(namespace, uid string, parameters []Parameter) (err error) {
	return c.updateWorkspace(namespace, uid, "update", "apply", &WorkspaceStatus{Phase: WorkspaceUpdating}, parameters...)
}

func (c *Client) PauseWorkspace(namespace, uid string) (err error) {
	return c.updateWorkspace(namespace, uid, "pause", "delete", &WorkspaceStatus{Phase: WorkspacePausing})
}

func (c *Client) ResumeWorkspace(namespace, uid string) (err error) {
	return c.updateWorkspace(namespace, uid, "create", "apply", &WorkspaceStatus{Phase: WorkspaceLaunching})
}

func (c *Client) DeleteWorkspace(namespace, uid string) (err error) {
	return c.updateWorkspace(namespace, uid, "delete", "delete", &WorkspaceStatus{Phase: WorkspaceTerminating})
}

// ArchiveWorkspace archives by setting the workspace to delete or terminate.
// Kicks off DB archiving and k8s cleaning.
func (c *Client) ArchiveWorkspace(namespace, uid string) (err error) {
	return c.updateWorkspace(namespace, uid, "delete", "delete", &WorkspaceStatus{Phase: WorkspaceTerminating})
}
