package v1

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/onepanelio/core/pkg/util/pagination"
	uid2 "github.com/onepanelio/core/pkg/util/uid"
	"strconv"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	argojson "github.com/argoproj/pkg/json"
	"github.com/ghodss/yaml"
	"github.com/onepanelio/core/pkg/util"
	"github.com/onepanelio/core/pkg/util/label"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// createWorkflowTemplate creates a WorkflowTemplate and all of the DB/Argo/K8s related resources
// The returned WorkflowTemplate has the ArgoWorkflowTemplate set to the newly created one.
func (c *Client) createWorkflowTemplate(namespace string, workflowTemplate *WorkflowTemplate) (*WorkflowTemplate, *WorkflowTemplateVersion, error) {
	uid, err := uid2.GenerateUID(workflowTemplate.Name, 30)
	if err != nil {
		return nil, nil, err
	}
	workflowTemplate.UID = uid
	tx, err := c.DB.Begin()
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback()

	versionUnix := time.Now().Unix()

	err = sb.Insert("workflow_templates").
		SetMap(sq.Eq{
			"uid":       uid,
			"name":      workflowTemplate.Name,
			"namespace": namespace,
			"is_system": workflowTemplate.IsSystem,
		}).
		Suffix("RETURNING id").
		RunWith(tx).
		QueryRow().
		Scan(&workflowTemplate.ID)
	if err != nil {
		return nil, nil, err
	}

	workflowTemplateVersion := &WorkflowTemplateVersion{}
	err = sb.Insert("workflow_template_versions").
		SetMap(sq.Eq{
			"workflow_template_id": workflowTemplate.ID,
			"version":              versionUnix,
			"is_latest":            true,
			"manifest":             workflowTemplate.Manifest,
		}).
		Suffix("RETURNING id").
		RunWith(tx).
		QueryRow().
		Scan(&workflowTemplateVersion.ID)
	if err != nil {
		return nil, nil, err
	}
	workflowTemplate.WorkflowTemplateVersionID = workflowTemplateVersion.ID

	_, err = c.InsertLabelsRunner(tx, TypeWorkflowTemplateVersion, workflowTemplateVersion.ID, workflowTemplate.Labels)
	if err != nil {
		return nil, nil, err
	}

	argoWft, err := createArgoWorkflowTemplate(workflowTemplate, versionUnix)
	if err != nil {
		return nil, nil, err
	}
	argoWft.Labels[label.WorkflowTemplateVersionUid] = strconv.FormatInt(versionUnix, 10)

	if workflowTemplate.Resource != nil && workflowTemplate.ResourceUID != nil {
		if *workflowTemplate.Resource == TypeWorkspaceTemplate {
			argoWft.Labels[label.WorkspaceTemplateVersionUid] = *workflowTemplate.ResourceUID
		}
	}

	argoWft, err = c.ArgoprojV1alpha1().WorkflowTemplates(namespace).Create(argoWft)
	if err != nil {
		return nil, nil, err
	}

	if err = tx.Commit(); err != nil {
		if errDelete := c.ArgoprojV1alpha1().WorkflowTemplates(namespace).Delete(argoWft.Name, &v1.DeleteOptions{}); errDelete != nil {
			err = fmt.Errorf("%w; %s", err, errDelete)
		}
		return nil, nil, err
	}

	workflowTemplate.ArgoWorkflowTemplate = argoWft
	workflowTemplate.Version = versionUnix

	return workflowTemplate, workflowTemplateVersion, nil
}

// baseWorkflowTemplatesSelectBuilder returns a SelectBuilder selecting a WorkflowTemplate from the database for a given namespace
// no columns are selected.
func (c *Client) baseWorkflowTemplatesSelectBuilder(namespace string) sq.SelectBuilder {
	sb := sb.Select().
		From("workflow_templates wt").
		Where(sq.Eq{
			"wt.namespace": namespace,
		})

	return sb
}

// workflowTemplatesSelectBuilder returns a SelectBuilder selecting a WorkflowTemplate from the database for a given namespace
// with all of the columns of a WorkflowTemplate
func (c *Client) workflowTemplatesSelectBuilder(namespace string) sq.SelectBuilder {
	sb := c.baseWorkflowTemplatesSelectBuilder(namespace).
		Columns(getWorkflowTemplateColumns("wt")...)

	return sb
}

