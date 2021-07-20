package migration

import (
	"database/sql"
	"github.com/pressly/goose"
	"path/filepath"
)

func initialize20210719190719() {
	if _, ok := initializedMigrations[20210719190719]; !ok {
		goose.AddMigration(Up20210719190719, Down20210719190719)
		initializedMigrations[20210719190719] = true
	}
}

// Up20210719190719 updates the workspace templates to use new v1.0.0 of filesyncer
func Up20210719190719(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	if err := updateWorkspaceTemplateManifest(
		filepath.Join("workspaces", "cvat", "20210719190719.yaml"),
		cvatTemplateName); err != nil {
		return err
	}

	if err := updateWorkspaceTemplateManifest(
		filepath.Join("workspaces", "jupyterlab", "20210719190719.yaml"),
		jupyterLabTemplateName); err != nil {
		return err
	}

	if err := updateWorkspaceTemplateManifest(
		filepath.Join("workspaces", "vnc", "20210719190719.yaml"),
		deepLearningDesktopTemplateName); err != nil {
		return err
	}

	return updateWorkspaceTemplateManifest(
		filepath.Join("workspaces", "vscode", "20210719190719.yaml"),
		vscodeWorkspaceTemplateName)
}

// Down20210719190719 rolls back the change to update filesyncer
func Down20210719190719(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
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

	if err := updateWorkspaceTemplateManifest(
		filepath.Join("workspaces", "vnc", "20210414165510.yaml"),
		deepLearningDesktopTemplateName); err != nil {
		return err
	}

	return updateWorkspaceTemplateManifest(
		filepath.Join("workspaces", "vscode", "20210323175655.yaml"),
		vscodeWorkspaceTemplateName)
}
