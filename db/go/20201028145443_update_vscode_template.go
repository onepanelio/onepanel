package migration

import (
	"database/sql"
	"github.com/pressly/goose"
	"path/filepath"
)

func initialize20201028145443() {
	if _, ok := initializedMigrations[20201028145443]; !ok {
		goose.AddMigration(Up20201028145443, Down20201028145443)
		initializedMigrations[20201028145443] = true
	}
}

// Up20201028145443 migration will add lifecycle hooks to VSCode template.
// These hooks will attempt to export the conda, pip, and vscode packages that are installed,
// to a text file.
// On workspace resume / start, the code then tries to install these packages.
func Up20201028145443(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	return updateWorkspaceTemplateManifest(
		filepath.Join("workspaces", "vscode", "20201028145443.yaml"),
		vscodeWorkspaceTemplateName)
}

// Down20201028145443 removes the lifecycle hooks from VSCode workspace template.
func Down20201028145443(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return updateWorkspaceTemplateManifest(
		filepath.Join("workspaces", "vscode", "20201028145443.yaml"),
		vscodeWorkspaceTemplateName)
}
