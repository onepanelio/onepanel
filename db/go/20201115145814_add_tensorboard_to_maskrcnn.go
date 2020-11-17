package migration

import (
	"database/sql"
	"github.com/pressly/goose"
)

func initialize20201115145814() {
	if _, ok := initializedMigrations[20201115145814]; !ok {
		goose.AddMigration(Up20201115145814, Down20201115145814)
		initializedMigrations[20201115145814] = true
	}
}

//Up20201115145814 add TensorBoard sidecar to TFODs
func Up20201115145814(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	return updateWorkflowTemplateManifest(
		"20201115145814_maskrcnn.yaml",
		maskRCNNWorkflowTemplateName,
		map[string]string{
			"used-by": "cvat",
		},
	)
}

//Down20201115145814 do nothing
func Down20201115145814(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return nil
}
