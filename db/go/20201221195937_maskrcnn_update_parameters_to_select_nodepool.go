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
