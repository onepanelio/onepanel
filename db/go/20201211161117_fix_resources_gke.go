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

func Up20201211161117(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	return updateWorkspaceTemplateManifest(
		filepath.Join("cvat", "20201211161117.yaml"),
		cvatTemplateName)
}

func Down20201211161117(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return updateWorkspaceTemplateManifest(
		filepath.Join("cvat", "20201115133046.yaml"),
		cvatTemplateName)
}
