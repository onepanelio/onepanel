package migration

import (
	"fmt"
	"github.com/jmoiron/sqlx"
	v1 "github.com/onepanelio/core/pkg"
)

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

	databaseDataSourceName := fmt.Sprintf("host=%v user=%v password=%v dbname=%v sslmode=disable",
		config["databaseHost"], config["databaseUsername"], config["databasePassword"], config["databaseName"])
	client.DB = v1.NewDB(sqlx.MustConnect(config["databaseDriverName"], databaseDataSourceName))

	return client, nil
}
