package migration

import (
	"database/sql"
	"errors"
	"fmt"
	v1 "github.com/onepanelio/core/pkg"
	"github.com/onepanelio/core/pkg/util/label"
	"github.com/pressly/goose"
)

func initialize20201214133459() {
	if _, ok := initializedMigrations[20201214133459]; !ok {
		goose.AddMigration(Up20201214133459, Down20201214133459)
		initializedMigrations[20201214133459] = true
	}
}

// Up20201214133459 updates the argo template labels
func Up20201214133459(tx *sql.Tx) error {
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

		workflowTemplate.Name = v1.ConvertToSystemName(workflowTemplate.Name)
		if len(workflowTemplate.Name) > 30 {
			workflowTemplate.Name = workflowTemplate.Name[:30]
		}
		if err := workflowTemplate.GenerateUID(workflowTemplate.Name); err != nil {
			return err
		}

		for _, argoTemplate := range templates {
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

// Down20201214133459 reverts the argo template label updates
func Down20201214133459(tx *sql.Tx) error {
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

		// Remove sys- prefix
		workflowTemplate.Name = workflowTemplate.Name[4:]
		if err := workflowTemplate.GenerateUID(workflowTemplate.Name); err != nil {
			return err
		}

		for _, argoTemplate := range templates {
			argoTemplate.Labels[label.WorkflowTemplateUid] = workflowTemplate.UID

			if _, err := client.ArgoprojV1alpha1().WorkflowTemplates(workflowTemplate.Namespace).Update(&argoTemplate); err != nil {
				return err
			}
		}
	}

	return err
}
