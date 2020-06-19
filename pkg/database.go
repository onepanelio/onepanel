package v1

import (
	sq "github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
)

type DB struct {
	sqlx.DB
}

func NewDB(db *sqlx.DB) *DB {
	return &DB{
		*db,
	}
}

// Selectx performs a select query using a squirrel SelectBuilder as an argument.
//
// This is a convenience wrapper. Any errors from squirrel are returned as is.
func (db *DB) Selectx(dest interface{}, builder sq.SelectBuilder) error {
	sql, args, err := builder.ToSql()
	if err != nil {
		return err
	}

	return db.Select(dest, sql, args...)
}
