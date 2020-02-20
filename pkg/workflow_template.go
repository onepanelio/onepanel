package v1

import (
	"database/sql"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/onepanelio/core/util"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
)

var sb = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

func (c *Client) insertWorkflowTemplateVersion(workflowTemplate *WorkflowTemplate, runner sq.BaseRunner) (err error) {
	err = sb.Insert("workflow_template_versions").
		SetMap(sq.Eq{
			"workflow_template_id": workflowTemplate.ID,
			"manifest":             workflowTemplate.Manifest,
			"version":              int32(time.Now().Unix()),
			"is_latest":            workflowTemplate.IsLatest,
		}).
		Suffix("RETURNING version").
		RunWith(runner).
		QueryRow().Scan(&workflowTemplate.Version)

	return
}

func (c *Client) createWorkflowTemplate(namespace string, workflowTemplate *WorkflowTemplate) (*WorkflowTemplate, error) {
	uid, err := workflowTemplate.GenerateUID()
	if err != nil {
		return nil, err
	}

	tx, err := c.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	err = sb.Insert("workflow_templates").
		SetMap(sq.Eq{
			"uid":       uid,
			"name":      workflowTemplate.Name,
			"namespace": namespace,
		}).
		Suffix("RETURNING id").
		RunWith(tx).
		QueryRow().Scan(&workflowTemplate.ID)
	if err != nil {
		return nil, err
	}

	if err = c.insertWorkflowTemplateVersion(workflowTemplate, tx); err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return workflowTemplate, nil
}

func (c *Client) removeIsLatestFromWorkflowTemplateVersions(workflowTemplate *WorkflowTemplate) error {
	query, args, err := sb.Update("workflow_template_versions").
		Set("is_latest", true).
		Where(sq.Eq{
			"workflow_template_id": workflowTemplate.ID,
			"is_latest":            false,
		}).
		ToSql()
	if err != nil {
		return err
	}

	if _, err := c.DB.Exec(query, args...); err != nil {
		return err
	}

	return nil
}

func (c *Client) createWorkflowTemplateVersion(namespace string, workflowTemplate *WorkflowTemplate) (*WorkflowTemplate, error) {
	query, args, err := sb.Select("id, name").
		From("workflow_templates").
		Where(sq.Eq{
			"namespace": namespace,
			"uid":       workflowTemplate.UID,
		}).
		Limit(1).ToSql()
	if err != nil {
		return nil, err
	}

	if err = c.DB.Get(workflowTemplate, query, args...); err == sql.ErrNoRows {
		return nil, nil
	}

	if err = c.insertWorkflowTemplateVersion(workflowTemplate, c.DB); err != nil {
		return nil, err
	}

	return workflowTemplate, nil
}

func (c *Client) updateWorkflowTemplateVersion(workflowTemplate *WorkflowTemplate) (*WorkflowTemplate, error) {
	query, args, err := sb.Update("workflow_template_versions").
		Set("manifest", workflowTemplate.Manifest).
		Where(sq.Eq{
			"workflow_template_id": workflowTemplate.ID,
			"version":              workflowTemplate.Version,
		}).
		ToSql()

	if err != nil {
		return nil, err
	}

	if _, err := c.DB.Exec(query, args...); err != nil {
		return nil, err
	}

	return workflowTemplate, nil
}

func (c *Client) workflowTemplatesSelectBuilder(namespace string) sq.SelectBuilder {
	sb := sb.Select("wt.id", "wt.created_at", "wt.uid", "wt.name", "wt.is_archived", "wtv.version", "wtv.is_latest").
		From("workflow_template_versions wtv").
		Join("workflow_templates wt ON wt.id = wtv.workflow_template_id").
		Where(sq.Eq{
			"wt.namespace": namespace,
		})

	return sb
}

