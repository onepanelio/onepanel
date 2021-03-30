package migration

import (
	"database/sql"
	"github.com/pressly/goose"
	"path/filepath"
)

func initialize20210323175655() {
	if _, ok := initializedMigrations[20210323175655]; !ok {
		goose.AddMigration(Up20210323175655, Down20210323175655)
		initializedMigrations[20210323175655] = true
	}
}

// Up20210323175655 update workflows to support new PNS mode
func Up20210323175655(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	if err := updateWorkflowTemplateManifest(
		filepath.Join("workflows", "pytorch-mnist-training", "20210323175655.yaml"),
		pytorchWorkflowTemplateName,
		map[string]string{
			"created-by": "system",
			"framework":  "pytorch",
		}); err != nil {
		return err
	}

	return updateWorkflowTemplateManifest(
		filepath.Join("workflows", "tensorflow-mnist-training", "20210323175655.yaml"),
		tensorflowWorkflowTemplateName,
		map[string]string{
			"created-by": "system",
			"framework":  "tensorflow",
		})
}

// Down20210323175655 reverts updating workflows to support PNS
func Down20210323175655(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	if err := updateWorkflowTemplateManifest(
		filepath.Join("workflows", "tensorflow-mnist-training", "20210118175809.yaml"),
		tensorflowWorkflowTemplateName,
		map[string]string{
			"created-by": "system",
			"framework":  "tensorflow",
		}); err != nil {
		return err
	}

	return updateWorkflowTemplateManifest(
		filepath.Join("workflows", "pytorch-mnist-training", "20210118175809.yaml"),
		pytorchWorkflowTemplateName,
		map[string]string{
			"created-by": "system",
			"framework":  "pytorch",
		})
}
