package v1

import (
	"fmt"
	sq "github.com/Masterminds/squirrel"
	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/hashicorp/go-uuid"
	"github.com/onepanelio/core/pkg/util/label"
	"github.com/onepanelio/core/pkg/util/mapping"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log"
)

func (c *Client) ListLabels(resource string, uid string) (labels []*Label, err error) {
	sb := sb.Select("l.id", "l.created_at", "l.key", "l.value", "l.resource", "l.resource_id").
		From("labels l").
		Where(sq.Eq{
			"resource": resource,
		}).
		OrderBy("l.created_at")

	switch resource {
	case TypeWorkflowTemplate:
		sb = sb.Join("workflow_templates wt ON wt.id = l.resource_id").
			Where(sq.Eq{"wt.uid": uid})
	case TypeWorkflowTemplateVersion:
		sb = sb.Join("workflow_template_versions wtv ON wtv.id = l.resource_id").
			Where(sq.Eq{"wtv.uid": uid})
	case TypeWorkflowExecution:
		sb = sb.Join("workflow_executions we ON we.id = l.resource_id").
			Where(sq.Eq{"we.uid": uid})
	case TypeCronWorkflow:
		sb = sb.Join("cron_workflows cw ON cw.id = l.resource_id").
			Where(sq.Eq{"cw.uid": uid})
	}

	query, args, sqlErr := sb.ToSql()
	if sqlErr != nil {
		err = sqlErr
		return
	}

	err = c.DB.Select(&labels, query, args...)

	return
}

func GetResourceIdBuilder(resource, uid string) (*sq.SelectBuilder, error) {
	tableName := TypeToTableName(resource)
	if tableName == "" {
		return nil, fmt.Errorf("unknown resources '%v'", resource)
	}

	sb := sb.Select("id").
		From(tableName).
		Where(sq.Eq{
			"uid": uid,
		})

	return &sb, nil
}

func (c *Client) AddLabels(namespace, resource, uid string, keyValues map[string]string) error {
	tx, err := c.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	tableName := TypeToTableName(resource)
	if tableName == "" {
		return fmt.Errorf("unknown resources '%v'", resource)
	}

	resourceId := uint64(0)
	err = sb.Select("id").
		From(tableName).
		Where(sq.Eq{
			"uid": uid,
		}).
		RunWith(tx).
		QueryRow().
		Scan(&resourceId)
	if err != nil {
		return err
	}

	_, err = sb.Delete("labels").
		Where(sq.Eq{
			"key":         mapping.PluckKeysStr(keyValues),
			"resource":    resource,
			"resource_id": resourceId,
		}).RunWith(tx).
		Exec()
	if err != nil {
		return err
	}

	_, err = c.InsertLabelsBuilder(resource, resourceId, keyValues).
		RunWith(tx).
		Exec()
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	source, meta, err := c.GetK8sLabelResource(namespace, resource, uid)
	if err != nil {
		return err
	}

	if meta.Labels == nil {
		meta.Labels = make(map[string]string)
	}
	label.MergeLabelsPrefix(meta.Labels, keyValues, label.TagPrefix)
	if err := c.UpdateK8sLabelResource(namespace, resource, source); err != nil {
		return err
	}

	return nil
}

func (c *Client) ReplaceLabels(namespace, resource, uid string, keyValues map[string]string) error {
	tx, err := c.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	tableName := TypeToTableName(resource)
	if tableName == "" {
		return fmt.Errorf("unknown resources '%v'", resource)
	}

	resourceId := uint64(0)
	err = sb.Select("id").
		From(tableName).
		Where(sq.Eq{
			"uid": uid,
		}).
		RunWith(tx).
		QueryRow().
		Scan(&resourceId)
	if err != nil {
		return err
	}

	_, err = sb.Delete("labels").
		Where(sq.Eq{
			"resource":    resource,
			"resource_id": resourceId,
		}).RunWith(tx).
		Exec()
	if err != nil {
		return err
	}

	_, err = c.InsertLabelsBuilder(resource, resourceId, keyValues).
		RunWith(tx).
		Exec()
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	source, meta, err := c.GetK8sLabelResource(namespace, resource, uid)
	if err != nil {
		return err
	}

	if meta.Labels == nil {
		meta.Labels = make(map[string]string)
	}
	label.MergeLabelsPrefix(meta.Labels, keyValues, label.TagPrefix)
	if err := c.UpdateK8sLabelResource(namespace, resource, source); err != nil {
		return err
	}

	return nil
}

