package migration

import (
	"database/sql"
	v1 "github.com/onepanelio/core/pkg"
	"github.com/pressly/goose"
)

func initialize20200728190804() {
	if _, ok := initializedMigrations[20200728190804]; !ok {
		goose.AddMigration(Up20200728190804, Down20200728190804)
		initializedMigrations[20200728190804] = true
	}
}

// Up20200728190804 is a legacy migration. Due to code changes, it no longer does anything.
// It used to update labels so that we keep track of WorkflowTemplate labels.
// Before, only workflow template versions had labels, but to speed up some queries, we now cache the latest version's labels
// for workflow templates themselves.
func Up20200728190804(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	if migrationHasAlreadyBeenRun(20200728190804) {
		return nil
	}

	// Do nothing, be preserve for legacy.

	return nil
}

// Down20200728190804 rolls down the migration by deleting all workflow template labels, since they did not exist before this
func Down20200728190804(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.

	client, err := getClient()
	if err != nil {
		return err
	}
	defer client.DB.Close()

	return client.DeleteResourceLabels(tx, v1.TypeWorkflowTemplate)
}
