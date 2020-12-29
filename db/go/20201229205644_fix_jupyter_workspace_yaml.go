package migration

import (
	"database/sql"
	"github.com/pressly/goose"
	"path/filepath"
)

func initialize20201229205644() {
	if _, ok := initializedMigrations[20201229205644]; !ok {
		goose.AddMigration(Up20201229205644, Down20201229205644)
		initializedMigrations[20201229205644] = true
	}
}

func Up20201229205644(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	return updateWorkspaceTemplateManifest(
		filepath.Join("workspaces", "jupyterlab", "20201229205644.yaml"),
		jupyterLabTemplateName)
}

func Down20201229205644(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return updateWorkspaceTemplateManifest(
		filepath.Join("workspaces", "jupyterlab", "20201214133458.yaml"),
		jupyterLabTemplateName)
}
