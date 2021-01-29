package migration

import (
	"database/sql"
	"github.com/pressly/goose"
	"path/filepath"
)

func initialize20210129152427() {
	if _, ok := initializedMigrations[20210129152427]; !ok {
		goose.AddMigration(Up20210129152427, Down20210129152427)
		initializedMigrations[20210129152427] = true
	}
}

// Up20210129152427 migration will add lifecycle hooks to VSCode template.
// These hooks will attempt to export the conda, pip, and vscode packages that are installed,
// to a text file.
// On workspace resume / start, the code then tries to install these packages.
func Up20210129152427(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	return updateWorkspaceTemplateManifest(
		filepath.Join("workspaces", "vscode", "20210129152427.yaml"),
		vscodeWorkspaceTemplateName)
}

// Down20210129152427 removes the lifecycle hooks from VSCode workspace template.
func Down20210129152427(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return updateWorkspaceTemplateManifest(
		filepath.Join("workspaces", "vscode", "20201028145443.yaml"),
		vscodeWorkspaceTemplateName)
}
