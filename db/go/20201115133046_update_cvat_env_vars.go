package migration

import (
	"database/sql"
	"github.com/pressly/goose"
	"path/filepath"
)

func initialize20201115133046() {
	if _, ok := initializedMigrations[20201115133046]; !ok {
		goose.AddMigration(Up20201115133046, Down20201115133046)
		initializedMigrations[20201115133046] = true
	}
}

//Up20201115133046 updates CVAT environment variables
func Up20201115133046(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	return updateWorkspaceTemplateManifest(
		filepath.Join("workspaces", "cvat", "20201115133046.yaml"),
		cvatTemplateName)
}

//Down20201115133046 reverts latest environment variable updates
func Down20201115133046(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return updateWorkspaceTemplateManifest(
		filepath.Join("workspaces", "cvat", "20201113094916.yaml"),
		cvatTemplateName)
}
