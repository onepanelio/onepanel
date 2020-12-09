package migration

import (
	"database/sql"
	"github.com/pressly/goose"
	"path/filepath"
)

func initialize20201102104048() {
	if _, ok := initializedMigrations[20201102104048]; !ok {
		goose.AddMigration(Up20201102104048, Down20201102104048)
		initializedMigrations[20201102104048] = true
	}
}

// Up20201102104048 updates CVAT to use less volumes.
// Through the use of environment variables, various CVAT data directories
// are placed under one path, and that path is on one volume.
func Up20201102104048(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	return updateWorkspaceTemplateManifest(
		filepath.Join("cvat", "20201102104048.yaml"),
		cvatTemplateName)
}

// Down20201102104048 reverts CVAT back to original amount of volumes.
func Down20201102104048(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return updateWorkspaceTemplateManifest(
		filepath.Join("cvat", "20201016170415.yaml"),
		cvatTemplateName)
}
