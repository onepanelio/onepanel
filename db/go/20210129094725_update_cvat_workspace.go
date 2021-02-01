package migration

import (
	"database/sql"
	"github.com/pressly/goose"
	"path/filepath"
)

func initialize20210129134326() {
	if _, ok := initializedMigrations[20210129134326]; !ok {
		goose.AddMigration(Up20210129134326, Down20210129134326)
		initializedMigrations[20210129134326] = true
	}
}

//Up20210129134326 updates CVAT to latest image
func Up20210129134326(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	return updateWorkspaceTemplateManifest(
		filepath.Join("workspaces", "cvat", "20210129134326.yaml"),
		cvatTemplateName)
}

//Down20210129134326 reverts to previous CVAT image
func Down20210129134326(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return updateWorkspaceTemplateManifest(
		filepath.Join("workspaces", "cvat", "20210107094725.yaml"),
		cvatTemplateName)
}
