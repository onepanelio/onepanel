package migration

import (
	"database/sql"
	"github.com/pressly/goose"
	"path/filepath"
)

func initialize20201211161117() {
	if _, ok := initializedMigrations[20201211161117]; !ok {
		goose.AddMigration(Up20201211161117, Down20201211161117)
		initializedMigrations[20201211161117] = true
	}
}

// Up20201211161117 updated cvat workspace template with a new ONEPANEL_MAIN_CONTAINER environment variable
func Up20201211161117(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	return updateWorkspaceTemplateManifest(
		filepath.Join("workspaces", "cvat", "20201211161117.yaml"),
		cvatTemplateName)
}

// Down20201211161117 reverts the cvat workspace update
func Down20201211161117(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return updateWorkspaceTemplateManifest(
		filepath.Join("workspaces", "cvat", "20201115133046.yaml"),
		cvatTemplateName)
}
