package v1

import (
	"fmt"
	sq "github.com/Masterminds/squirrel"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	argojson "github.com/argoproj/pkg/json"
	"github.com/onepanelio/core/pkg/util"
	"github.com/onepanelio/core/pkg/util/label"
	"github.com/onepanelio/core/pkg/util/pagination"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"regexp"
	"strings"
)

func (c *Client) UpdateCronWorkflow(namespace string, name string, cronWorkflow *CronWorkflow) (*CronWorkflow, error) {
	err := c.cronWorkflowSelectBuilderNoColumns(namespace, cronWorkflow.WorkflowExecution.WorkflowTemplate.UID).
		Columns("cw.id").
		RunWith(c.DB).
		QueryRow().
		Scan(&cronWorkflow.ID)
	if err != nil {
		return nil, err
	}

	workflow := cronWorkflow.WorkflowExecution
	workflowTemplate, err := c.GetWorkflowTemplate(namespace, workflow.WorkflowTemplate.UID, workflow.WorkflowTemplate.Version)
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace":    namespace,
			"CronWorkflow": cronWorkflow,
			"Error":        err.Error(),
		}).Error("Error with getting workflow template.")
		return nil, util.NewUserError(codes.NotFound, "Error with getting workflow template.")
	}

	// TODO: Need to pull system parameters from k8s config/secret here, example: HOST
	opts := &WorkflowExecutionOptions{}
	re, _ := regexp.Compile(`[^a-zA-Z0-9-]{1,}`)
	opts.GenerateName = strings.ToLower(re.ReplaceAllString(workflowTemplate.Name, `-`)) + "-"
	for _, param := range workflow.Parameters {
		opts.Parameters = append(opts.Parameters, Parameter{
			Name:  param.Name,
			Value: param.Value,
		})
	}

	if err := workflowTemplate.UpdateManifestParameters(workflow.Parameters); err != nil {
		return nil, err
	}

	rawCronManifest := cronWorkflow.Manifest
	workflowTemplateManifest := workflowTemplate.GetManifestBytes()

	if err := cronWorkflow.AddToManifestSpec("workflowSpec", string(workflowTemplateManifest)); err != nil {
		return nil, err
	}

	if opts.Labels == nil {
		opts.Labels = &map[string]string{}
	}
	(*opts.Labels)[workflowTemplateUIDLabelKey] = workflowTemplate.UID
	(*opts.Labels)[workflowTemplateVersionLabelKey] = fmt.Sprint(workflowTemplate.Version)
	var argoCronWorkflow wfv1.CronWorkflow
	var argoCronWorkflowSpec wfv1.CronWorkflowSpec
	if err := argojson.UnmarshalStrict([]byte(rawCronManifest), &argoCronWorkflowSpec); err != nil {
		return nil, err
	}
	argoCronWorkflow.Spec = argoCronWorkflowSpec
	manifestBytes, err := workflowTemplate.GetWorkflowManifestBytes()
	if err != nil {
		return nil, err
	}

	workflows, err := UnmarshalWorkflows(manifestBytes, true)
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace":    namespace,
			"CronWorkflow": cronWorkflow,
			"Error":        err.Error(),
		}).Error("Error parsing workflow.")
		return nil, err
	}
	if len(workflows) != 1 {
		return nil, fmt.Errorf("more than one workflow in spec")
	}

	wf := workflows[0]
	argoCronWorkflow.Spec.WorkflowSpec = wf.Spec
	_, err = c.updateCronWorkflow(namespace, name, &workflowTemplate.ID, &wf, &argoCronWorkflow, opts)
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace":    namespace,
			"CronWorkflow": cronWorkflow,
			"Error":        err.Error(),
		}).Error("Error parsing workflow.")
		return nil, err
	}
	cronWorkflow.WorkflowExecution.WorkflowTemplate = workflowTemplate
	// Manifests could get big, don't return them in this case.
	cronWorkflow.WorkflowExecution.WorkflowTemplate.Manifest = ""

	tx, err := c.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	_, err = sb.Update("cron_workflows").
		SetMap(sq.Eq{
			"manifest": cronWorkflow.Manifest,
		}).Where(sq.Eq{
		"id": cronWorkflow.ID,
	}).
		RunWith(tx).
		Exec()
	if err != nil {
		return nil, err
	}

	// delete all labels then replace
	_, err = sb.Delete("labels").
		Where(sq.Eq{
			"resource":    TypeCronWorkflow,
			"resource_id": cronWorkflow.ID,
		}).RunWith(tx).
		Exec()
	if err != nil {
		return nil, err
	}

	if len(cronWorkflow.Labels) > 0 {
		_, err = c.InsertLabelsBuilder(TypeCronWorkflow, cronWorkflow.ID, cronWorkflow.Labels).
			RunWith(tx).
			Exec()
		if err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return cronWorkflow, nil
}

