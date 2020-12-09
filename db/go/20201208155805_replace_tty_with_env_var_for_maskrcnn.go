package migration

import (
	"database/sql"
	"github.com/pressly/goose"
	"path/filepath"
)

func initialize20201208155805() {
	if _, ok := initializedMigrations[20201208155805]; !ok {
		goose.AddMigration(Up20201208155805, Down20201208155805)
		initializedMigrations[20201208155805] = true
	}
}

// Up20201208155805 update the maskrcnn workflow template to replace tty with an environment variable
func Up20201208155805(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	return updateWorkflowTemplateManifest(
		filepath.Join("maskrcnn", "20201208155115.yaml"),
		maskRCNNWorkflowTemplateName,
		map[string]string{
			"used-by": "cvat",
		},
	)
}

// Down20201208155805 rolls back the environment variable change
func Down20201208155805(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return updateWorkflowTemplateManifest(
		filepath.Join("maskrcnn", "20201115145814.yaml"),
		maskRCNNWorkflowTemplateName,
		map[string]string{
			"used-by": "cvat",
		},
	)
}