// countWorkflowTemplateSelectBuilder returns a select builder selecting the total count of WorkflowTemplates
// given the namespace
func (c *Client) countWorkflowTemplateSelectBuilder(namespace string) sq.SelectBuilder {
	sb := c.baseWorkflowTemplatesSelectBuilder(namespace).
		Columns("COUNT(*)")

	return sb
}

func (c *Client) workflowTemplatesVersionSelectBuilder(namespace string) sq.SelectBuilder {
	sb := sb.Select(getWorkflowTemplateVersionColumns("wtv")...).
		From("workflow_template_versions wtv").
		Join("workflow_templates wt ON wt.id = wtv.workflow_template_id").
		Where(sq.Eq{
			"wt.namespace": namespace,
		})

	return sb
}

// GetWorkflowTemplateDB returns a WorkflowTemplate from the database that is not archived, should one exist.
func (c *Client) GetWorkflowTemplateDB(namespace, name string) (workflowTemplate *WorkflowTemplate, err error) {
	workflowTemplate = &WorkflowTemplate{}

	sb := c.workflowTemplatesSelectBuilder(namespace).
		Where(sq.Eq{
			"wt.name":        name,
			"wt.is_archived": false,
		})

	err = c.DB.Getx(workflowTemplate, sb)

	return
}

// GetWorkflowTemplateVersionDB will return a WorkflowTemplateVersion given the arguments.
// version can be a number as a string, or the string "latest" to get the latest.
func (c *Client) GetWorkflowTemplateVersionDB(namespace, name, version string) (workflowTemplateVersion *WorkflowTemplateVersion, err error) {
	whereMap := sq.Eq{
		"wt.name": name,
	}

	if version == "latest" {
		whereMap["wtv.is_latest"] = "true"
	} else {
		whereMap["wtv.version"] = version
	}

	sb := c.workflowTemplatesVersionSelectBuilder(namespace).
		Where(whereMap)

	err = c.DB.Getx(workflowTemplateVersion, sb)

	return
}

// GetLatestWorkflowTemplateVersionDB returns the latest WorkflowTemplateVersion
func (c *Client) GetLatestWorkflowTemplateVersionDB(namespace, name string) (workflowTemplateVersion *WorkflowTemplateVersion, err error) {
	return c.GetWorkflowTemplateVersionDB(namespace, name, "latest")
}

func (c *Client) getWorkflowTemplateById(id uint64) (workflowTemplate *WorkflowTemplate, err error) {
	workflowTemplate = &WorkflowTemplate{}

	query, args, err := sb.Select(getWorkflowTemplateColumns("wt", "")...).
		From("workflow_templates wt").
		Where(sq.Eq{"id": id}).
		ToSql()

	if err != nil {
		return nil, err
	}

	err = c.DB.Get(workflowTemplate, query, args...)

	return
}

// If version is 0, the latest workflow template is fetched.
func (c *Client) getWorkflowTemplate(namespace, uid string, version int64) (workflowTemplate *WorkflowTemplate, err error) {
	workflowTemplate = &WorkflowTemplate{
		WorkflowExecutionStatisticReport: &WorkflowExecutionStatisticReport{},
	}

	// A new workflow template version is created upon a change, so we use it's createdAt
	// as a modified_at for the workflow template.
	sb := c.workflowTemplatesSelectBuilder(namespace).
		Columns("wtv.manifest", "wtv.version", "wtv.id workflow_template_version_id", "wtv.created_at modified_at").
		Join("workflow_template_versions wtv ON wt.id = wtv.workflow_template_id").
		Where(sq.Eq{
			"wt.uid":         uid,
			"wt.is_archived": false,
		})

	if version == 0 {
		sb = sb.Where(sq.Eq{"wtv.is_latest": true})
	} else {
		sb = sb.Where(sq.Eq{"wtv.version": version})
	}

	if err = c.Getx(workflowTemplate, sb); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, err
	}

	versionAsString := "latest"
	if version != 0 {
		versionAsString = fmt.Sprintf("%v", version)
	}

	argoWft, err := c.getArgoWorkflowTemplate(namespace, uid, versionAsString)
	if err != nil {
		return nil, err
	}
	workflowTemplate.ArgoWorkflowTemplate = argoWft

	templateVersion, err := strconv.ParseInt(argoWft.Labels[label.Version], 10, 64)
	if err != nil {
		return nil, err
	}

	workflowTemplate.Version = templateVersion

	labelsMap, err := c.GetDBLabelsMapped(TypeWorkflowTemplateVersion, workflowTemplate.WorkflowTemplateVersionID)
	if err != nil {
		return workflowTemplate, err
	}

	workflowTemplate.Labels = labelsMap[workflowTemplate.WorkflowTemplateVersionID]

	return workflowTemplate, nil
}