func (c *Client) CreateCronWorkflow(namespace string, cronWorkflow *CronWorkflow) (*CronWorkflow, error) {
	workflow := cronWorkflow.WorkflowExecution
	workflowTemplate, err := c.GetWorkflowTemplate(namespace, workflow.WorkflowTemplate.UID, workflow.WorkflowTemplate.Version)
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace":    namespace,
			"CronWorkflow": cronWorkflow,
			"Error":        err.Error(),
		}).Error("Error with getting workflow template.")
		return nil, util.NewUserError(codes.NotFound, "Error with getting workflow template.")
	}

	//// TODO: Need to pull system parameters from k8s config/secret here, example: HOST
	opts := &WorkflowExecutionOptions{}
	re, _ := regexp.Compile(`[^a-zA-Z0-9-]{1,}`)
	opts.GenerateName = strings.ToLower(re.ReplaceAllString(workflowTemplate.Name, `-`)) + "-"
	for _, param := range workflow.Parameters {
		opts.Parameters = append(opts.Parameters, Parameter{
			Name:  param.Name,
			Value: param.Value,
		})
	}

	if err := workflowTemplate.UpdateManifestParameters(workflow.Parameters); err != nil {
		return nil, err
	}

	rawCronManifest := cronWorkflow.Manifest
	workflowTemplateManifest := workflowTemplate.GetManifestBytes()

	if err := cronWorkflow.AddToManifestSpec("workflowSpec", string(workflowTemplateManifest)); err != nil {
		return nil, err
	}

	if opts.Labels == nil {
		opts.Labels = &map[string]string{}
	}
	(*opts.Labels)[workflowTemplateUIDLabelKey] = workflowTemplate.UID
	(*opts.Labels)[workflowTemplateVersionLabelKey] = fmt.Sprint(workflowTemplate.Version)
	label.MergeLabelsPrefix(*opts.Labels, workflow.Labels, label.TagPrefix)

	var argoCronWorkflow wfv1.CronWorkflow
	var argoCronWorkflowSpec wfv1.CronWorkflowSpec
	if err := argojson.UnmarshalStrict([]byte(rawCronManifest), &argoCronWorkflowSpec); err != nil {
		return nil, err
	}
	argoCronWorkflow.Spec = argoCronWorkflowSpec

	manifestBytes, err := workflowTemplate.GetWorkflowManifestBytes()
	if err != nil {
		return nil, err
	}

	workflows, err := UnmarshalWorkflows(manifestBytes, true)
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace":    namespace,
			"CronWorkflow": cronWorkflow,
			"Error":        err.Error(),
		}).Error("Error parsing workflow.")
		return nil, err
	}
	if len(workflows) != 1 {
		return nil, fmt.Errorf("more than one workflow in spec")
	}

	wf := workflows[0]

	argoCronWorkflow.Spec.WorkflowSpec = wf.Spec
	argoCreatedCronWorkflow, err := c.createCronWorkflow(namespace, &workflowTemplate.ID, &wf, &argoCronWorkflow, opts)
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace":    namespace,
			"CronWorkflow": cronWorkflow,
			"Error":        err.Error(),
		}).Error("Error parsing workflow.")
		return nil, err
	}

	cronWorkflow.Name = argoCreatedCronWorkflow.Name
	cronWorkflow.CreatedAt = argoCreatedCronWorkflow.CreationTimestamp.UTC()
	cronWorkflow.UID = string(argoCreatedCronWorkflow.ObjectMeta.UID)
	cronWorkflow.WorkflowExecution.WorkflowTemplate = workflowTemplate
	// Manifests could get big, don't return them in this case.
	cronWorkflow.WorkflowExecution.WorkflowTemplate.Manifest = ""

	tx, err := c.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	err = sb.Insert("cron_workflows").
		SetMap(sq.Eq{
			"uid":                          cronWorkflow.UID,
			"name":                         cronWorkflow.Name,
			"workflow_template_version_id": workflowTemplate.WorkflowTemplateVersionId,
			"manifest":                     cronWorkflow.Manifest,
		}).
		Suffix("RETURNING id").
		RunWith(tx).
		QueryRow().
		Scan(&cronWorkflow.ID)
	if err != nil {
		return nil, err
	}

	if len(cronWorkflow.Labels) > 0 {
		_, err = c.InsertLabelsBuilder(TypeCronWorkflow, cronWorkflow.ID, cronWorkflow.Labels).
			RunWith(tx).
			Exec()
		if err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return cronWorkflow, nil
}

