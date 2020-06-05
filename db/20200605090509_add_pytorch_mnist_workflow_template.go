package migration

import (
	"database/sql"
	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(Up20200605090509, Down20200605090509)
}

func Up20200605090509(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	return nil
}

func Down20200605090509(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return nil
}
