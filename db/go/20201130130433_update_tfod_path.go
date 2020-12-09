package migration

import (
	"database/sql"
	"github.com/pressly/goose"
	"path/filepath"
)

func initialize20201130130433() {
	if _, ok := initializedMigrations[20201130130433]; !ok {
		goose.AddMigration(Up20201130130433, Down20201130130433)
		initializedMigrations[20201130130433] = true
	}
}

// Up20201130130433 remove namespace to resolve checkpoint path issue
func Up20201130130433(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	return updateWorkflowTemplateManifest(
		filepath.Join("tfod", "20201130130433.yaml"),
		tensorflowObjectDetectionWorkflowTemplateName,
		map[string]string{
			"used-by": "cvat",
		},
	)
}

// Down20201130130433 do nothing
func Down20201130130433(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return updateWorkflowTemplateManifest(
		filepath.Join("tfod", "20201115134934.yaml"),
		tensorflowObjectDetectionWorkflowTemplateName,
		map[string]string{
			"used-by": "cvat",
		},
	)
}