func (c *Client) listWorkflowTemplateVersions(namespace, uid string) (workflowTemplateVersions []*WorkflowTemplate, err error) {
	dbVersions, err := c.listDBWorkflowTemplateVersions(namespace, uid)
	if err != nil {
		return nil, err
	}

	labelsMap, err := c.GetDBLabelsMapped(TypeWorkflowTemplateVersion, WorkflowTemplateVersionsToIDs(dbVersions)...)
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace": namespace,
			"Error":     err.Error(),
		}).Error("Unable to get Workflow Template Version Labels")
		return nil, err
	}

	for _, version := range dbVersions {
		version.Labels = labelsMap[version.ID]

		newItem := WorkflowTemplate{
			ID:         version.WorkflowTemplate.ID,
			CreatedAt:  version.CreatedAt.UTC(),
			UID:        version.UID,
			Name:       version.WorkflowTemplate.Name,
			Manifest:   version.Manifest,
			Version:    version.Version,
			IsLatest:   version.IsLatest,
			IsArchived: version.WorkflowTemplate.IsArchived,
			Labels:     version.Labels,
		}

		workflowTemplateVersions = append(workflowTemplateVersions, &newItem)
	}

	return
}

func (c *Client) listWorkflowTemplates(namespace string, paginator *pagination.PaginationRequest) (workflowTemplateVersions []*WorkflowTemplate, err error) {
	workflowTemplateVersions = []*WorkflowTemplate{}

	sb := c.workflowTemplatesSelectBuilder(namespace).
		Column("COUNT(wtv.*) versions, MAX(wtv.id) workflow_template_version_id").
		Join("workflow_template_versions wtv ON wtv.workflow_template_id = wt.id").
		GroupBy("wt.id", "wt.created_at", "wt.uid", "wt.name", "wt.is_archived").
		Where(sq.Eq{
			"wt.is_archived": false,
			"wt.is_system":   false,
		}).
		OrderBy("wt.created_at DESC")

	sb = *paginator.ApplyToSelect(&sb)
	query, args, err := sb.ToSql()
	if err != nil {
		return
	}

	err = c.DB.Select(&workflowTemplateVersions, query, args...)

	return
}

func (c *Client) CountWorkflowTemplates(namespace string) (count int, err error) {
	err = sb.Select("COUNT( DISTINCT( wt.id ))").
		From("workflow_templates wt").
		Join("workflow_template_versions wtv ON wtv.workflow_template_id = wt.id").
		Where(sq.Eq{
			"wt.namespace":   namespace,
			"wt.is_archived": false,
			"wt.is_system":   false,
		}).
		RunWith(c.DB.DB).
		QueryRow().
		Scan(&count)

	return
}

func (c *Client) validateWorkflowTemplate(namespace string, workflowTemplate *WorkflowTemplate) (err error) {
	// validate workflow template
	finalBytes, err := workflowTemplate.WrapSpec()
	if err != nil {
		return
	}
	err = c.ValidateWorkflowExecution(namespace, finalBytes)
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace":        namespace,
			"WorkflowTemplate": workflowTemplate,
			"Error":            err.Error(),
		}).Error("Workflow could not be validated.")
	}

	return
}

