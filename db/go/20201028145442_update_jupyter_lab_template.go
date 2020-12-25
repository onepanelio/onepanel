package migration

import (
	"database/sql"
	"github.com/pressly/goose"
	"path/filepath"
)

func initialize20201028145442() {
	if _, ok := initializedMigrations[20201028145442]; !ok {
		goose.AddMigration(Up20201028145442, Down20201028145442)
		initializedMigrations[20201028145442] = true
	}
}

// Up20201028145442 updates the jupyterlab workspace to include container lifecycle hooks.
// These hooks will attempt to persist conda, pip, and jupyterlab extensions between pause and shut-down.
func Up20201028145442(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	return updateWorkspaceTemplateManifest(
		filepath.Join("workspaces", "jupyterlab", "20201028145442.yaml"),
		jupyterLabTemplateName)
}

// Down20201028145442 removes the lifecycle hooks from the template.
func Down20201028145442(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return nil
}
