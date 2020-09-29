package migration

import (
	"database/sql"
	uid2 "github.com/onepanelio/core/pkg/util/uid"
	"github.com/pressly/goose"
)

func initialize20200922103448() {
	if _, ok := initializedMigrations[20200922103448]; !ok {
		goose.AddMigration(Up20200922103448, Down20200922103448)
		initializedMigrations[20200922103448] = true
	}
}

// Up20200922103448 adds a description to the jupyterlab workspace template
func Up20200922103448(tx *sql.Tx) error {
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

	if _, ok := migrationsRan[20200922103448]; ok {
		return nil
	}

	namespaces, err := client.ListOnepanelEnabledNamespaces()
	if err != nil {
		return err
	}

	uid, err := uid2.GenerateUID(jupyterLabTemplateName, 30)
	if err != nil {
		return err
	}

	for _, namespace := range namespaces {
		workspaceTemplate, err := client.GetWorkspaceTemplate(namespace.Name, uid, 0)
		if err != nil {
			return err
		}
		if workspaceTemplate == nil {
			continue
		}

		// Adding description
		workspaceTemplate.Description = "Interactive development environment for notebooks"

		if _, err := client.UpdateWorkspaceTemplate(namespace.Name, workspaceTemplate); err != nil {
			return err
		}
	}

	return nil
}

// Down20200922103448 does nothing
func Down20200922103448(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return nil
}
