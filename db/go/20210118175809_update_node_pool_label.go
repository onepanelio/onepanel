package migration

import (
	"database/sql"
	"github.com/pressly/goose"
	"path/filepath"
)

func initialize20210118175809() {
	if _, ok := initializedMigrations[20210118175809]; !ok {
		goose.AddMigration(Up20210118175809, Down20210118175809)
		initializedMigrations[20210118175809] = true
	}
}

// Up20210118175809 updates workflows so that the nodePoolSelector label is based on k8s config
func Up20210118175809(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	if err := updateWorkflowTemplateManifest(
		filepath.Join("workflows", "hyperparameter-tuning", "20210118175809.yaml"),
		tensorflowObjectDetectionWorkflowTemplateName,
		map[string]string{
			"framework":  "tensorflow",
			"tuner":      "TPE",
			"created-by": "system",
		},
	); err != nil {
		return err
	}

	if err := updateWorkflowTemplateManifest(
		filepath.Join("workflows", "maskrcnn-training", "20210118175809.yaml"),
		tensorflowObjectDetectionWorkflowTemplateName,
		map[string]string{
			"created-by": "system",
			"used-by":    "cvat",
		},
	); err != nil {
		return err
	}

	if err := updateWorkflowTemplateManifest(
		filepath.Join("workflows", "pytorch-mnist-training", "20210118175809.yaml"),
		tensorflowObjectDetectionWorkflowTemplateName,
		map[string]string{
			"created-by": "system",
		},
	); err != nil {
		return err
	}

	if err := updateWorkflowTemplateManifest(
		filepath.Join("workflows", "tensorflow-mnist-training", "20210118175809.yaml"),
		tensorflowObjectDetectionWorkflowTemplateName,
		map[string]string{
			"created-by": "system",
			"framework":  "tensorflow",
			"used-by":    "cvat",
		},
	); err != nil {
		return err
	}

	return updateWorkflowTemplateManifest(
		filepath.Join("workflows", "tf-object-detection-training", "20210118175809.yaml"),
		tensorflowObjectDetectionWorkflowTemplateName,
		map[string]string{
			"created-by": "system",
			"used-by":    "cvat",
		},
	)
}

// Down20210118175809 reverts the migration
func Down20210118175809(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	if err := updateWorkflowTemplateManifest(
		filepath.Join("workflows", "tf-object-detection-training", "20201223202929.yaml"),
		tensorflowObjectDetectionWorkflowTemplateName,
		map[string]string{
			"created-by": "system",
			"used-by":    "cvat",
		},
	); err != nil {
		return err
	}

	if err := updateWorkflowTemplateManifest(
		filepath.Join("workflows", "tensorflow-mnist-training", "20201223062947.yaml"),
		tensorflowObjectDetectionWorkflowTemplateName,
		map[string]string{
			"created-by": "system",
			"framework":  "tensorflow",
			"used-by":    "cvat",
		},
	); err != nil {
		return err
	}

	if err := updateWorkflowTemplateManifest(
		filepath.Join("workflows", "pytorch-mnist-training", "20201221194344.yaml"),
		tensorflowObjectDetectionWorkflowTemplateName,
		map[string]string{
			"created-by": "system",
		},
	); err != nil {
		return err
	}

	if err := updateWorkflowTemplateManifest(
		filepath.Join("workflows", "maskrcnn-training", "20201221195937.yaml"),
		tensorflowObjectDetectionWorkflowTemplateName,
		map[string]string{
			"created-by": "system",
			"used-by":    "cvat",
		},
	); err != nil {
		return err
	}

	return updateWorkflowTemplateManifest(
		filepath.Join("workflows", "hyperparameter-tuning", "20201225172926.yaml"),
		tensorflowObjectDetectionWorkflowTemplateName,
		map[string]string{
			"framework":  "tensorflow",
			"tuner":      "TPE",
			"created-by": "system",
		},
	)
}
