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
		Suffix("RETURNING id, uid").
		RunWith(tx).
		QueryRow().Scan(&workflowTemplate.ID, &workflowTemplate.UID)
	if err != nil {
		return nil, err
	}
	tx.Commit()

	return workflowTemplate, nil
}
