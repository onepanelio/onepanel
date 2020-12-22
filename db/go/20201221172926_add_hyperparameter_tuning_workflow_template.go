package migration

import (
	"database/sql"
	"github.com/pressly/goose"
	"path/filepath"
)

const hyperparameterTuningTemplateName = "Hyperparameter Tuning"

func initialize20201221172926() {
	if _, ok := initializedMigrations[20201221172926]; !ok {
		goose.AddMigration(Up20201221172926, Down20201221172926)
		initializedMigrations[20201221172926] = true
	}
}

// Up20201221172926 adds Hyperparameter Tuning Workflow Template
func Up20201221172926(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	return createWorkflowTemplate(
		filepath.Join("hyperparam_tuning", "20201221172926.yaml"),
		hyperparameterTuningTemplateName,
		map[string]string{
			"framework": "pytorch",
			"tuner":     "TPE",
		},
	)
}

// Down20201221172926 archives Hyperparameter Tuning Workflow Template
func Down20201221172926(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return archiveWorkflowTemplate(hyperparameterTuningTemplateName)
}
