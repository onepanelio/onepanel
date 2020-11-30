package migrations

import (
	"database/sql"
	"github.com/pressly/goose"
)

func initialize20201130130433() {
	if _, ok := initializedMigrations[20201130130433]; !ok {
		goose.AddMigration(Up20201130130433, Down20201130130433)
		initializedMigrations[20201130130433] = true
	}
}

//Up20201130130433 remove namespace to resolve checkpoint path issue
func Up20201130130433(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	return updateWorkflowTemplateManifest(
		"20201130130433_tfod.yaml",
		tfodWorkflowTemplateName,
		map[string]string{
			"used-by": "cvat",
		},
	)
}

//Down20201130130433 do nothing
func Down20201130130433(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return nil
}
