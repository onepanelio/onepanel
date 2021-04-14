package migration

import (
	"database/sql"
	"github.com/pressly/goose"
	"path/filepath"
)

var deepLearningDesktopTemplateName = "Deep Learning Desktop"

func initialize20210414165510() {
	if _, ok := initializedMigrations[20210414165510]; !ok {
		goose.AddMigration(Up20210414165510, Down20210414165510)
		initializedMigrations[20210414165510] = true
	}
}

// Up20210414165510 creates the Deep Learning Desktop Workspace Template
func Up20210414165510(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	return createWorkspaceTemplate(
		filepath.Join("workspaces", "vnc", "20210414165510.yaml"),
		deepLearningDesktopTemplateName,
		"Deep learning desktop with VNC")
}

// Down20210414165510 removes the  Deep Learning Desktop Workspace Template
func Down20210414165510(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return archiveWorkspaceTemplate(deepLearningDesktopTemplateName)
}
