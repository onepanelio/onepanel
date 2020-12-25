package migration

import (
	"database/sql"
	"github.com/pressly/goose"
	"path/filepath"
)

func initialize20201115145814() {
	if _, ok := initializedMigrations[20201115145814]; !ok {
		goose.AddMigration(Up20201115145814, Down20201115145814)
		initializedMigrations[20201115145814] = true
	}
}

// Up20201115145814 add TensorBoard sidecar to TFODs
func Up20201115145814(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	return updateWorkflowTemplateManifest(
		filepath.Join("workflows", "maskrcnn-training", "20201115145814.yaml"),
		maskRCNNWorkflowTemplateName,
		map[string]string{
			"used-by": "cvat",
		},
	)
}

// Down20201115145814 do nothing
func Down20201115145814(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return nil
}
