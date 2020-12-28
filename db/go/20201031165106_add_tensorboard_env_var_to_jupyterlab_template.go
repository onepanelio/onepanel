package migration

import (
	"database/sql"
	"github.com/pressly/goose"
	"path/filepath"
)

func initialize20201031165106() {
	if _, ok := initializedMigrations[20201031165106]; !ok {
		goose.AddMigration(Up20201031165106, Down20201031165106)
		initializedMigrations[20201031165106] = true
	}
}

// Up20201031165106 updates the jupyterlab workspace to include container lifecycle hooks.
// These hooks will attempt to persist conda, pip, and jupyterlab extensions between pause and shut-down.
func Up20201031165106(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	return updateWorkspaceTemplateManifest(
		filepath.Join("workspaces", "jupyterlab", "20201031165106.yaml"),
		jupyterLabTemplateName)
}

// Down20201031165106 removes the lifecycle hooks from the template.
func Down20201031165106(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return updateWorkspaceTemplateManifest(
		filepath.Join("workspaces", "jupyterlab", "20201028145442.yaml"),
		jupyterLabTemplateName)
}
