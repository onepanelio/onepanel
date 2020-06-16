package v1

import (
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

var mockWorkflowExecutionLabels = sqlmock.NewRows(
	[]string{"id", "created_at", "key", "value", "resource", "resource_id"}).
	AddRow(1, time.Time{}.UTC(), "os", "linux", "workflow_execution", 1).
	AddRow(2, time.Time{}.UTC(), "env", "tensorflow", "workflow_execution", 1)

func TestListLabels(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	c := NewTestClient(db)

	mock.ExpectQuery("SELECT l.id, l.created_at, l.key, l.value, l.resource, l.resource_id FROM labels l JOIN workflow_executions we ON we.id = l.resource_id WHERE resource = \\$1 AND we.uid = \\$2 ORDER BY l.created_at").
		WithArgs("workflow_execution", "workflow-1").
		WillReturnRows(mockWorkflowExecutionLabels)

	labels, err := c.ListLabels("workflow_execution", "workflow-1")
	if err != nil {
		t.Errorf("error was not expected while listing labels: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.NotEmpty(t, labels)
	assert.NoError(t, err)
	assert.Equal(t, labels[0].ID, uint64(1))
	assert.Equal(t, labels[1].ID, uint64(2))
}
