package migration

import (
	sq "github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
	v1 "github.com/onepanelio/core/pkg"
)

// initializedMigrations is used to keep track of which migrations have been initialized.
// if they are initialzed more than once, goose panics.
var initializedMigrations = make(map[int]bool)

// Initialize sets up the go migrations.
func Initialize() {
	initialize20200525160514()
	initialize20200528140124()
	initialize20200605090509()
	initialize20200605090535()
	initialize20200626113635()
	initialize20200704151301()
}

func getClient() (*v1.Client, error) {
	kubeConfig := v1.NewConfig()
	client, err := v1.NewClient(kubeConfig, nil, nil)
	if err != nil {
		return nil, err
	}
	config, err := client.GetSystemConfig()
	if err != nil {
		return nil, err
	}

	dbDriverName, dbDataSourceName := config.DatabaseConnection()
	client.DB = v1.NewDB(sqlx.MustConnect(dbDriverName, dbDataSourceName))

	return client, nil
}

// getRanSQLMigrations returns a map where each key is a sql migration version ran.
func getRanSQLMigrations(client *v1.Client) (map[uint64]bool, error) {
	sb := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query := sb.Select("version_id").
		From("goose_db_version")

	versions := make([]uint64, 0)
	if err := client.Selectx(&versions, query); err != nil {
		return nil, err
	}

	result := make(map[uint64]bool)
	for _, version := range versions {
		result[version] = true
	}

	return result, nil
}
