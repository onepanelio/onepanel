package v1

import (
	"database/sql"
	"fmt"
	sq "github.com/Masterminds/squirrel"
	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/onepanelio/core/pkg/util/label"
	"github.com/onepanelio/core/pkg/util/mapping"
	"github.com/onepanelio/core/pkg/util/types"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *Client) ListLabels(resource string, uid string) (labels []*Label, err error) {
	sb := sb.Select("labels").
		From(TypeToTableName(resource))

	switch resource {
	case TypeWorkflowTemplate:
		sb = sb.Where(sq.Eq{"uid": uid})
	case TypeWorkflowExecution:
		sb = sb.Where(sq.Eq{"uid": uid})
	case TypeCronWorkflow:
		sb = sb.Where(sq.Eq{"uid": uid})
	case TypeWorkspace:
		sb = sb.Where(sq.And{
			sq.Eq{"uid": uid},
			sq.NotEq{"phase": "Terminated"},
		})
	default:
		return nil, fmt.Errorf("unsupported label resource %v", resource)
	}

	result := types.JSONLabels{}
	err = c.DB.Getx(&result, sb)
	if err != nil {
		return
	}

	for key, value := range result {
		newLabel := &Label{
			Key:      key,
			Value:    value,
			Resource: resource,
		}

		labels = append(labels, newLabel)
	}

	return
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

	var whereCondition interface{} = nil
	if resource == TypeWorkspace {
		whereCondition = sq.And{
			sq.Eq{"uid": uid},
			sq.NotEq{"phase": "Terminated"},
		}
	} else if resource == TypeWorkspaceTemplate || resource == TypeWorkflowExecution {
		whereCondition =
			sq.Eq{
				"uid":         uid,
				"is_archived": false,
			}
	}

	_, err = sb.Update(tableName).
		SetMap(sq.Eq{
			"labels": types.JSONLabels(keyValues),
		}).
		Where(whereCondition).
		RunWith(tx).
		Exec()
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return c.ReplaceLabelsUsingKnownID(namespace, resource, uid, keyValues)
}

func (c *Client) ReplaceLabelsUsingKnownID(namespace, resource string, uid string, keyValues map[string]string) error {
	source, meta, err := c.GetK8sLabelResource(namespace, resource, uid)
	if err != nil {
		return err
	}

	if meta != nil {
		if meta.Labels == nil {
			meta.Labels = make(map[string]string)
		}
		label.MergeLabelsPrefix(meta.Labels, keyValues, label.TagPrefix)
		if err := c.UpdateK8sLabelResource(namespace, resource, source); err != nil {
			return err
		}
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

// DeleteResourceLabels deletes all of the labels for a specific resource, like workflow templates.
// NOTE: this does NOT delete k8s labels, and is only meant to be used for special cases.
func (c *Client) DeleteResourceLabels(runner sq.BaseRunner, resource string) error {
	tableName := TypeToTableName(resource)
	if tableName == "" {
		return fmt.Errorf("unknown resources '%v'", resource)
	}

	_, err := sb.Delete("labels").
		Where(sq.Eq{
			"resource": resource,
		}).
		RunWith(runner).
		Exec()

	return err
}

func (c *Client) InsertLabelsBuilder(resource string, resourceID uint64, keyValues map[string]string) sq.InsertBuilder {
	sb := sb.Insert("labels").
		Columns("resource", "resource_id", "key", "value")

	for key, value := range keyValues {
		sb = sb.Values(resource, resourceID, key, value)
	}

	return sb
}

// InsertLabelsRunner inserts the labels for the resource into the db using the provided runner.
// If no labels are provided, does nothing and returns nil, nil.
func (c *Client) InsertLabelsRunner(runner sq.BaseRunner, resource string, resourceID uint64, keyValues map[string]string) (sql.Result, error) {
	if len(keyValues) == 0 {
		return nil, nil
	}

	return c.InsertLabelsBuilder(resource, resourceID, keyValues).
		RunWith(runner).
		Exec()
}

// InsertLabels inserts the labels for the resource into the db using the client's DB.
// If no labels are provided, does nothing and returns nil, nil.
func (c *Client) InsertLabels(resource string, resourceID uint64, keyValues map[string]string) (sql.Result, error) {
	return c.InsertLabelsRunner(c.DB, resource, resourceID, keyValues)
}

func (c *Client) GetDbLabels(resource string, ids ...uint64) (labels []*Label, err error) {
	if len(ids) == 0 {
		return make([]*Label, 0), nil
	}

	tx, err := c.DB.Begin()
	if err != nil {
		return nil, err
	}

	defer tx.Rollback()

	query, args, err := sb.Select("id", "created_at", "key", "value", "resource", "resource_id").
		From("labels").
		Where(sq.Eq{
			"resource_id": ids,
			"resource":    resource,
		}).
		OrderBy("key").
		ToSql()

	if err != nil {
		return nil, err
	}

	err = c.DB.Select(&labels, query, args...)
	if err != nil {
		return nil, err
	}

	return
}

// GetDBLabelsMapped returns a map where the key is the id of the resource
// and the value is the labels as a map[string]string
func (c *Client) GetDBLabelsMapped(resource string, ids ...uint64) (result map[uint64]map[string]string, err error) {
	dbLabels, err := c.GetDbLabels(resource, ids...)
	if err != nil {
		return
	}

	result = make(map[uint64]map[string]string)
	for _, dbLabel := range dbLabels {
		_, ok := result[dbLabel.ResourceID]
		if !ok {
			result[dbLabel.ResourceID] = make(map[string]string)
		}
		result[dbLabel.ResourceID][dbLabel.Key] = dbLabel.Value
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
	case TypeWorkspaceTemplateVersion:
		return c.getK8sLabelResourceWorkspaceTemplate(namespace, uid)
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
	workflow, err := c.ArgoprojV1alpha1().Workflows(namespace).Get(uid, v1.GetOptions{})
	if err != nil {
		return nil, nil, err
	}

	return workflow, &workflow.ObjectMeta, nil
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

func (c *Client) getK8sLabelResourceWorkspaceTemplate(namespace, uid string) (source interface{}, result *v1.ObjectMeta, err error) {
	labelSelect := fmt.Sprintf("%v=%v", label.WorkspaceTemplateVersionUid, uid)

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
		workflowExecution, ok := obj.(*v1alpha1.Workflow)
		if !ok {
			return fmt.Errorf("unable to convert object to workflow")
		}

		if _, err := c.ArgoprojV1alpha1().Workflows(namespace).Update(workflowExecution); err != nil {
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
	} else if resource == TypeWorkspaceTemplateVersion {
		workflowTemplate, ok := obj.(v1alpha1.WorkflowTemplate)
		if !ok {
			return fmt.Errorf("unable to convert object to WorkflowTemplate")
		}

		if _, err := c.ArgoprojV1alpha1().WorkflowTemplates(namespace).Update(&workflowTemplate); err != nil {
			return err
		}
	}

	return nil
}
