package migration

import (
	"database/sql"
	"github.com/pressly/goose"
)

func initialize20200814160856() {
	if _, ok := initializedMigrations[20200814160856]; !ok {
		goose.AddMigration(Up20200814160856, Down20200814160856)
		initializedMigrations[20200814160856] = true
	}
}

func Up20200814160856(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	// This code is executed when the migration is applied.
	client, err := getClient()
	if err != nil {
		return err
	}
	defer client.DB.Close()

	migrationsRan, err := getRanSQLMigrations(client)
	if err != nil {
		return err
	}

	if _, ok := migrationsRan[20200814160856]; ok {
		return nil
	}

	namespaces, err := client.ListOnepanelEnabledNamespaces()
	if err != nil {
		return err
	}

	for _, namespace := range namespaces {
		workspaceTemplate, err := client.GetWorkspaceTemplate(namespace.Name, "jupyterlab", 0)
		if err != nil {
			return err
		}

		workspaceTemplate.Description = "Interactive development environment for notebooks"
		if _, err := client.UpdateWorkspaceTemplate(namespace.Name, workspaceTemplate); err != nil {
			return err
		}
	}

	return nil
}

func Down20200814160856(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return nil
}