func (c *Client) getWorkflowTemplate(namespace, uid string, version int32) (workflowTemplate *WorkflowTemplate, err error) {
	workflowTemplate = &WorkflowTemplate{}

	sb := c.workflowTemplatesSelectBuilder(namespace).Where(sq.Eq{"wt.uid": uid}).
		Columns("wtv.manifest").
		OrderBy("wtv.version desc").
		Limit(1)
	if version != 0 {
		sb = sb.Where(sq.Eq{"wtv.version": version})
	}
	query, args, err := sb.ToSql()
	if err != nil {
		return
	}

	if err = c.DB.Get(workflowTemplate, query, args...); err == sql.ErrNoRows {
		err = nil
		workflowTemplate = nil
	}

	return
}

func (c *Client) listWorkflowTemplateVersions(namespace, uid string) (workflowTemplateVersions []*WorkflowTemplate, err error) {
	workflowTemplateVersions = []*WorkflowTemplate{}

	query, args, err := c.workflowTemplatesSelectBuilder(namespace).Where(sq.Eq{"wt.uid": uid}).
		Columns("wtv.manifest").
		OrderBy("wtv.version desc").ToSql()
	if err != nil {
		return
	}

	err = c.DB.Select(&workflowTemplateVersions, query, args...)

	return
}

func (c *Client) listWorkflowTemplates(namespace string) (workflowTemplateVersions []*WorkflowTemplate, err error) {
	workflowTemplateVersions = []*WorkflowTemplate{}

	query, args, err := c.workflowTemplatesSelectBuilder(namespace).
		Options("DISTINCT ON (wt.id) wt.id,").
		Where(sq.Eq{
			"wt.is_archived": false,
		}).
		OrderBy("wt.id desc").ToSql()
	if err != nil {
		return
	}

	err = c.DB.Select(&workflowTemplateVersions, query, args...)

	return
}

func (c *Client) archiveWorkflowTemplate(namespace, uid string) (bool, error) {
	query, args, err := sb.Update("workflow_templates").
		Set("is_archived", true).
		Where(sq.Eq{
			"uid":       uid,
			"namespace": namespace,
		}).
		ToSql()

	if err != nil {
		return false, err
	}

	if _, err := c.DB.Exec(query, args...); err != nil {
		return false, err
	}

	return true, nil
}

func (c *Client) CreateWorkflowTemplate(namespace string, workflowTemplate *WorkflowTemplate) (*WorkflowTemplate, error) {
	allowed, err := c.IsAuthorized(namespace, "create", "argoproj.io", "workflow", "")
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace":        namespace,
			"WorkflowTemplate": workflowTemplate,
			"Error":            err.Error(),
		}).Error("IsAuthorized failed.")
		return nil, util.NewUserError(codes.Unknown, "Could not create workflow template.")
	}
	if !allowed {
		return nil, util.NewUserError(codes.PermissionDenied, "Permission denied.")
	}

	// validate workflow template
	if err := c.ValidateWorkflow(namespace, workflowTemplate.GetManifestBytes()); err != nil {
		log.WithFields(log.Fields{
			"Namespace":        namespace,
			"WorkflowTemplate": workflowTemplate,
			"Error":            err.Error(),
		}).Error("Workflow could not be validated.")
		return nil, util.NewUserError(codes.InvalidArgument, err.Error())
	}

	workflowTemplate, err = c.createWorkflowTemplate(namespace, workflowTemplate)
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace":        namespace,
			"WorkflowTemplate": workflowTemplate,
			"Error":            err.Error(),
		}).Error("Could not create workflow template.")
		return nil, util.NewUserErrorWrap(err, "Workflow template")
	}

	return workflowTemplate, nil
}

