package repository

import (
	"database/sql"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/onepanelio/core/model"
)

type WorkflowRepository struct {
	db *DB
	sb sq.StatementBuilderType
}

func NewWorkflowRepository(db *DB) *WorkflowRepository {
	return &WorkflowRepository{db: db, sb: sq.StatementBuilder.PlaceholderFormat(sq.Dollar)}
}

func (r *WorkflowRepository) insertWorkflowTemplateVersion(workflowTemplate *model.WorkflowTemplate, runner sq.BaseRunner) (err error) {
	err = r.sb.Insert("workflow_template_versions").
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

func (r *WorkflowRepository) CreateWorkflowTemplate(namespace string, workflowTemplate *model.WorkflowTemplate) (*model.WorkflowTemplate, error) {
	uid, err := workflowTemplate.GenerateUID()
	if err != nil {
		return nil, err
	}

	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	err = r.sb.Insert("workflow_templates").
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

	if err = r.insertWorkflowTemplateVersion(workflowTemplate, tx); err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return workflowTemplate, nil
}

func (r *WorkflowRepository) RemoveIsLatestFromWorkflowTemplateVersions(workflowTemplate *model.WorkflowTemplate) error {
	query, args, err := r.sb.Update("workflow_template_versions").
		Set("is_latest", true).
		Where(sq.Eq{
			"workflow_template_id": workflowTemplate.ID,
			"is_latest":            false,
		}).
		ToSql()
	if err != nil {
		return err
	}

	if _, err := r.db.Exec(query, args...); err != nil {
		return err
	}

	return nil
}

func (r *WorkflowRepository) CreateWorkflowTemplateVersion(namespace string, workflowTemplate *model.WorkflowTemplate) (*model.WorkflowTemplate, error) {
	query, args, err := r.sb.Select("id, name").
		From("workflow_templates").
		Where(sq.Eq{
			"namespace": namespace,
			"uid":       workflowTemplate.UID,
		}).
		Limit(1).ToSql()
	if err != nil {
		return nil, err
	}

	if err = r.db.Get(workflowTemplate, query, args...); err == sql.ErrNoRows {
		return nil, nil
	}

	if err = r.insertWorkflowTemplateVersion(workflowTemplate, r.db.Base()); err != nil {
		return nil, err
	}

	return workflowTemplate, nil
}

func (r *WorkflowRepository) UpdateWorkflowTemplateVersion(workflowTemplate *model.WorkflowTemplate) (*model.WorkflowTemplate, error) {
	query, args, err := r.sb.Update("workflow_template_versions").
		Set("manifest", workflowTemplate.Manifest).
		Where(sq.Eq{
			"workflow_template_id": workflowTemplate.ID,
			"version":              workflowTemplate.Version,
		}).
		ToSql()

	if err != nil {
		return nil, err
	}

	if _, err := r.db.Exec(query, args...); err != nil {
		return nil, err
	}

	return workflowTemplate, nil
}

func (r *WorkflowRepository) workflowTemplatesSelectBuilder(namespace string) sq.SelectBuilder {
	sb := r.sb.Select("wt.id", "wt.created_at", "wt.uid", "wt.name", "wtv.version", "wtv.is_latest").
		From("workflow_template_versions wtv").
		Join("workflow_templates wt ON wt.id = wtv.workflow_template_id").
		Where(sq.Eq{
			"wt.namespace": namespace,
		})

	return sb
}

func (r *WorkflowRepository) GetWorkflowTemplate(namespace, uid string, version int32) (workflowTemplate *model.WorkflowTemplate, err error) {
	workflowTemplate = &model.WorkflowTemplate{}

	sb := r.workflowTemplatesSelectBuilder(namespace).Where(sq.Eq{"wt.uid": uid}).
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

	if err = r.db.Get(workflowTemplate, query, args...); err == sql.ErrNoRows {
		err = nil
		workflowTemplate = nil
	}

	return
}

func (r *WorkflowRepository) ListWorkflowTemplateVersions(namespace, uid string) (workflowTemplateVersions []*model.WorkflowTemplate, err error) {
	workflowTemplateVersions = []*model.WorkflowTemplate{}

	query, args, err := r.workflowTemplatesSelectBuilder(namespace).Where(sq.Eq{"wt.uid": uid}).
		Columns("wtv.manifest").
		OrderBy("wtv.version desc").ToSql()
	if err != nil {
		return
	}

	err = r.db.Select(&workflowTemplateVersions, query, args...)

	return
}

func (r *WorkflowRepository) ListWorkflowTemplates(namespace string) (workflowTemplateVersions []*model.WorkflowTemplate, err error) {
	workflowTemplateVersions = []*model.WorkflowTemplate{}

	query, args, err := r.workflowTemplatesSelectBuilder(namespace).
		Options("DISTINCT ON (wt.id) wt.id,").
		OrderBy("wt.id desc").ToSql()
	if err != nil {
		return
	}

	err = r.db.Select(&workflowTemplateVersions, query, args...)

	return
}
