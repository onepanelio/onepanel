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

// Up20201221194344 updates pytorch_training with the sys.nodepool changes
func Up20201221194344(tx *sql.Tx) error {
	return updateWorkflowTemplateManifest(
		filepath.Join("workflows", "pytorch-mnist-training", "20201221194344.yaml"),
		pytorchMnistWorkflowTemplateName,
		map[string]string{
			"created-by": "system",
		},
	)
}

// Down20201221194344 undoes the sys-nodepool changes
func Down20201221194344(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return updateWorkflowTemplateManifest(
		filepath.Join("workflows", "pytorch-mnist-training", "20200605090509.yaml"),
		pytorchMnistWorkflowTemplateName,
		map[string]string{
			"created-by": "system",
		},
	)
}
