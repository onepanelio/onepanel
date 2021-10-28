package migration

import (
	"database/sql"
	"github.com/pressly/goose"
	"path/filepath"
)

func initialize20211028205201() {
	if _, ok := initializedMigrations[20211028205201]; !ok {
		goose.AddMigration(Up20211028205201, Down20211028205201)
		initializedMigrations[20211028205201] = true
	}
}

// Up20211028205201 creates the new cvat 1.6.0 workspace template
func Up20211028205201(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	return createWorkspaceTemplate(
		filepath.Join("workspaces", "cvat_1_6_0", "20211028205201.yaml"),
		"CVAT_1.6.0",
		"Powerful and efficient Computer Vision Annotation Tool (CVAT)")
}

// Down20211028205201 archives the new cvat 1.6.0 workspace template
func Down20211028205201(tx *sql.Tx) error {
	return archiveWorkspaceTemplate("CVAT_1.6.0")
}
