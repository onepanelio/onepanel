package v1

import (
	"fmt"
	sq "github.com/Masterminds/squirrel"
	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/onepanelio/core/pkg/util/label"
	"github.com/onepanelio/core/pkg/util/mapping"
	"github.com/onepanelio/core/pkg/util/types"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
)

// SelectLabelsQuery represents the options available to filter a select labels query
type SelectLabelsQuery struct {
	Table     string
	Alias     string
	Namespace string
	KeyLike   string
	Skip      []string
}

// SkipKeysFromString parses keys encoded in a string and returns an array of keys
// The separator is ";"
func SkipKeysFromString(keys string) []string {
	results := make([]string, 0)
	for _, key := range strings.Split(keys, ";") {
		if key == "" {
			continue
		}

		results = append(results, key)
	}

	return results
}

// SelectLabels returns a SelectBuilder that selects key, value columns from the criteria specified in query
func SelectLabels(query *SelectLabelsQuery) sq.SelectBuilder {
	// Sample query
	// SELECT DISTINCT labels.*
	//	FROM workflow_executions w,
	//	jsonb_each_text(w.labels) labels
	// WHERE labels.key LIKE 'ca%'
	// AND labels.key NOT IN ('catdog')
	// AND namespace = 'onepanel'
	// AND labels != 'null'::jsonb

	fromTable := fmt.Sprintf("%s %s", query.Table, query.Alias)
	fromJsonb := fmt.Sprintf("jsonb_each_text(%s.labels) labels", query.Alias)

	bld := sb.Select("key", "value").
		Distinct().
		From(fromTable + ", " + fromJsonb).
		Where("labels != 'null'::jsonb")

	if query.Namespace != "" {
		bld = bld.Where(sq.Eq{query.Alias + ".namespace": query.Namespace})
	}
	if query.KeyLike != "" {
		bld = bld.Where(sq.Like{"labels.key": query.KeyLike})
	}
	if len(query.Skip) != 0 {
		bld = bld.Where(sq.NotEq{"labels.key": query.Skip})
	}

	return bld
}

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

// ListAvailableLabels lists the labels available for the resource specified by the query
func (c *Client) ListAvailableLabels(query *SelectLabelsQuery) (result []*Label, err error) {
	selectLabelsBuilder := SelectLabels(query)

	// Don't select labels from Terminated workspaces.
	if query.Table == "workspaces" {
		selectLabelsBuilder = selectLabelsBuilder.Where(sq.NotEq{
			"l.phase": "Terminated",
		})
	}

	err = c.Selectx(&result, selectLabelsBuilder)

	return
}

func (c *Client) AddLabels(namespace, resource, uid string, keyValues map[string]string) error {
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

// ReplaceLabelsUsingKnownID updates the k8s resource labels for the given resource/uid
// deprecated
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
