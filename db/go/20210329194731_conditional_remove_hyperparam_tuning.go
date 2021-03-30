package migration

import (
	"database/sql"
	uid2 "github.com/onepanelio/core/pkg/util/uid"
	"github.com/pressly/goose"
	"path/filepath"
)

func initialize20210329194731() {
	if _, ok := initializedMigrations[20210329194731]; !ok {
		goose.AddMigration(Up20210329194731, Down20210329194731)
		initializedMigrations[20210329194731] = true
	}
}

func init() {
	goose.AddMigration(Up20210329194731, Down20210329194731)
}

// Up20210329194731 removes the hyperparameter-tuning workflow if there are no executions
func Up20210329194731(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	client, err := getClient()
	if err != nil {
		return err
	}
	defer client.DB.Close()

	namespaces, err := client.ListOnepanelEnabledNamespaces()
	if err != nil {
		return err
	}

	uid, err := uid2.GenerateUID("hyperparameter-tuning", 30)
	if err != nil {
		return err
	}

	for _, namespace := range namespaces {
		workflowTemplate, err := client.GetWorkflowTemplateRaw(namespace.Name, uid)
		if err != nil {
			return err
		}

		if workflowTemplate == nil {
			continue
		}

		workflowExecutionsCount, err := client.CountWorkflowExecutionsForWorkflowTemplate(workflowTemplate.ID)
		if err != nil {
			return err
		}

		cronWorkflowsCount, err := client.CountCronWorkflows(namespace.Name, uid)
		if err != nil {
			return err
		}

		// Archive the template if we have no resources associated with it
		if workflowExecutionsCount == 0 && cronWorkflowsCount == 0 {
			if _, err := client.ArchiveWorkflowTemplate(namespace.Name, uid); err != nil {
				return err
			}
		}
	}

	return nil
}

// Down20210329194731 returns the hyperparameter-tuning workflow if it was deleted
func Down20210329194731(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	client, err := getClient()
	if err != nil {
		return err
	}
	defer client.DB.Close()

	namespaces, err := client.ListOnepanelEnabledNamespaces()
	if err != nil {
		return err
	}

	uid, err := uid2.GenerateUID("hyperparameter-tuning", 30)
	if err != nil {
		return err
	}

	for _, namespace := range namespaces {
		workflowTemplate, err := client.GetWorkflowTemplateRaw(namespace.Name, uid)
		if err != nil {
			return err
		}

		if workflowTemplate == nil {
			err := createWorkflowTemplate(
				filepath.Join("workflows", "hyperparameter-tuning", "20210118175809.yaml"),
				hyperparameterTuningTemplateName,
				map[string]string{
					"framework":  "tensorflow",
					"tuner":      "TPE",
					"created-by": "system",
				},
			)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
