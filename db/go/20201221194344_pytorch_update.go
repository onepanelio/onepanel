package migration

import (
	"database/sql"
	"github.com/pressly/goose"
	"path/filepath"
)

func initialize20201221194344() {
	if _, ok := initializedMigrations[20201221194344]; !ok {
		goose.AddMigration(Up20201221194344, Down20201221194344)
		initializedMigrations[20201221194344] = true
	}
}

func Up20201221194344(tx *sql.Tx) error {
	return updateWorkflowTemplateManifest(
		filepath.Join("pytorch_training", "20201221194344.yaml"),
		pytorchMnistWorkflowTemplateName,
		map[string]string{
			"created-by": "system",
		},
	)
}

func Down20201221194344(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return updateWorkflowTemplateManifest(
		filepath.Join("pytorch_training", "20200605090509.yaml"),
		pytorchMnistWorkflowTemplateName,
		map[string]string{
			"created-by": "system",
		},
	)
}
