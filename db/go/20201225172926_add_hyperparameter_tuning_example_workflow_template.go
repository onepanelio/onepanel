package migration

import (
	"database/sql"
	"github.com/pressly/goose"
	"path/filepath"
)

const hyperparameterTuningTemplateName = "Hyperparameter Tuning Example"

func initialize20201225172926() {
	if _, ok := initializedMigrations[20201225172926]; !ok {
		goose.AddMigration(Up20201225172926, Down20201225172926)
		initializedMigrations[20201225172926] = true
	}
}

// Up20201225172926 adds Hyperparameter Tuning Workflow Template
func Up20201225172926(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	return createWorkflowTemplate(
		filepath.Join("workflows", "hyperparameter-tuning", "20201225172926.yaml"),
		hyperparameterTuningTemplateName,
		map[string]string{
			"framework":  "pytorch",
			"tuner":      "TPE",
			"created-by": "system",
		},
	)
}

// Down20201225172926 archives Hyperparameter Tuning Workflow Template
func Down20201225172926(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return archiveWorkflowTemplate(hyperparameterTuningTemplateName)
}