func (c *Client) GetCronWorkflow(namespace, name string) (cronWorkflow *CronWorkflow, err error) {
	cronWorkflow = &CronWorkflow{}

	err = c.cronWorkflowSelectBuilderNamespaceName(namespace, name).
		RunWith(c.DB).
		QueryRow().
		Scan(cronWorkflow)

	return
}

// prefix is the label prefix.
// e.g. prefix/my-label-key: my-label-value
func (c *Client) GetCronWorkflowLabels(namespace, name, prefix string) (labels map[string]string, err error) {
	cwf, err := c.ArgoprojV1alpha1().CronWorkflows(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace": namespace,
			"Name":      name,
			"Error":     err.Error(),
		}).Error("CronWorkflow not found.")
		return nil, util.NewUserError(codes.NotFound, "CronWorkflow not found.")
	}

	labels = label.FilterByPrefix(prefix, cwf.Labels)
	labels = label.RemovePrefix(prefix, labels)

	return
}

// prefix is the label prefix.
// we delete all labels with that prefix and set the new ones
// e.g. prefix/my-label-key: my-label-value
func (c *Client) SetCronWorkflowLabels(namespace, name, prefix string, keyValues map[string]string, deleteOld bool) (workflowLabels map[string]string, err error) {
	cwf, err := c.ArgoprojV1alpha1().CronWorkflows(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace": namespace,
			"Name":      name,
			"Error":     err.Error(),
		}).Error("CronWorkflow not found.")
		return nil, util.NewUserError(codes.NotFound, "CronWorkflow not found.")
	}

	if deleteOld {
		label.DeleteWithPrefix(cwf.Labels, prefix)
	}

	label.MergeLabelsPrefix(cwf.Labels, keyValues, prefix)

	cwf, err = c.ArgoprojV1alpha1().CronWorkflows(namespace).Update(cwf)
	if err != nil {
		return nil, err
	}

	filteredMap := label.FilterByPrefix(prefix, cwf.Labels)
	filteredMap = label.RemovePrefix(prefix, filteredMap)

	return filteredMap, nil
}

func (c *Client) DeleteCronWorkflowLabel(namespace, name string, keysToDelete ...string) (labels map[string]string, err error) {
	wf, err := c.ArgoprojV1alpha1().CronWorkflows(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace": namespace,
			"Name":      name,
			"Error":     err.Error(),
		}).Error("CronWorkflow not found.")
		return nil, util.NewUserError(codes.NotFound, "CronWorkflow not found.")
	}

	label.Delete(wf.Labels, keysToDelete...)

	return wf.Labels, nil
}

func (c *Client) ListCronWorkflows(namespace, workflowTemplateUID string, pagination *pagination.PaginationRequest) (cronWorkflows []*CronWorkflow, err error) {
	sb := c.cronWorkflowSelectBuilder(namespace, workflowTemplateUID).
		OrderBy("cw.created_at DESC")

	sb = *pagination.ApplyToSelect(&sb)
	query, args, err := sb.ToSql()

	if err != nil {
		return nil, err
	}

	if err := c.DB.Select(&cronWorkflows, query, args...); err != nil {
		return nil, err
	}
	labelsMap, err := c.GetDbLabelsMapped(TypeCronWorkflow, CronWorkflowsToIds(cronWorkflows)...)
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace": namespace,
			"Error":     err.Error(),
		}).Error("Unable to get Workflow Template Labels")
		return nil, err
	}

	for _, resource := range cronWorkflows {
		resource.Labels = labelsMap[resource.ID]
	}

	return
}

func (c *Client) CountCronWorkflows(namespace, workflowTemplateUID string) (count int, err error) {
	err = c.cronWorkflowSelectBuilderNoColumns(namespace, workflowTemplateUID).
		Columns("COUNT(*)").
		RunWith(c.DB.DB).
		QueryRow().
		Scan(&count)

	return
}

