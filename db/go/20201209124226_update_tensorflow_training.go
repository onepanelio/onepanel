package migration

import (
	"database/sql"
	"github.com/pressly/goose"
	"path/filepath"
)

func initialize20201209124226() {
	if _, ok := initializedMigrations[20201209124226]; !ok {
		goose.AddMigration(Up20201209124226, Down20201209124226)
		initializedMigrations[20201209124226] = true
	}
}

func Up20201209124226(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	return updateWorkflowTemplateManifest(
		filepath.Join("tf_training", "20201209124226.yaml"),
		tensorflowWorkflowTemplateName,
		map[string]string{
			"framework": "tensorflow",
		},
	)
}

func Down20201209124226(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return updateWorkflowTemplateManifest(
		filepath.Join("tf_training", "20200605090535.yaml"),
		tensorflowWorkflowTemplateName,
		map[string]string{
			"framework": "tensorflow",
		},
	)
}
