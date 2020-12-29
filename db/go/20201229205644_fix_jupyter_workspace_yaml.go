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

// Up20201229205644 updates the jupyterlab workspace template
func Up20201229205644(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	return updateWorkspaceTemplateManifest(
		filepath.Join("workspaces", "jupyterlab", "20201229205644.yaml"),
		jupyterLabTemplateName)
}

// Down20201229205644 rolls back the jupyterab workspace template update
func Down20201229205644(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return updateWorkspaceTemplateManifest(
		filepath.Join("workspaces", "jupyterlab", "20201214133458.yaml"),
		jupyterLabTemplateName)
}
