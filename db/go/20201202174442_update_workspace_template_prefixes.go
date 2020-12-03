package migration

import (
	"database/sql"
	"errors"
	"fmt"
	v1 "github.com/onepanelio/core/pkg"
	"github.com/onepanelio/core/pkg/util/label"
	"github.com/pressly/goose"
)

func initialize20201202174442() {
	if _, ok := initializedMigrations[20201202174442]; !ok {
		goose.AddMigration(Up20201202174442, Down20201202174442)
		initializedMigrations[20201202174442] = true
	}
}

func Up20201202174442(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	client, err := getClient()
	if err != nil {
		return err
	}
	defer client.DB.Close()

	rows, err := tx.Query(`
		SELECT namespace, name, uid
		FROM workflow_templates
		WHERE id IN (
    		SELECT workflow_template_id
    		FROM workspace_templates
		) AND uid NOT LIKE 'sys-%'
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	workflowTemplates := make([]v1.WorkflowTemplate, 0)
	for rows.Next() {
		namespace := ""
		name := ""
		uid := ""
		if err := rows.Scan(&namespace, &name, &uid); err != nil {
			return err
		}

		workflowTemplates = append(workflowTemplates, v1.WorkflowTemplate{
			Namespace: namespace,
			Name:      name,
			UID:       uid,
		})
	}

	for _, workflowTemplate := range workflowTemplates {
		labelSelect := fmt.Sprintf("%v=%v", label.WorkflowTemplateUid, workflowTemplate.UID)
		workflowTemplates, err := client.ArgoprojV1alpha1().WorkflowTemplates(workflowTemplate.Namespace).List(v1.ListOptions{
			LabelSelector: labelSelect,
		})
		if err != nil {
			return err
		}

		templates := workflowTemplates.Items
		if templates.Len() == 0 {
			return fmt.Errorf("argo workflowtemplate not found for label: %v=%v", label.WorkflowTemplateUid, workflowTemplate.UID)
		}

		for _, argoTemplate := range templates {
			workflowTemplate.Name = v1.ConvertToSystemName(workflowTemplate.Name)
			if err := workflowTemplate.GenerateUID(workflowTemplate.Name); err != nil {
				return err
			}
			argoTemplate.Labels[label.WorkflowTemplateUid] = workflowTemplate.UID

			if _, err := client.ArgoprojV1alpha1().WorkflowTemplates(workflowTemplate.Namespace).Update(&argoTemplate); err != nil {
				return err
			}
		}
	}

	_, err = tx.Exec(`
		UPDATE workflow_templates
		SET name = CONCAT('sys-', name),
			uid = CONCAT('sys-', uid)
		WHERE id IN (
			SELECT workflow_template_id
			FROM workspace_templates
		)`)
	if err != nil {
		return err
	}

	return err
}

func Down20201202174442(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	client, err := getClient()
	if err != nil {
		return err
	}
	defer client.DB.Close()

	_, err = tx.Exec(`
		UPDATE workflow_templates
		SET name = 	substring(name from 5),
			uid  = 	substring(uid from 5)
		WHERE id IN (
			SELECT workflow_template_id
			FROM workspace_templates
		) AND name LIKE 'sys-%'`)
	if err != nil {
		return err
	}

	rows, err := tx.Query(`
		SELECT namespace, name, uid
		FROM workflow_templates
		WHERE id IN (
    		SELECT workflow_template_id
    		FROM workspace_templates
		) AND uid LIKE 'sys-%'
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	workflowTemplates := make([]v1.WorkflowTemplate, 0)
	for rows.Next() {
		namespace := ""
		name := ""
		uid := ""
		if err := rows.Scan(&namespace, &name, &uid); err != nil {
			return err
		}

		workflowTemplates = append(workflowTemplates, v1.WorkflowTemplate{
			Namespace: namespace,
			Name:      name,
			UID:       uid,
		})
	}

	for _, workflowTemplate := range workflowTemplates {
		labelSelect := fmt.Sprintf("%v=%v", label.WorkflowTemplateUid, workflowTemplate.UID)
		workflowTemplates, err := client.ArgoprojV1alpha1().WorkflowTemplates(workflowTemplate.Namespace).List(v1.ListOptions{
			LabelSelector: labelSelect,
		})
		if err != nil {
			return err
		}

		templates := workflowTemplates.Items
		if templates.Len() == 0 {
			return errors.New("not found")
		}

		for _, argoTemplate := range templates {
			// Remove sys- prefix
			workflowTemplate.Name = workflowTemplate.Name[4:]
			workflowTemplate.Name = v1.ConvertToSystemName(workflowTemplate.Name)
			if err := workflowTemplate.GenerateUID(workflowTemplate.Name); err != nil {
				return err
			}
			argoTemplate.Labels[label.WorkflowTemplateUid] = workflowTemplate.UID

			if _, err := client.ArgoprojV1alpha1().WorkflowTemplates(workflowTemplate.Namespace).Update(&argoTemplate); err != nil {
				return err
			}
		}
	}

	return err
}
