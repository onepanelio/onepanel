package repository

import (
	"testing"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/onepanelio/core/model"
)

func TestWorkflowRepositoryCreateWorkflowTemplate(t *testing.T) {
	uid, err := uuid.NewRandom()
	if err != nil {
		t.Error(err)
		return
	}
	workflowTemplate := &model.WorkflowTemplate{
		UID: uid.String(),
	}

	sql, args, err := sq.Insert("workflow_templates").
		SetMap(sq.Eq{
			"UID": workflowTemplate.UID,
		}).ToSql()
	if err != nil {
		t.Error(err)
		return
	}
	t.Log(sql, args)
}
