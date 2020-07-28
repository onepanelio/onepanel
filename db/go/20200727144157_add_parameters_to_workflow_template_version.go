package migration

import (
	"database/sql"
	v1 "github.com/onepanelio/core/pkg"
	"github.com/onepanelio/core/pkg/util/pagination"
	"github.com/pressly/goose"
)

func initialize20200727144157() {
	if _, ok := initializedMigrations[20200727144157]; !ok {
		goose.AddMigration(Up20200727144157, Down20200727144157)
		initializedMigrations[20200727144157] = true
	}
}

func Up20200727144157(tx *sql.Tx) error {
	// This code is executed when the migration is applied.

	client, err := getClient()
	if err != nil {
		return err
	}

	migrationsRan, err := getRanSQLMigrations(client)
	if err != nil {
		return err
	}

	if _, ok := migrationsRan[20200727144157]; ok {
		return nil
	}

	namespaces, err := client.ListOnepanelEnabledNamespaces()
	if err != nil {
		return err
	}

	paginator := pagination.NewRequest(0, 1000)
	wtsResults := -1
	for _, namespace := range namespaces {
		for wtsResults != 0 {
			wts, err := client.ListWorkflowTemplates(namespace.Name, &paginator)
			if err != nil {
				return err
			}
			wtsResults = len(wts)

			for _, wt := range wts {
				wtvs, err := client.ListWorkflowTemplateVersionsDB(namespace.Name, wt.UID)
				if err != nil {
					return err
				}
				for _, wtv := range wtvs {
					params, err := v1.ParseParametersFromManifest(wtv.WorkflowTemplate.GetManifestBytes())
					if err != nil {
						return err
					}
					wtv.Parameters = params
					err = client.UpdateWorkflowTemplateVersionDB(namespace.Name, wtv)
				}

			}
		}
	}

	return nil
}

func Down20200727144157(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return nil
}
