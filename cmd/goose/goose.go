// This is custom goose binary to support .go migration files in ./db dir

package main

import (
	"flag"
	"github.com/jmoiron/sqlx"
	migrations "github.com/onepanelio/core/db/go"
	v1 "github.com/onepanelio/core/pkg"
	"log"
	"os"
	"path/filepath"

	"github.com/pressly/goose"
)

var (
	flags = flag.NewFlagSet("goose", flag.ExitOnError)
	dir   = flags.String("dir", ".", "directory with migration files")
)

func main() {
	flags.Parse(os.Args[1:])
	args := flags.Args()

	if len(args) < 1 {
		flags.Usage()
		return
	}

	kubeConfig := v1.NewConfig()
	client, err := v1.NewClient(kubeConfig, nil, nil)
	if err != nil {
		log.Fatalf("Failed to connect to Kubernetes cluster: %v", err)
	}
	config, err := client.GetSystemConfig()
	if err != nil {
		log.Fatalf("Failed to get system config: %v", err)
	}

	dbDriverName, dbDataSourceName := config.DatabaseConnection()
	db := sqlx.MustConnect(dbDriverName, dbDataSourceName)

	command := args[0]

	arguments := []string{}
	if len(args) > 2 {
		arguments = append(arguments, args[2:]...)
	}

	goose.SetTableName("goose_db_version")
	if err := goose.Run(command, db.DB, filepath.Join(*dir, "sql"), arguments...); err != nil {
		log.Fatalf("Failed to run database sql migrations: %v %v", command, err)
	}

	goose.SetTableName("goose_db_go_version")
	migrations.Initialize()
	if err := goose.Run(command, db.DB, filepath.Join(*dir, "go"), arguments...); err != nil {
		log.Fatalf("Failed to run database go migrations: %v %v", command, err)
	}
}