func (c *Client) CreateWorkflowTemplate(namespace string, workflowTemplate *WorkflowTemplate) (*WorkflowTemplate, error) {
	// validate workflow template
	if err := c.validateWorkflowTemplate(namespace, workflowTemplate); err != nil {
		return nil, util.NewUserError(codes.InvalidArgument, err.Error())
	}

	newWorkflowTemplate, _, err := c.createWorkflowTemplate(namespace, workflowTemplate)
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace":        namespace,
			"WorkflowTemplate": workflowTemplate,
			"Error":            err.Error(),
		}).Error("Could not create workflow template.")
		return nil, util.NewUserErrorWrap(err, "Workflow template")
	}

	return newWorkflowTemplate, nil
}

func (c *Client) CreateWorkflowTemplateVersion(namespace string, workflowTemplate *WorkflowTemplate) (*WorkflowTemplate, error) {
	if workflowTemplate.UID == "" {
		return nil, fmt.Errorf("uid required for CreateWorkflowTemplateVersion")
	}

	// validate workflow template
	if err := c.validateWorkflowTemplate(namespace, workflowTemplate); err != nil {
		return nil, util.NewUserError(codes.InvalidArgument, err.Error())
	}

	versionUnix := time.Now().Unix()

	tx, err := c.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	wftSb := c.workflowTemplatesSelectBuilder(namespace).
		Where(sq.Eq{
			"wt.uid":         workflowTemplate.UID,
			"wt.is_archived": false,
		})
	query, args, err := wftSb.ToSql()
	if err != nil {
		return nil, err
	}
	workflowTemplateDb := &WorkflowTemplate{}
	if err = c.DB.Get(workflowTemplateDb, query, args...); err != nil {
		return nil, err
	}

	_, err = sb.Update("workflow_template_versions").
		Set("is_latest", false).
		Where(sq.Eq{
			"workflow_template_id": workflowTemplateDb.ID,
		}).
		RunWith(tx).
		Exec()
	if err != nil {
		return nil, err
	}

	workflowTemplateVersionID := uint64(0)
	err = sb.Insert("workflow_template_versions").
		SetMap(sq.Eq{
			"workflow_template_id": workflowTemplateDb.ID,
			"version":              versionUnix,
			"is_latest":            true,
			"manifest":             workflowTemplate.Manifest,
		}).
		Suffix("RETURNING id").
		RunWith(tx).
		QueryRow().
		Scan(&workflowTemplateVersionID)
	if err != nil {
		return nil, err
	}

	workflowTemplate.WorkflowTemplateVersionID = workflowTemplateVersionID
	latest, err := c.getArgoWorkflowTemplate(namespace, workflowTemplate.UID, "latest")
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace":        namespace,
			"WorkflowTemplate": workflowTemplate,
			"Error":            err.Error(),
		}).Error("Could not get latest argo workflow template")

		return nil, err
	}

	delete(latest.Labels, label.VersionLatest)

	latest, err = c.ArgoprojV1alpha1().WorkflowTemplates(namespace).Update(latest)
	if err != nil {
		return nil, err
	}

	updatedTemplate, err := createArgoWorkflowTemplate(workflowTemplate, versionUnix)
	if err != nil {
		latest.Labels[label.VersionLatest] = "true"
		if _, err := c.ArgoprojV1alpha1().WorkflowTemplates(namespace).Update(latest); err != nil {
			return nil, err
		}

		return nil, err
	}

	updatedTemplate.TypeMeta = v1.TypeMeta{}
	updatedTemplate.ObjectMeta.ResourceVersion = ""
	updatedTemplate.ObjectMeta.SetSelfLink("")
	updatedTemplate.Labels[label.WorkflowTemplateVersionUid] = strconv.FormatInt(versionUnix, 10)

	parametersMap, err := workflowTemplate.GetParametersKeyString()
	if err != nil {
		return nil, err
	}

	if updatedTemplate.Annotations == nil {
		updatedTemplate.Annotations = make(map[string]string)
	}
	for key, value := range parametersMap {
		updatedTemplate.Annotations[key] = value
	}

	if _, err := c.ArgoprojV1alpha1().WorkflowTemplates(namespace).Create(updatedTemplate); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	workflowTemplate.Version = versionUnix

	return workflowTemplate, nil
}

