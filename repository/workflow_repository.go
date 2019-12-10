package repository

import (
	sq "github.com/Masterminds/squirrel"

	"github.com/onepanelio/core/model"
)

type WorkflowRepositoryInterface interface {
	Create(*model.Workflow) error
}

type WorkflowRepository struct {
	db *DB
}

func NewWorkflowRepository(db *DB) *WorkflowRepository {
	return &WorkflowRepository{db: db}
}

func (w *WorkflowRepository) Create(workflow *model.Workflow) (err error) {
	sql, _, err := sq.Insert("workflows").
		SetMap(sq.Eq{
			"UID": workflow.UID,
		}).ToSql()
	if err != nil {
		return
	}

	err = w.db.NamedQueryWithStructScan(sql, workflow)

	return
}