func (c *Client) CreateWorkflowTemplateVersion(namespace string, workflowTemplate *WorkflowTemplate) (*WorkflowTemplate, error) {
	allowed, err := c.IsAuthorized(namespace, "create", "argoproj.io", "workflow", "")
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace":        namespace,
			"WorkflowTemplate": workflowTemplate,
			"Error":            err.Error(),
		}).Error("IsAuthorized failed.")
		return nil, util.NewUserError(codes.Unknown, "Could not create template version.")
	}
	if !allowed {
		return nil, util.NewUserError(codes.PermissionDenied, "Permission denied.")
	}

	// validate workflow template
	if err := c.ValidateWorkflow(namespace, workflowTemplate.GetManifestBytes()); err != nil {
		log.WithFields(log.Fields{
			"Namespace":        namespace,
			"WorkflowTemplate": workflowTemplate,
			"Error":            err.Error(),
		}).Error("Workflow could not be validated.")
		return nil, util.NewUserError(codes.InvalidArgument, err.Error())
	}

	if err := c.removeIsLatestFromWorkflowTemplateVersions(workflowTemplate); err != nil {
		log.WithFields(log.Fields{
			"Namespace":        namespace,
			"WorkflowTemplate": workflowTemplate,
			"Error":            err.Error(),
		}).Error("Could not remove IsLatest from workflow template versions.")
		return nil, util.NewUserError(codes.Unknown, "Unable to Create Workflow Template Version.")
	}

	workflowTemplate, err = c.createWorkflowTemplateVersion(namespace, workflowTemplate)
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace":        namespace,
			"WorkflowTemplate": workflowTemplate,
			"Error":            err.Error(),
		}).Error("Could not create workflow template version.")
		return nil, util.NewUserErrorWrap(err, "Workflow template")
	}
	if workflowTemplate == nil {
		return nil, util.NewUserError(codes.NotFound, "Workflow template not found.")
	}

	return workflowTemplate, nil
}

func (c *Client) UpdateWorkflowTemplateVersion(namespace string, workflowTemplate *WorkflowTemplate) (*WorkflowTemplate, error) {
	allowed, err := c.IsAuthorized(namespace, "update", "argoproj.io", "workflow", "")
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace":        namespace,
			"WorkflowTemplate": workflowTemplate,
			"Error":            err.Error(),
		}).Error("IsAuthorized failed.")
		return nil, util.NewUserError(codes.Unknown, "Could not update workflow template version.")
	}
	if !allowed {
		return nil, util.NewUserError(codes.PermissionDenied, "Permission denied.")
	}

	// validate workflow template
	if err := c.ValidateWorkflow(namespace, workflowTemplate.GetManifestBytes()); err != nil {
		log.WithFields(log.Fields{
			"Namespace":        namespace,
			"WorkflowTemplate": workflowTemplate,
			"Error":            err.Error(),
		}).Error("Workflow could not be validated.")
		return nil, util.NewUserError(codes.InvalidArgument, err.Error())
	}

	originalWorkflowTemplate, err := c.getWorkflowTemplate(namespace, workflowTemplate.UID, workflowTemplate.Version)
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace":        namespace,
			"WorkflowTemplate": workflowTemplate,
			"Error":            err.Error(),
		}).Error("Could not get workflow template.")
		return nil, util.NewUserError(codes.Unknown, "Could not update workflow template version.")
	}

	workflowTemplate.ID = originalWorkflowTemplate.ID
	workflowTemplate, err = c.updateWorkflowTemplateVersion(workflowTemplate)
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace":        namespace,
			"WorkflowTemplate": workflowTemplate,
			"Error":            err.Error(),
		}).Error("Could not update workflow template version.")
		return nil, util.NewUserErrorWrap(err, "Workflow template")
	}
	if workflowTemplate == nil {
		return nil, util.NewUserError(codes.NotFound, "Workflow template not found.")
	}

	return workflowTemplate, nil
}