// GetWorkflowTemplate returns a WorkflowTemplate with data loaded from various sources
// If version is 0, it returns the latest version data.
//
// Data loaded includes
// * Database Information
// * ArgoWorkflowTemplate
// * Labels
func (c *Client) GetWorkflowTemplate(namespace, uid string, version int64) (workflowTemplate *WorkflowTemplate, err error) {
	workflowTemplate, err = c.getWorkflowTemplate(namespace, uid, version)
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace":        namespace,
			"WorkflowTemplate": workflowTemplate,
			"Error":            err.Error(),
		}).Error("Get Workflow Template failed.")
		return nil, util.NewUserError(codes.Unknown, "Unknown error.")
	}
	if workflowTemplate == nil {
		return nil, util.NewUserError(codes.NotFound, "Workflow template not found.")
	}

	return
}

// GetLatestWorkflowTemplate returns a workflow template with the latest version data.
func (c *Client) GetLatestWorkflowTemplate(namespace, uid string) (workflowTemplate *WorkflowTemplate, err error) {
	return c.GetWorkflowTemplate(namespace, uid, 0)
}

// CountWorkflowTemplatesByName returns the number of WorkflowTemplates given the arguments.
// If archived is nil, it is not considered.
func (c *Client) CountWorkflowTemplatesByName(namespace, name string, archived *bool) (count uint64, err error) {
	sb := c.countWorkflowTemplateSelectBuilder(namespace).
		Where(sq.Eq{"wt.name": name})

	if archived != nil {
		sb = sb.Where(sq.Eq{"wt.is_archived": *archived})
	}

	err = sb.RunWith(c.DB).
		QueryRow().
		Scan(&count)

	return
}

// CountWorkflowTemplateVersions returns the number of versions a non-archived WorkflowTemplate has.
func (c *Client) CountWorkflowTemplateVersions(namespace, uid string) (count uint64, err error) {
	err = sb.Select("COUNT(*)").
		From("workflow_templates wt").
		Join("workflow_template_versions wtv ON wtv.workflow_template_id = wt.id").
		Where(sq.Eq{
			"wt.namespace":   namespace,
			"wt.uid":         uid,
			"wt.is_archived": false,
		}).
		RunWith(c.DB).
		QueryRow().
		Scan(&count)

	return
}

func (c *Client) ListWorkflowTemplateVersions(namespace, uid string) (workflowTemplateVersions []*WorkflowTemplate, err error) {
	workflowTemplateVersions, err = c.listWorkflowTemplateVersions(namespace, uid)
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace": namespace,
			"UID":       uid,
			"Error":     err.Error(),
		}).Error("Workflow template versions not found.")
		return nil, util.NewUserError(codes.NotFound, "Workflow template versions not found.")
	}

	return
}

func (c *Client) ListWorkflowTemplates(namespace string, paginator *pagination.PaginationRequest) (workflowTemplateVersions []*WorkflowTemplate, err error) {
	workflowTemplateVersions, err = c.listWorkflowTemplates(namespace, paginator)
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace": namespace,
			"Error":     err.Error(),
		}).Error("Workflow templates not found.")
		return nil, util.NewUserError(codes.NotFound, "Workflow templates not found.")
	}

	err = c.GetWorkflowExecutionStatisticsForTemplates(workflowTemplateVersions...)
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace": namespace,
			"Error":     err.Error(),
		}).Error("Unable to get Workflow Execution Statistic for Templates.")
		return nil, util.NewUserError(codes.NotFound, "Unable to get Workflow Execution Statistic for Templates.")
	}

	err = c.GetCronWorkflowStatisticsForTemplates(workflowTemplateVersions...)
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace": namespace,
			"Error":     err.Error(),
		}).Error("Unable to get Cron Workflow Statistic for Templates.")
		return nil, util.NewUserError(codes.NotFound, "Unable to get Cron Workflow Statistic for Templates.")
	}

	labelsMap, err := c.GetDBLabelsMapped(TypeWorkflowTemplateVersion, WorkflowTemplatesToVersionIDs(workflowTemplateVersions)...)
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace": namespace,
			"Error":     err.Error(),
		}).Error("Unable to get Workflow Template Labels")
		return nil, err
	}

	for _, workflowTemplate := range workflowTemplateVersions {
		workflowTemplate.Labels = labelsMap[workflowTemplate.WorkflowTemplateVersionID]
	}

	return
}

