package migration

import (
	"database/sql"
	"github.com/pressly/goose"
	"path/filepath"
)

func initialize20210107094725() {
	if _, ok := initializedMigrations[20210107094725]; !ok {
		goose.AddMigration(Up20210107094725, Down20210107094725)
		initializedMigrations[20210107094725] = true
	}
}

//Up20210107094725 updates CVAT to latest image
func Up20210107094725(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	return updateWorkspaceTemplateManifest(
		filepath.Join("workspaces", "cvat", "20210107094725.yaml"),
		cvatTemplateName)
}

//Down20210107094725 reverts to previous CVAT image
func Down20210107094725(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return updateWorkspaceTemplateManifest(
		filepath.Join("workspaces", "cvat", "20201211161117.yaml"),
		cvatTemplateName)
}
