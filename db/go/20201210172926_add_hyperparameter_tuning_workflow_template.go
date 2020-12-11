package migration

import (
	"database/sql"
	"github.com/pressly/goose"
	"path/filepath"
)

const hyperparameterTuningTemplateName = "Hyperparameter Tuning"

func initialize20201210172926() {
	goose.AddMigration(Up20201210172926, Down20201210172926)
}

func Up20201210172926(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	return createWorkflowTemplate(
		filepath.Join("hyperparam_tuning", "20201210172926.yaml"),
		hyperparameterTuningTemplateName,
		map[string]string{
			"framework": "pytorch",
			"tuner":     "TPE",
		},
	)
}

func Down20201210172926(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return archiveWorkflowTemplate(hyperparameterTuningTemplateName)
}
