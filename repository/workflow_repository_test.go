package repository

import (
	"testing"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/onepanelio/core/model"
)

func TestWorkflowRepositoryCreate(t *testing.T) {
	uuid, err := uuid.NewRandom()
	if err != nil {
		t.Error(err)
		return
	}
	workflow := &model.Workflow{
		UUID: uuid,
	}

	sql, args, err := sq.Insert("workflows").
		SetMap(sq.Eq{
			"UUID": workflow.UUID,
		}).ToSql()
	if err != nil {
		t.Error(err)
		return
	}
	t.Log(args)
	t.Log(sql)
}
