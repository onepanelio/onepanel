package migration

import (
	"database/sql"
	"github.com/pressly/goose"
	"path/filepath"
)

func initialize20201223202929() {
	if _, ok := initializedMigrations[20201223202929]; !ok {
		goose.AddMigration(Up20201223202929, Down20201223202929)
		initializedMigrations[20201223202929] = true
	}
}

// Up20201223202929 updates tfod with sys.nodepool
func Up20201223202929(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	return updateWorkflowTemplateManifest(
		filepath.Join("tfod", "20201223202929.yaml"),
		tensorflowObjectDetectionWorkflowTemplateName,
		map[string]string{
			"created-by": "system",
			"used-by":    "cvat",
		},
	)
}

// Down20201223202929 undoes the sys.nodepool changes
func Down20201223202929(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return updateWorkflowTemplateManifest(
		filepath.Join("tfod", "20201208155115.yaml"),
		tensorflowObjectDetectionWorkflowTemplateName,
		map[string]string{
			"created-by": "system",
			"used-by":    "cvat",
		},
	)
}
