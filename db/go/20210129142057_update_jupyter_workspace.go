package migration

import (
	"database/sql"
	"github.com/pressly/goose"
	"path/filepath"
)

func initialize20210129142057() {
	if _, ok := initializedMigrations[20210129142057]; !ok {
		goose.AddMigration(Up20210129142057, Down20210129142057)
		initializedMigrations[20210129142057] = true
	}
}

// Up20210129142057 updates the jupyterlab workspace template
func Up20210129142057(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	return updateWorkspaceTemplateManifest(
		filepath.Join("workspaces", "jupyterlab", "20210129142057.yaml"),
		jupyterLabTemplateName)
}

// Down20210129142057 rolls back the jupyterab workspace template update
func Down20210129142057(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return updateWorkspaceTemplateManifest(
		filepath.Join("workspaces", "jupyterlab", "20201229205644.yaml"),
		jupyterLabTemplateName)
}