func (c *Client) getLatestWorkflowTemplate(namespace, uid string) (*WorkflowTemplate, error) {
	return c.getWorkflowTemplate(namespace, uid, 0) //version=0 means latest
}

func (c *Client) ArchiveWorkflowTemplate(namespace, uid string) (archived bool, err error) {
	workflowTemplate, err := c.getLatestWorkflowTemplate(namespace, uid)
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace": namespace,
			"UID":       uid,
			"Error":     err.Error(),
		}).Error("Get Workflow Template failed.")
		return false, util.NewUserError(codes.Unknown, "Unable to archive workflow template.")
	}
	if workflowTemplate == nil {
		return false, util.NewUserError(codes.NotFound, "Workflow template not found.")
	}

	wftVersions, err := c.listWorkflowTemplateVersions(namespace, uid)
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace": namespace,
			"UID":       uid,
			"Error":     err.Error(),
		}).Error("Get Workflow Template Versions failed.")
		return false, util.NewUserError(codes.Unknown, "Unable to archive workflow template.")
	}

	//cron workflows
	cronWorkflows := []*CronWorkflow{}
	cwfSB := c.cronWorkflowSelectBuilder(namespace, uid).
		OrderBy("cw.created_at DESC")

	if err := c.DB.Selectx(&cronWorkflows, cwfSB); err != nil {
		log.WithFields(log.Fields{
			"Namespace": namespace,
			"UID":       uid,
			"Error":     err.Error(),
		}).Error("Get Cron Workflows failed.")
		return false, util.NewUserError(codes.Unknown, "Unable to archive workflow template.")
	}
	for _, cwf := range cronWorkflows {
		err = c.ArchiveCronWorkflow(namespace, cwf.Name)
		if err != nil {
			log.WithFields(log.Fields{
				"Namespace": namespace,
				"UID":       uid,
				"Error":     err.Error(),
			}).Error("Archive Cron Workflow failed.")
			return false, util.NewUserError(codes.Unknown, "Unable to archive workflow template.")
		}
	}

	//workflow executions
	paginator := pagination.NewRequest(0, 100)

	for _, wftVer := range wftVersions {
		wfTempVer := strconv.FormatInt(wftVer.Version, 10)
		workflowTemplateName := uid + "-v" + wfTempVer

		for {
			wfs, err := c.ListWorkflowExecutions(namespace, uid, wfTempVer, &paginator)
			if err != nil {
				log.WithFields(log.Fields{
					"Namespace": namespace,
					"UID":       uid,
					"Error":     err.Error(),
				}).Error("Get Workflow Executions failed.")
				return false, util.NewUserError(codes.Unknown, "Unable to archive workflow template.")
			}
			if len(wfs) == 0 {
				break
			}
			for _, wf := range wfs {
				err = c.ArchiveWorkflowExecution(namespace, wf.UID)
				if err != nil {
					log.WithFields(log.Fields{
						"Namespace": namespace,
						"UID":       uid,
						"Error":     err.Error(),
					}).Error("Archive Workflow Execution Failed.")
					return false, util.NewUserError(codes.Unknown, "Unable to archive workflow template.")
				}
			}
		}

		err = c.ArgoprojV1alpha1().WorkflowTemplates(namespace).Delete(workflowTemplateName, nil)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				return true, nil
			}
			return false, err
		}
	}

	_, err = sb.Update("workflow_templates").
		Set("is_archived", true).
		Where(sq.Eq{
			"uid":       uid,
			"namespace": namespace,
		}).RunWith(c.DB).
		Exec()

	if err != nil {
		log.WithFields(log.Fields{
			"Namespace": namespace,
			"UID":       uid,
			"Error":     err.Error(),
		}).Error("Archive Workflow Template DB failed.")
		return false, util.NewUserError(codes.Unknown, "Unable to archive workflow template.")
	}

	return true, nil
}

