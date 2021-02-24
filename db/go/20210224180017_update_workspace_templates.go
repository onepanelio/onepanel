package migration

import (
	"database/sql"
	"github.com/pressly/goose"
	"path/filepath"
)

func initialize20210224180017() {
	if _, ok := initializedMigrations[20210224180017]; !ok {
		goose.AddMigration(Up20210224180017, Down20210224180017)
		initializedMigrations[20210224180017] = true
	}
}

// Up20210224180017 Updates workspace templates with the latest filesyncer image
func Up20210224180017(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	if err := updateWorkspaceTemplateManifest(
		filepath.Join("workspaces", "cvat", "20210224180017.yaml"),
		cvatTemplateName); err != nil {
		return err
	}

	if err := updateWorkspaceTemplateManifest(
		filepath.Join("workspaces", "jupyterlab", "20210224180017.yaml"),
		jupyterLabTemplateName); err != nil {
		return err
	}

	return updateWorkspaceTemplateManifest(
		filepath.Join("workspaces", "vscode", "20210224180017.yaml"),
		vscodeWorkspaceTemplateName)
}

// Down20210224180017 Rolls back the filesyncer image updates
func Down20210224180017(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	if err := updateWorkspaceTemplateManifest(
		filepath.Join("workspaces", "vscode", "20210224180017.yaml"),
		vscodeWorkspaceTemplateName); err != nil {
		return err
	}

	if err := updateWorkspaceTemplateManifest(
		filepath.Join("workspaces", "jupyterlab", "20210224180017.yaml"),
		jupyterLabTemplateName); err != nil {
		return err
	}

	return updateWorkspaceTemplateManifest(
		filepath.Join("workspaces", "cvat", "20210224180017.yaml"),
		cvatTemplateName)
}
