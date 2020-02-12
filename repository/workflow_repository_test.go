package repository

import (
	"database/sql"
	"github.com/DATA-DOG/go-sqlmock"
	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
	"github.com/onepanelio/core/model"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
	"time"
)

func TestWorkflowRepository_GetWorkflowTemplate(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	dbRepo := getDBRepo(db)
	defer dbRepo.Close()

	row := sqlmock.NewRows([]string{"id", "created_at", "uid", "name", "is_archived", "version", "is_latest", "manifest"}).
		AddRow(1, time.Time{}.UTC(), "1", "name", false, 1, false, "")
	mock.ExpectQuery("SELECT wt.id, wt.created_at, wt.uid, wt.name, wt.is_archived, wtv.version, wtv.is_latest, wtv.manifest" +
		" FROM workflow_template_versions wtv JOIN workflow_templates wt ON wt.id = wtv.workflow_template_id " +
		"WHERE (.+)").WillReturnRows(row)

	workflowRepo := NewWorkflowRepository(&dbRepo)
	workflowGet, err := workflowRepo.GetWorkflowTemplate("default", "1", 1)
	if err != nil {
		t.Errorf("Unexpected err: %s", err)
	}
	print(workflowGet)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

/**
This relies on int32(time.Now().Unix()), to execute fast enough that
there won't be a time difference. This should be reviewed.
*/
func TestWorkflowRepository_InsertWorkflowTemplateVersion(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	dbRepo := getDBRepo(db)
	namespace := "default"
	workflowModel := model.WorkflowTemplate{
		ID:         1,
		CreatedAt:  time.Time{},
		UID:        "test",
		Name:       "test",
		Manifest:   "test",
		Version:    1,
		IsLatest:   true,
		IsArchived: false,
	}

	defer db.Close()
	mock.ExpectQuery("INSERT INTO workflow_template_versions \\(is_latest,manifest,version,workflow_template_id\\) VALUES \\(\\$1,\\$2,\\$3,\\$4\\) RETURNING version").
		WithArgs(workflowModel.IsLatest, workflowModel.Manifest, int32(time.Now().Unix()), workflowModel.ID).
		WillReturnRows(sqlmock.NewRows([]string{"version"}).AddRow(workflowModel.Version))

	workflowRepo := NewWorkflowRepository(&dbRepo)
	_, err2 := workflowRepo.CreateWorkflowTemplateVersion(namespace, &workflowModel)
	if err2 != nil {
		t.Fatalf("an error '%s' was not expected", err2)
	}
	// we make sure that all expectations were met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

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

func getDBRepo(db *sql.DB) DB {
	return DB{
		DB: &sqlx.DB{
			DB:     db,
			Mapper: reflectx.NewMapperFunc("db", strings.ToLower),
		},
	}
}
