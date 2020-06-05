package migration

import (
	"database/sql"
	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(Up20200605090535, Down20200605090535)
}

func Up20200605090535(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	return nil
}

func Down20200605090535(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return nil
}
