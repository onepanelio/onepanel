package repository

import (
	"database/sql"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type DB struct {
	*sqlx.DB
}

func NewDB(driverName, dataSourceName string) *DB {
	db := sqlx.MustConnect(driverName, dataSourceName)

	return &DB{DB: db}
}

func (db *DB) BaseConnection() *sql.DB {
	return db.DB.DB
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
