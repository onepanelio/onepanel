package repository

import (
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

func (r *WorkflowRepository) CreateWorkflowTemplate(workflowTemplate *model.WorkflowTemplate) (*model.WorkflowTemplate, error) {
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
			"uid":  uid,
			"name": workflowTemplate.Name,
		}).
		Suffix("RETURNING id").
		RunWith(tx).
		QueryRow().Scan(&workflowTemplate.ID)
	if err != nil {
		return nil, err
	}

	err = r.sb.Insert("workflow_template_versions").
		SetMap(sq.Eq{
			"workflow_template_id": workflowTemplate.ID,
			"manifest":             workflowTemplate.Manifest,
		}).
		Suffix("RETURNING version").
		RunWith(tx).
		QueryRow().Scan(&workflowTemplate.Version)
	if err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return workflowTemplate, nil
}
