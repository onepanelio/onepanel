package migration

import (
	"database/sql"
	"github.com/pressly/goose"
	"path/filepath"
)

func initialize20201221195937() {
	if _, ok := initializedMigrations[20201221195937]; !ok {
		goose.AddMigration(Up20201221195937, Down20201221195937)
		initializedMigrations[20201221195937] = true
	}
}

// Up20201221195937 updates maskrcnn with sys.nodepool changes
func Up20201221195937(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	return updateWorkflowTemplateManifest(
		filepath.Join("maskrcnn", "20201221195937.yaml"),
		maskRCNNWorkflowTemplateName,
		map[string]string{
			"created-by": "system",
			"used-by":    "cvat",
		},
	)
}

// Down20201221195937 undoes the sys.nodepool changes
func Down20201221195937(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return updateWorkflowTemplateManifest(
		filepath.Join("maskrcnn", "20201208155115.yaml"),
		maskRCNNWorkflowTemplateName,
		map[string]string{
			"created-by": "system",
			"used-by":    "cvat",
		},
	)
}
