package migration

import (
	"database/sql"
	"github.com/pressly/goose"
	"path/filepath"
)

func initialize20201208155115() {
	if _, ok := initializedMigrations[20201208155115]; !ok {
		goose.AddMigration(Up20201208155115, Down20201208155115)
		initializedMigrations[20201208155115] = true
	}
}

func Up20201208155115(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	return updateWorkflowTemplateManifest(
		filepath.Join("tfod", "20201208155115.yaml"),
		tensorflowObjectDetectionWorkflowTemplateName,
		map[string]string{
			"used-by": "cvat",
		},
	)
}

func Down20201208155115(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return updateWorkflowTemplateManifest(
		filepath.Join("tfod", "20201130130433.yaml"),
		tensorflowObjectDetectionWorkflowTemplateName,
		map[string]string{
			"used-by": "cvat",
		},
	)
}
