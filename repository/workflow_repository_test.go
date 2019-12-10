package repository

import (
	"testing"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/onepanelio/core/model"
)

func TestWorkflowRepositoryCreate(t *testing.T) {
	uid, err := uuid.NewRandom()
	if err != nil {
		t.Error(err)
		return
	}
	workflow := &model.Workflow{
		UID: uid.String(),
	}

	sql, args, err := sq.Insert("workflows").
		SetMap(sq.Eq{
			"UID": workflow.UID,
		}).ToSql()
	if err != nil {
		t.Error(err)
		return
	}
	t.Log(args)
	t.Log(sql)
}
