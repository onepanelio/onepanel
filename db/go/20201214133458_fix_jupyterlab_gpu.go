package migration

import (
	"database/sql"
	"github.com/pressly/goose"
	"path/filepath"
)

func initialize20201214133458() {
	if _, ok := initializedMigrations[20201214133458]; !ok {
		goose.AddMigration(Up20201214133458, Down20201214133458)
	}
}

// Up20201214133458 fixes an issue where LD_LIBRARY_PATH is not present for JupyterLab
func Up20201214133458(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	return updateWorkspaceTemplateManifest(
		filepath.Join("jupyterlab", "20201214133458.yaml"),
		cvatTemplateName)
}

// Down20201214133458 undoes the change
func Down20201214133458(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return updateWorkspaceTemplateManifest(
		filepath.Join("jupyterlab", "20201031165106.yaml"),
		cvatTemplateName)
}