func (c *Client) buildCronWorkflowDefinition(namespace string, workflowTemplateId *uint64, wf *wfv1.Workflow, cwf *wfv1.CronWorkflow, opts *WorkflowExecutionOptions) (cronWorkflow *wfv1.CronWorkflow, err error) {
	if opts == nil {
		opts = &WorkflowExecutionOptions{}
	}

	if opts.Name != "" {
		cwf.ObjectMeta.Name = opts.Name
	}
	if opts.GenerateName != "" {
		cwf.ObjectMeta.GenerateName = opts.GenerateName
	}
	if opts.Entrypoint != "" {
		cwf.Spec.WorkflowSpec.Entrypoint = opts.Entrypoint
	}
	if opts.ServiceAccount != "" {
		cwf.Spec.WorkflowSpec.ServiceAccountName = opts.ServiceAccount
	}
	if len(opts.Parameters) > 0 {
		newParams := make([]wfv1.Parameter, 0)
		passedParams := make(map[string]bool)
		for _, param := range opts.Parameters {
			newParams = append(newParams, wfv1.Parameter{
				Name:  param.Name,
				Value: param.Value,
			})
			passedParams[param.Name] = true
		}

		for _, param := range cwf.Spec.WorkflowSpec.Arguments.Parameters {
			if _, ok := passedParams[param.Name]; ok {
				// this parameter was overridden via command line
				continue
			}
			newParams = append(newParams, param)
		}
		cwf.Spec.WorkflowSpec.Arguments.Parameters = newParams
		wf.Spec.Arguments.Parameters = newParams
	}
	if opts.Labels != nil {
		cwf.ObjectMeta.Labels = *opts.Labels
	}

	err = injectExitHandlerWorkflowExecutionStatistic(wf, workflowTemplateId)
	if err != nil {
		return nil, err
	}
	err = injectInitHandlerWorkflowExecutionStatistic(wf, workflowTemplateId)
	if err != nil {
		return nil, err
	}
	if err = c.injectAutomatedFields(namespace, wf, opts); err != nil {
		return nil, err
	}

	cwf.Spec.WorkflowSpec = wf.Spec
	cwf.Spec.WorkflowMetadata = &wf.ObjectMeta

	//merge the labels
	mergedLabels := wf.ObjectMeta.Labels
	if mergedLabels == nil {
		mergedLabels = make(map[string]string)
	}
	for k, v := range *opts.Labels {
		mergedLabels[k] = v
	}
	cwf.Spec.WorkflowMetadata.Labels = mergedLabels

	return cwf, nil
}

func (c *Client) updateCronWorkflow(namespace string, name string, workflowTemplateId *uint64, wf *wfv1.Workflow, cwf *wfv1.CronWorkflow, opts *WorkflowExecutionOptions) (updatedCronWorkflow *wfv1.CronWorkflow, err error) {
	//Make sure the CronWorkflow exists before we edit it
	toUpdateCWF, err := c.ArgoprojV1alpha1().CronWorkflows(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace": namespace,
			"Name":      name,
			"Error":     err.Error(),
		}).Error("CronWorkflow not found.")
		return nil, util.NewUserError(codes.NotFound, "CronWorkflow not found.")
	}

	cwf, err = c.buildCronWorkflowDefinition(namespace, workflowTemplateId, wf, cwf, opts)
	if err != nil {
		return
	}

	cwf.Name = name
	cwf.ResourceVersion = toUpdateCWF.ResourceVersion
	updatedCronWorkflow, err = c.ArgoprojV1alpha1().CronWorkflows(namespace).Update(cwf)
	if err != nil {
		return nil, err
	}

	return
}

func (c *Client) createCronWorkflow(namespace string, workflowTemplateId *uint64, wf *wfv1.Workflow, cwf *wfv1.CronWorkflow, opts *WorkflowExecutionOptions) (createdCronWorkflow *wfv1.CronWorkflow, err error) {
	cwf, err = c.buildCronWorkflowDefinition(namespace, workflowTemplateId, wf, cwf, opts)
	if err != nil {
		return
	}

	createdCronWorkflow, err = c.ArgoprojV1alpha1().CronWorkflows(namespace).Create(cwf)
	if err != nil {
		return nil, err
	}

	return
}

func (c *Client) TerminateCronWorkflow(namespace, name string) (err error) {
	query, args, err := sb.Select(cronWorkflowColumns("wtv.version")...).
		From("cron_workflows cw").
		Join("workflow_template_versions wtv ON wtv.id = cw.workflow_template_version_id").
		Join("workflow_templates wt ON wt.id = wtv.workflow_template_id").
		Where(sq.Eq{
			"wt.namespace": namespace,
			"cw.name":      name,
		}).
		ToSql()

	cronWorkflow := &CronWorkflow{}
	if err := c.DB.Get(cronWorkflow, query, args...); err != nil {
		return err
	}

	query = `DELETE FROM workflow_executions
			 WHERE cron_workflow_id = $1`

	if _, err := c.DB.Exec(query, cronWorkflow.ID); err != nil {
		return err
	}

	query = `DELETE FROM cron_workflows 
			  USING workflow_template_versions, workflow_templates
			  WHERE cron_workflows.workflow_template_version_id = workflow_template_versions.id 
			   AND workflow_template_versions.workflow_template_id = workflow_templates.id 
			   AND workflow_templates.namespace = $1 
			   AND cron_workflows.name = $2`
	if _, err := c.DB.Exec(query, namespace, name); err != nil {
		return err
	}

	err = c.ArgoprojV1alpha1().CronWorkflows(namespace).Delete(name, nil)
	if err != nil && strings.Contains(err.Error(), "not found") {
		err = nil
	}

	return
}

