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

// Up20201209124226 updates the tensorflow workflow
func Up20201209124226(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	return updateWorkflowTemplateManifest(
		filepath.Join("tensorflow-mnist-training", "20201209124226.yaml"),
		tensorflowWorkflowTemplateName,
		map[string]string{
			"framework": "tensorflow",
		},
	)
}

// Down20201209124226 rolls back the tensorflow workflow
func Down20201209124226(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return updateWorkflowTemplateManifest(
		filepath.Join("tensorflow-mnist-training", "20200605090535.yaml"),
		tensorflowWorkflowTemplateName,
		map[string]string{
			"framework": "tensorflow",
		},
	)
}
