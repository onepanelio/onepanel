package migration

import (
	"database/sql"
	"github.com/pressly/goose"
	"path/filepath"
)

func initialize20210329171739() {
	if _, ok := initializedMigrations[20210329171739]; !ok {
		goose.AddMigration(Up20210329171739, Down20210329171739)
		initializedMigrations[20210329171739] = true
	}
}

// Up20210329171739 updates workspaces to use new images
func Up20210329171739(tx *sql.Tx) error {
	// This code is executed when the migration is applied.

	if err := updateWorkspaceTemplateManifest(
		filepath.Join("workspaces", "cvat", "20210323175655.yaml"),
		cvatTemplateName); err != nil {
		return err
	}

	if err := updateWorkspaceTemplateManifest(
		filepath.Join("workspaces", "jupyterlab", "20210323175655.yaml"),
		jupyterLabTemplateName); err != nil {
		return err
	}

	return updateWorkspaceTemplateManifest(
		filepath.Join("workspaces", "vscode", "20210323175655.yaml"),
		vscodeWorkspaceTemplateName)
}

// Down20210329171739 rolls back image updates for workspaces
func Down20210329171739(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
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