func unmarshalCronWorkflows(cwfBytes []byte, strict bool) (cwfs wfv1.CronWorkflow, err error) {
	var cwf wfv1.CronWorkflow
	var jsonOpts []argojson.JSONOpt
	if strict {
		jsonOpts = append(jsonOpts, argojson.DisallowUnknownFields)
	}
	err = argojson.Unmarshal(cwfBytes, &cwf, jsonOpts...)
	if err == nil {
		return cwf, nil
	}
	return
}

func (c *Client) cronWorkflowSelectBuilder(namespace string, workflowTemplateUid string) sq.SelectBuilder {
	sb := c.cronWorkflowSelectBuilderNoColumns(namespace, workflowTemplateUid).
		Columns(cronWorkflowColumns("wtv.version")...)

	return sb
}

func (c *Client) cronWorkflowSelectBuilderNamespaceName(namespace string, name string) sq.SelectBuilder {
	sb := sb.Select("cw.id", "cw.created_at", "cw.uid", "cw.name", "cw.workflow_template_version_id").
		Columns("cw.manifest", "wtv.version").
		From("cron_workflows cw").
		Join("workflow_template_versions wtv ON wtv.id = cw.workflow_template_version_id").
		Join("workflow_templates wt ON wt.id = wtv.workflow_template_id").
		Where(sq.Eq{
			"wt.namespace": namespace,
			"cw.name":      name,
		})

	return sb
}

func (c *Client) cronWorkflowSelectBuilderNoColumns(namespace string, workflowTemplateUid string) sq.SelectBuilder {
	sb := sb.Select().
		From("cron_workflows cw").
		Join("workflow_template_versions wtv ON wtv.id = cw.workflow_template_version_id").
		Join("workflow_templates wt ON wt.id = wtv.workflow_template_id").
		Where(sq.Eq{
			"wt.namespace": namespace,
			"wt.uid":       workflowTemplateUid,
		})

	return sb
}

func (c *Client) GetCronWorkflowStatisticsForTemplates(workflowTemplates ...*WorkflowTemplate) (err error) {
	if len(workflowTemplates) == 0 {
		return nil
	}

	tx, err := c.DB.Begin()
	if err != nil {
		return err
	}

	whereIn := "wtv.workflow_template_id IN (?"
	for i := range workflowTemplates {
		if i == 0 {
			continue
		}

		whereIn += ",?"
	}
	whereIn += ")"

	ids := make([]interface{}, len(workflowTemplates))
	for i, workflowTemplate := range workflowTemplates {
		ids[i] = workflowTemplate.ID
	}

	defer tx.Rollback()

	statsSelect := `
		workflow_template_id,
		COUNT(*) total`

	query, args, err := sb.Select(statsSelect).
		From("cron_workflows cw").
		Join("workflow_template_versions wtv ON wtv.id = cw.workflow_template_version_id").
		Where(whereIn, ids...).
		GroupBy("wtv.workflow_template_id").
		ToSql()

	if err != nil {
		return err
	}
	result := make([]*CronWorkflowStatisticReport, 0)
	err = c.DB.Select(&result, query, args...)
	if err != nil {
		return err
	}

	resultMapping := make(map[uint64]*CronWorkflowStatisticReport)
	for i := range result {
		report := result[i]
		resultMapping[report.WorkflowTemplateId] = report
	}

	for _, workflowTemplate := range workflowTemplates {
		resultMap, ok := resultMapping[workflowTemplate.ID]
		if ok {
			workflowTemplate.CronWorkflowsStatisticsReport = resultMap
		}
	}

	return
}

func cronWorkflowColumns(extraColumns ...string) []string {
	results := []string{"cw.id", "cw.created_at", "cw.uid", "cw.name", "cw.workflow_template_version_id", "cw.manifest"}

	for _, str := range extraColumns {
		results = append(results, str)
	}

	return results
}
