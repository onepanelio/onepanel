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

// Up20201208155115 update the tfod workflow template to replace tty with an environment variable
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

// Down20201208155115 rolls back the environment variable change
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
