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

//Up20201113094916 updates CVAT with python-sdk 0.15.0
//Of note, this replaces the authentication request endpoint.
func Up20201113094916(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	return updateWorkspaceTemplateManifest("20201113094916_cvat.yaml", cvatTemplateName)
}

//Down20201113094916 updates CVAT back to previous python-sdk version of 0.14.0
func Down20201113094916(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return updateWorkspaceTemplateManifest("20201102104048_cvat.yaml", cvatTemplateName)
}