func createArgoWorkflowTemplate(workflowTemplate *WorkflowTemplate, version int64) (*v1alpha1.WorkflowTemplate, error) {
	var argoWft *v1alpha1.WorkflowTemplate
	var jsonOpts []argojson.JSONOpt
	jsonOpts = append(jsonOpts, argojson.DisallowUnknownFields)

	finalBytes, err := workflowTemplate.WrapSpec()
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(finalBytes, &argoWft)
	if err != nil {
		return nil, err
	}

	worfklowTemplateName, err := uid2.GenerateUID(workflowTemplate.Name, 30)
	if err != nil {
		return nil, err
	}

	argoWft.Name = fmt.Sprintf("%v-v%v", worfklowTemplateName, version)

	labels := map[string]string{
		label.WorkflowTemplate:    worfklowTemplateName,
		label.WorkflowTemplateUid: workflowTemplate.UID,
		label.Version:             fmt.Sprintf("%v", version),
		label.VersionLatest:       "true",
	}

	label.MergeLabelsPrefix(labels, workflowTemplate.Labels, label.TagPrefix)
	argoWft.Labels = labels

	return argoWft, nil
}

// version "latest" will get the latest version.
func (c *Client) getArgoWorkflowTemplate(namespace, workflowTemplateUid, version string) (*v1alpha1.WorkflowTemplate, error) {
	labelSelect := fmt.Sprintf("%v=%v", label.WorkflowTemplateUid, workflowTemplateUid)
	if version == "latest" {
		labelSelect += "," + label.VersionLatest + "=true"
	} else {
		labelSelect += fmt.Sprintf(",%v=%v", label.Version, version)
	}

	workflowTemplates, err := c.ArgoprojV1alpha1().WorkflowTemplates(namespace).List(v1.ListOptions{
		LabelSelector: labelSelect,
	})
	if err != nil {
		return nil, err
	}

	templates := workflowTemplates.Items
	if templates.Len() == 0 {
		return nil, errors.New("not found")
	}

	if templates.Len() > 1 {
		return nil, errors.New("not unique result")
	}

	return &templates[0], nil
}

func (c *Client) listArgoWorkflowTemplates(namespace, workflowTemplateUid string) (*[]v1alpha1.WorkflowTemplate, error) {
	labelSelect := fmt.Sprintf("%v=%v", label.WorkflowTemplateUid, workflowTemplateUid)
	workflowTemplates, err := c.ArgoprojV1alpha1().WorkflowTemplates(namespace).List(v1.ListOptions{
		LabelSelector: labelSelect,
	})
	if err != nil {
		return nil, err
	}

	templates := []v1alpha1.WorkflowTemplate(workflowTemplates.Items)

	return &templates, nil
}

func (c *Client) listDBWorkflowTemplateVersions(namespace, workflowTemplateUID string) ([]*WorkflowTemplateVersion, error) {
	versions := make([]*WorkflowTemplateVersion, 0)

	sb := c.workflowTemplatesVersionSelectBuilder(namespace).
		Columns(`wt.id "workflow_template.id"`, `wt.created_at "workflow_template.created_at"`).
		Columns(`wt.name "workflow_template.name"`, `wt.is_archived "workflow_template.is_archived"`).
		Where(sq.Eq{
			"wt.uid":         workflowTemplateUID,
			"wt.is_archived": false,
		}).
		OrderBy("wtv.created_at DESC")

	query, args, err := sb.ToSql()
	if err != nil {
		return versions, err
	}

	if err := c.DB.Select(&versions, query, args...); err != nil {
		return versions, err
	}

	return versions, nil
}

// prefix is the label prefix.
// e.g. prefix/my-label-key: my-label-value
// if version is 0, latest is used.
func (c *Client) GetWorkflowTemplateLabels(namespace, name, prefix string, version int64) (labels map[string]string, err error) {
	versionAsString := "latest"
	if version != 0 {
		versionAsString = fmt.Sprintf("%v", version)
	}

	wf, err := c.getArgoWorkflowTemplate(namespace, name, versionAsString)
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace": namespace,
			"Name":      name,
			"Error":     err.Error(),
		}).Error("Workflow Template not found.")
		return nil, util.NewUserError(codes.NotFound, "Workflow Template not found.")
	}

	labels = label.FilterByPrefix(prefix, wf.Labels)
	labels = label.RemovePrefix(prefix, labels)

	return
}