func (c *Client) DeleteLabels(namespace, resource, uid string, keyValues map[string]string) error {
	tx, err := c.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	tableName := TypeToTableName(resource)
	if tableName == "" {
		return fmt.Errorf("unknown resources '%v'", resource)
	}

	resourceId := uint64(0)
	err = sb.Select("id").
		From(tableName).
		Where(sq.Eq{
			"uid": uid,
		}).
		RunWith(tx).
		QueryRow().
		Scan(&resourceId)
	if err != nil {
		return err
	}

	_, err = sb.Delete("labels").
		Where(sq.Eq{
			"key":         mapping.PluckKeysStr(keyValues),
			"resource":    resource,
			"resource_id": resourceId,
		}).RunWith(tx).
		Exec()
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	source, meta, err := c.GetK8sLabelResource(namespace, resource, uid)
	if err != nil {
		return err
	}

	if meta.Labels == nil {
		meta.Labels = make(map[string]string)
	}

	toDelete := make([]string, 0)
	for key := range keyValues {
		toDelete = append(toDelete, key)
	}

	label.Delete(meta.Labels, toDelete...)
	if err := c.UpdateK8sLabelResource(namespace, resource, source); err != nil {
		return err
	}

	return nil
}

func (c *Client) InsertLabelsBuilder(resource string, resourceId uint64, keyValues map[string]string) sq.InsertBuilder {
	sb := sb.Insert("labels").
		Columns("uid", "resource", "resource_id", "key", "value")

	for key, value := range keyValues {
		uid, err := uuid.GenerateUUID()
		if err != nil {
			log.Fatal("unable to generate uuid")
		}
		sb = sb.Values(uid, resource, resourceId, key, value)
	}

	return sb
}

func (c *Client) GetDbLabels(resource string, ids ...uint64) (labels []*Label, err error) {
	if len(ids) == 0 {
		return make([]*Label, 0), nil
	}

	tx, err := c.DB.Begin()
	if err != nil {
		return nil, err
	}

	whereIn := "resource_id IN (?"
	for i := range ids {
		if i == 0 {
			continue
		}

		whereIn += ",?"
	}
	whereIn += ")"

	defer tx.Rollback()

	query, args, err := sb.Select("id", "key", "value", "resource", "resource_id").
		From("labels").
		Where(whereIn, ids).
		Where(sq.Eq{
			"resource": resource,
		}).
		OrderBy("key").
		ToSql()

	if err != nil {
		return nil, err
	}

	allArgs := make([]interface{}, 0)
	for _, arg := range args[0].([]uint64) {
		allArgs = append(allArgs, arg)
	}
	allArgs = append(allArgs, args[1])

	err = c.DB.Select(&labels, query, allArgs...)
	if err != nil {
		return nil, err
	}

	return
}

func (c *Client) GetDbLabelsMapped(resource string, ids ...uint64) (result map[uint64]map[string]string, err error) {
	dbLabels, err := c.GetDbLabels(resource, ids...)
	if err != nil {
		return
	}

	result = make(map[uint64]map[string]string)
	for _, dbLabel := range dbLabels {
		_, ok := result[dbLabel.ResourceId]
		if !ok {
			result[dbLabel.ResourceId] = make(map[string]string)
		}
		result[dbLabel.ResourceId][dbLabel.Key] = dbLabel.Value
	}

	return
}

