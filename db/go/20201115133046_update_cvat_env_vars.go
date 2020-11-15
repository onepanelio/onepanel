package migration

import (
	"database/sql"
	"github.com/pressly/goose"
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
	return updateWorkspaceTemplateManifest("20201115133046_cvat.yaml", cvatTemplateName)
}

//Down20201115133046 reverts latest environment variable updates
func Down20201115133046(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return updateWorkspaceTemplateManifest("20201113094916_cvat.yaml", cvatTemplateName)
}