func (c *Client) GetWorkflowTemplate(namespace, uid string, version int32) (workflowTemplate *WorkflowTemplate, err error) {
	allowed, err := c.IsAuthorized(namespace, "get", "argoproj.io", "workflow", "")
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace":        namespace,
			"WorkflowTemplate": workflowTemplate,
			"Error":            err.Error(),
		}).Error("IsAuthorized failed.")
		return nil, util.NewUserError(codes.Unknown, "Unknown error.")
	}
	if !allowed {
		return nil, util.NewUserError(codes.PermissionDenied, "Permission denied.")
	}

	workflowTemplate, err = c.getWorkflowTemplate(namespace, uid, version)
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace":        namespace,
			"WorkflowTemplate": workflowTemplate,
			"Error":            err.Error(),
		}).Error("Get Workflow Template failed.")
		return nil, util.NewUserError(codes.Unknown, "Unknown error.")
	}
	if workflowTemplate == nil {
		return nil, util.NewUserError(codes.NotFound, "Workflow template not found.")
	}

	return
}

func (c *Client) ListWorkflowTemplateVersions(namespace, uid string) (workflowTemplateVersions []*WorkflowTemplate, err error) {
	allowed, err := c.IsAuthorized(namespace, "list", "argoproj.io", "workflow", "")
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace": namespace,
			"UID":       uid,
			"Error":     err.Error(),
		}).Error("IsAuthorized failed.")
		return nil, util.NewUserError(codes.Unknown, "Unknown error.")
	}
	if !allowed {
		return nil, util.NewUserError(codes.PermissionDenied, "Permission denied.")
	}

	workflowTemplateVersions, err = c.listWorkflowTemplateVersions(namespace, uid)
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace": namespace,
			"UID":       uid,
			"Error":     err.Error(),
		}).Error("Workflow template versions not found.")
		return nil, util.NewUserError(codes.NotFound, "Workflow template versions not found.")
	}

	return
}

func (c *Client) ListWorkflowTemplates(namespace string) (workflowTemplateVersions []*WorkflowTemplate, err error) {
	allowed, err := c.IsAuthorized(namespace, "list", "argoproj.io", "workflow", "")
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace": namespace,
			"Error":     err.Error(),
		}).Error("IsAuthorized failed.")
		return nil, util.NewUserError(codes.Unknown, "Unable to list workflow templates.")
	}
	if !allowed {
		return nil, util.NewUserError(codes.PermissionDenied, "Permission denied.")
	}

	workflowTemplateVersions, err = c.listWorkflowTemplates(namespace)
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace": namespace,
			"Error":     err.Error(),
		}).Error("Workflow templates not found.")
		return nil, util.NewUserError(codes.NotFound, "Workflow templates not found.")
	}

	return
}

func (c *Client) ArchiveWorkflowTemplate(namespace, uid string) (archived bool, err error) {
	allowed, err := c.IsAuthorized(namespace, "delete", "argoproj.io", "workflow", "")
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace": namespace,
			"UID":       uid,
			"Error":     err.Error(),
		}).Error("IsAuthorized failed.")
		return false, util.NewUserError(codes.Unknown, "Unable to archive workflow template.")
	}
	if !allowed {
		return false, util.NewUserError(codes.PermissionDenied, "Permission denied.")
	}

	workflowTemplate, err := c.getWorkflowTemplate(namespace, uid, 0)
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace": namespace,
			"UID":       uid,
			"Error":     err.Error(),
		}).Error("Get Workflow Template failed.")
		return false, util.NewUserError(codes.Unknown, "Unable to archive workflow template.")
	}
	if workflowTemplate == nil {
		return false, util.NewUserError(codes.NotFound, "Workflow template not found.")
	}

	archived, err = c.archiveWorkflowTemplate(namespace, uid)
	if !archived || err != nil {
		if err != nil {
			log.WithFields(log.Fields{
				"Namespace": namespace,
				"UID":       uid,
				"Error":     err.Error(),
			}).Error("Archive Workflow Template failed.")
		}
		return false, util.NewUserError(codes.Unknown, "Unable to archive workflow template.")
	}

	return
}
