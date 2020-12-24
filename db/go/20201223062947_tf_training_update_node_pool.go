package migration

import (
	"database/sql"
	"github.com/pressly/goose"
	"path/filepath"
)

func initialize20201223062947() {
	if _, ok := initializedMigrations[20201223062947]; !ok {
		goose.AddMigration(Up20201223062947, Down20201223062947)
		initializedMigrations[20201223062947] = true
	}
}

// Up20201223062947 updates tensorflow-mnist-training with sys.nodepool changes
func Up20201223062947(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	return updateWorkflowTemplateManifest(
		filepath.Join("tensorflow-mnist-training", "20201223062947.yaml"),
		tensorflowWorkflowTemplateName,
		map[string]string{
			"created-by": "system",
			"used-by":    "cvat",
		},
	)
}

// Down20201223062947 undoes sys.nodepool changes
func Down20201223062947(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return updateWorkflowTemplateManifest(
		filepath.Join("tensorflow-mnist-training", "20201223062947.yaml"),
		tensorflowWorkflowTemplateName,
		map[string]string{
			"created-by": "system",
			"used-by":    "cvat",
		},
	)
}
