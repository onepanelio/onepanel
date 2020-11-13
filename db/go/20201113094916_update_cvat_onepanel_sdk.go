package migration

import (
	"database/sql"
	"github.com/pressly/goose"
)

func initialize20201113094916() {
	if _, ok := initializedMigrations[20201113094916]; !ok {
		goose.AddMigration(Up20201113094916, Down20201113094916)
		initializedMigrations[20201113094916] = true
	}
}

func Up20201113094916(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	return updateWorkspaceTemplateManifest("20201113094916_cvat.yaml", cvatTemplateName)
}

func Down20201113094916(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return updateWorkspaceTemplateManifest("20201102104048_cvat.yaml", cvatTemplateName)
}
