package migration

import (
	"database/sql"
	v1 "github.com/onepanelio/core/pkg"
	"github.com/onepanelio/core/pkg/util/pagination"
	"github.com/pressly/goose"
)

func initialize20200728190804() {
	if _, ok := initializedMigrations[20200728190804]; !ok {
		goose.AddMigration(Up20200728190804, Down20200728190804)
		initializedMigrations[20200704151301] = true
	}
}

// Up20200728190804 updates labels so that we keep track of WorkflowTemplate labels.
// Before, only workflow template versions had labels, but to speed up some queries, we now cache the latest version's labels
// for workflow templates themselves.
func Up20200728190804(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	client, err := getClient()
	if err != nil {
		return err
	}

	migrationsRan, err := getRanSQLMigrations(client)
	if err != nil {
		return err
	}

	if _, ok := migrationsRan[20200728190804]; ok {
		return nil
	}

	namespaces, err := client.ListOnepanelEnabledNamespaces()
	if err != nil {
		return err
	}

	for _, namespace := range namespaces {
		page := int32(1)
		paginator := pagination.NewRequest(page, 500)

		resultCount := -1
		for resultCount != 0 {
			workflowTemplates, err := client.ListAllWorkflowTemplates(namespace.Name, &paginator, nil)
			if err != nil {
				return err
			}

			for _, workflowTemplate := range workflowTemplates {
				if err := client.ReplaceLabelsUsingKnownID(namespace.Name, v1.TypeWorkflowTemplate, workflowTemplate.ID, workflowTemplate.UID, workflowTemplate.Labels); err != nil {
					return err
				}
			}

			resultCount = len(workflowTemplates)
		}
	}

	return nil
}

// Down20200728190804 rolls down the migration by deleting all workflow template labels, since they did not exist before this
func Down20200728190804(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.

	client, err := getClient()
	if err != nil {
		return err
	}

	return client.DeleteResourceLabels(tx, v1.TypeWorkflowTemplate)
}
