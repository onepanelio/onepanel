package migrations

import (
	"database/sql"
	"github.com/pressly/goose"
)


func initialize20201130130725() {
	if _, ok := initializedMigrations[20201130130725]; !ok {
		goose.AddMigration(Up20201130130725, 20201130130725)
		initializedMigrations[20201130130725] = true
	}
}


func Up20201130130725(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	return updateWorkspaceTemplateManifest("20201130130725_cvat.yaml", cvatTemplateName)
}

//Down20201113094916 updates CVAT back to previous python-sdk version of 0.14.0
func Down20201130130725(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return updateWorkspaceTemplateManifest("20201130130725_cvat.yaml", cvatTemplateName)
}


