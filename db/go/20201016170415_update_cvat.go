package migration

import (
	"database/sql"
	"github.com/pressly/goose"
	"path/filepath"
)

func initialize20201016170415() {
	if _, ok := initializedMigrations[20201016170415]; !ok {
		goose.AddMigration(Up20201016170415, Down20201016170415)
		initializedMigrations[20201016170415] = true
	}
}

// Up20201016170415 updates cvat to a new version
func Up20201016170415(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	return updateWorkspaceTemplateManifest(
		filepath.Join("cvat", "20201016170415.yaml"),
		cvatTemplateName)
}

// Down20201016170415 does nothing
func Down20201016170415(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return nil
}
