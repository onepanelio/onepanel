package migration

import (
	"github.com/jmoiron/sqlx"
	v1 "github.com/onepanelio/core/pkg"
)

// Initialize sets up the go migrations.
func Initialize() {
	initialize20200525160514()
	initialize20200528140124()
	initialize20200605090509()
	initialize20200605090535()
	initialize20200626113635()
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