func (c *Client) GetK8sLabelResource(namespace, resource, uid string) (source interface{}, result *v1.ObjectMeta, err error) {
	switch resource {
	case TypeWorkflowTemplateVersion:
		return c.getK8sLabelResourceWorkflowTemplateVersion(namespace, uid)
	case TypeWorkflowExecution:
		return c.getK8sLabelResourceWorkflowExecution(namespace, uid)
	case TypeCronWorkflow:
		return c.getK8sLabelResourceCronWorkflow(namespace, uid)
	}

	return nil, nil, nil
}

func (c *Client) getK8sLabelResourceWorkflowTemplateVersion(namespace, uid string) (source interface{}, result *v1.ObjectMeta, err error) {
	labelSelect := fmt.Sprintf("%v=%v", label.WorkflowTemplateVersionUid, uid)

	workflowTemplates, err := c.ArgoprojV1alpha1().WorkflowTemplates(namespace).List(v1.ListOptions{
		LabelSelector: labelSelect,
	})
	if err != nil {
		return nil, nil, err
	}

	if workflowTemplates.Items.Len() != 1 {
		return nil, nil, fmt.Errorf("no argo resource found")
	}

	item := workflowTemplates.Items[0]

	return item, &item.ObjectMeta, nil
}

func (c *Client) getK8sLabelResourceWorkflowExecution(namespace, uid string) (source interface{}, result *v1.ObjectMeta, err error) {
	labelSelect := fmt.Sprintf("%v=%v", label.WorkflowUid, uid)

	workflows, err := c.ArgoprojV1alpha1().Workflows(namespace).List(v1.ListOptions{
		LabelSelector: labelSelect,
	})
	if err != nil {
		return nil, nil, err
	}

	if workflows.Items.Len() != 1 {
		return nil, nil, fmt.Errorf("no argo resource found")
	}

	item := workflows.Items[0]

	return item, &item.ObjectMeta, nil
}

func (c *Client) getK8sLabelResourceCronWorkflow(namespace, uid string) (source interface{}, result *v1.ObjectMeta, err error) {
	labelSelect := fmt.Sprintf("%v=%v", label.CronWorkflowUid, uid)

	cronWorkflows, err := c.ArgoprojV1alpha1().CronWorkflows(namespace).List(v1.ListOptions{
		LabelSelector: labelSelect,
	})
	if err != nil {
		return nil, nil, err
	}

	if len(cronWorkflows.Items) != 1 {
		return nil, nil, fmt.Errorf("no argo resource found")
	}

	item := cronWorkflows.Items[0]

	return item, &item.ObjectMeta, nil
}

func (c *Client) UpdateK8sLabelResource(namespace, resource string, obj interface{}) error {
	if resource == TypeWorkflowTemplateVersion {
		workflowTemplate, ok := obj.(v1alpha1.WorkflowTemplate)
		if !ok {
			return fmt.Errorf("unable to convert object to WorkflowTemplate")
		}

		if _, err := c.ArgoprojV1alpha1().WorkflowTemplates(namespace).Update(&workflowTemplate); err != nil {
			return err
		}
	} else if resource == TypeWorkflowExecution {
		workflowExecution, ok := obj.(v1alpha1.Workflow)
		if !ok {
			return fmt.Errorf("unable to convert object to workflow")
		}

		if _, err := c.ArgoprojV1alpha1().Workflows(namespace).Update(&workflowExecution); err != nil {
			return err
		}
	} else if resource == TypeCronWorkflow {
		cronWorkflow, ok := obj.(v1alpha1.CronWorkflow)
		if !ok {
			return fmt.Errorf("unable to convert object to cron workflow")
		}

		if _, err := c.ArgoprojV1alpha1().CronWorkflows(namespace).Update(&cronWorkflow); err != nil {
			return err
		}
	}

	return nil
}
