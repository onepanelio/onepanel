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

	_, err = r.sb.Insert("workflow_template_versions").
		SetMap(sq.Eq{
			"workflow_template_id": workflowTemplate.ID,
			"manifest":             workflowTemplate.Manifest,
			"version":              int32(time.Now().Unix()),
		}).
		RunWith(tx).Exec()
	if err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return workflowTemplate, nil
}

func (r *WorkflowRepository) workflowTemplatesSelectBuilder(namespace, uid string) sq.SelectBuilder {
	sb := r.sb.Select("wt.uid", "wt.name", "wtv.version").
		From("workflow_template_versions wtv").
		Join("workflow_templates wt ON wt.id = wtv.workflow_template_id").
		Where(sq.Eq{
			"wt.uid":       uid,
			"wt.namespace": namespace,
		}).
		OrderBy("wtv.version desc")

	return sb
}

func (r *WorkflowRepository) GetWorkflowTemplate(namespace, uid string, version int32) (workflowTemplate *model.WorkflowTemplate, err error) {
	workflowTemplate = &model.WorkflowTemplate{}

	sb := r.workflowTemplatesSelectBuilder(namespace, uid).Columns("wtv.manifest").Limit(1)
	if version != 0 {
		sb = sb.Where(sq.Eq{"wtv.version": version})
	}
	query, args, err := sb.ToSql()
	if err != nil {
		return
	}

	err = r.db.Get(workflowTemplate, query, args...)
	if err == sql.ErrNoRows {
		err = nil
		workflowTemplate = nil
	}

	return
}

func (r *WorkflowRepository) ListWorkflowTemplateVersions(namespace, uid string) (workflowTemplateVersions []*model.WorkflowTemplate, err error) {
	workflowTemplateVersions = []*model.WorkflowTemplate{}

	query, args, err := r.workflowTemplatesSelectBuilder(namespace, uid).ToSql()
	if err != nil {
		return
	}

	err = r.db.Select(&workflowTemplateVersions, query, args...)

	return
}
