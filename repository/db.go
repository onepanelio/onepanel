package repository

import (
	"github.com/jmoiron/sqlx"
)

type DB struct {
	*sqlx.DB
}

func (db *DB) NamedQueryWithStructScan(query string, dest interface{}) (err error) {
	rows, err := db.NamedQuery(query, dest)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		err = rows.StructScan(dest)
		if err != nil {
			return
		}
	}
	err = rows.Err()

	return
}
