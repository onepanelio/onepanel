package v1

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	argojson "github.com/argoproj/pkg/json"
	"github.com/ghodss/yaml"
	"github.com/google/uuid"
	"github.com/onepanelio/core/pkg/util/number"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/onepanelio/core/pkg/util"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
)

var sb = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

// Deprecated
func (c *Client) insertWorkflowTemplateVersion(workflowTemplate *WorkflowTemplate, runner sq.BaseRunner) (err error) {
	err = sb.Insert("workflow_template_versions").
		SetMap(sq.Eq{
			"workflow_template_id": workflowTemplate.ID,
			"version":              int32(time.Now().Unix()),
			"is_latest":            workflowTemplate.IsLatest,
		}).
		Suffix("RETURNING version").
		RunWith(runner).
		QueryRow().Scan(&workflowTemplate.Version)

	return
}

func (c *Client) createWorkflowTemplate(namespace string, workflowTemplate *WorkflowTemplate) (*WorkflowTemplate, error) {
	uid, err := workflowTemplate.GenerateUID()
	if err != nil {
		return nil, err
	}

	tx, err := c.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	err = sb.Insert("workflow_templates").
		SetMap(sq.Eq{
			"uid":       uid,
			"name":      workflowTemplate.Name,
			"namespace": namespace,
		}).
		Suffix("RETURNING id").
		RunWith(tx).
		QueryRow().Scan(&workflowTemplate.ID)
	if err != nil {
		return nil, err
	}

	argoWft, err := createArgoWorkflowTemplate(workflowTemplate, "1")
	argoWft, err = c.ArgoprojV1alpha1().WorkflowTemplates(namespace).Create(argoWft)
	if err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		if err := c.ArgoprojV1alpha1().WorkflowTemplates(namespace).Delete(argoWft.Name, &v1.DeleteOptions{}); err != nil {
			log.Printf("Unable to delete argo workflow template")
		}
		return nil, err
	}

	return workflowTemplate, nil
}

func (c *Client) removeIsLatestFromWorkflowTemplateVersions(workflowTemplate *WorkflowTemplate) error {
	query, args, err := sb.Update("workflow_template_versions").
		Set("is_latest", true).
		Where(sq.Eq{
			"workflow_template_id": workflowTemplate.ID,
			"is_latest":            false,
		}).
		ToSql()
	if err != nil {
		return err
	}

	if _, err := c.DB.Exec(query, args...); err != nil {
		return err
	}

	return nil
}

func (c *Client) createWorkflowTemplateVersion(namespace string, workflowTemplate *WorkflowTemplate) (*WorkflowTemplate, error) {
	query, args, err := sb.Select("id, name").
		From("workflow_templates").
		Where(sq.Eq{
			"namespace": namespace,
			"uid":       workflowTemplate.UID,
		}).
		Limit(1).ToSql()
	if err != nil {
		return nil, err
	}

	if err = c.DB.Get(workflowTemplate, query, args...); err == sql.ErrNoRows {
		return nil, nil
	}

	if err = c.insertWorkflowTemplateVersion(workflowTemplate, c.DB); err != nil {
		return nil, err
	}

	return workflowTemplate, nil
}

func (c *Client) updateWorkflowTemplateVersion(workflowTemplate *WorkflowTemplate) (*WorkflowTemplate, error) {
	query, args, err := sb.Update("workflow_template_versions").
		Set("manifest", workflowTemplate.Manifest).
		Where(sq.Eq{
			"workflow_template_id": workflowTemplate.ID,
			"version":              workflowTemplate.Version,
		}).
		ToSql()

	if err != nil {
		return nil, err
	}

	if _, err := c.DB.Exec(query, args...); err != nil {
		return nil, err
	}

	return workflowTemplate, nil
}

func (c *Client) workflowTemplatesSelectBuilder(namespace string) sq.SelectBuilder {
	sb := sb.Select("wt.id", "wt.created_at", "wt.uid", "wt.name", "wt.is_archived").
		From("workflow_templates wt").
		Where(sq.Eq{
			"wt.namespace": namespace,
		})

	return sb
}

// todo get version here?
// todo what is the version supposed to provide here? What information?
func (c *Client) getWorkflowTemplate(namespace, uid string, version int32) (workflowTemplate *WorkflowTemplate, err error) {
	workflowTemplate = &WorkflowTemplate{}

	sb := c.workflowTemplatesSelectBuilder(namespace).Where(sq.Eq{"wt.uid": uid})
	query, args, err := sb.ToSql()
	if err != nil {
		return
	}

	if err = c.DB.Get(workflowTemplate, query, args...); err == sql.ErrNoRows {
		err = nil
		workflowTemplate = nil
	}

	if workflowTemplate == nil {
		return workflowTemplate, nil
	}

	versionAsString := "latest"
	if version != 0 {
		versionAsString = fmt.Sprintf("%v", version)
	}

	argoWft, err := c.getArgoWorkflowTemplate(namespace, uid, versionAsString)
	if err != nil {
		return nil, err
	}

	manifest, err := yaml.Marshal(argoWft)
	if err != nil {
		return nil, err
	}
	workflowTemplate.Manifest = string(manifest)

	return
}

func (c *Client) getWorkflowTemplateByName(namespace, name string, version int32) (workflowTemplate *WorkflowTemplate, err error) {
	workflowTemplate = &WorkflowTemplate{}

	sb := c.workflowTemplatesSelectBuilder(namespace).Where(sq.Eq{"wt.name": name}).
		Columns("wtv.manifest").
		OrderBy("wtv.version desc").
		Limit(1)
	if version != 0 {
		sb = sb.Where(sq.Eq{"wtv.version": version})
	}
	query, args, err := sb.ToSql()
	if err != nil {
		return
	}

	if err = c.DB.Get(workflowTemplate, query, args...); err == sql.ErrNoRows {
		err = nil
		workflowTemplate = nil
	}

	return
}

func (c *Client) listWorkflowTemplateVersions(namespace, uid string) (workflowTemplateVersions []*WorkflowTemplate, err error) {
	workflowTemplateVersions = []*WorkflowTemplate{}

	query, args, err := c.workflowTemplatesSelectBuilder(namespace).Where(sq.Eq{"wt.uid": uid}).
		OrderBy("wtv.version desc").ToSql()
	if err != nil {
		return
	}

	err = c.DB.Select(&workflowTemplateVersions, query, args...)

	return
}

func (c *Client) listWorkflowTemplates(namespace string) (workflowTemplateVersions []*WorkflowTemplate, err error) {
	workflowTemplateVersions = []*WorkflowTemplate{}

	query, args, err := c.workflowTemplatesSelectBuilder(namespace).
		Options("DISTINCT ON (wt.id) wt.id,").
		Where(sq.Eq{
			"wt.is_archived": false,
		}).
		OrderBy("wt.id desc").ToSql()
	if err != nil {
		return
	}

	err = c.DB.Select(&workflowTemplateVersions, query, args...)

	return
}

func (c *Client) archiveWorkflowTemplate(namespace, uid string) (bool, error) {
	query, args, err := sb.Update("workflow_templates").
		Set("is_archived", true).
		Where(sq.Eq{
			"uid":       uid,
			"namespace": namespace,
		}).
		ToSql()

	if err != nil {
		return false, err
	}

	if _, err := c.DB.Exec(query, args...); err != nil {
		return false, err
	}

	return true, nil
}

func (c *Client) CreateWorkflowTemplate(namespace string, workflowTemplate *WorkflowTemplate) (*WorkflowTemplate, error) {
	// validate workflow template
	finalBytes, err := WrapSpecInK8s(workflowTemplate.GetManifestBytes())
	if err != nil {
		return nil, util.NewUserError(codes.InvalidArgument, err.Error())
	}

	if err := c.ValidateWorkflowExecution(namespace, finalBytes); err != nil {
		log.WithFields(log.Fields{
			"Namespace":        namespace,
			"WorkflowTemplate": workflowTemplate,
			"Error":            err.Error(),
		}).Error("Workflow could not be validated.")
		return nil, util.NewUserError(codes.InvalidArgument, err.Error())
	}

	workflowTemplate, err = c.createWorkflowTemplate(namespace, workflowTemplate)
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace":        namespace,
			"WorkflowTemplate": workflowTemplate,
			"Error":            err.Error(),
		}).Error("Could not create workflow template.")
		return nil, util.NewUserErrorWrap(err, "Workflow template")
	}

	return workflowTemplate, nil
}

func (c *Client) CreateWorkflowTemplateVersion(namespace string, workflowTemplate *WorkflowTemplate) (*WorkflowTemplate, error) {
	// validate workflow template
	finalBytes, err := WrapSpecInK8s(workflowTemplate.GetManifestBytes())
	if err != nil {
		return nil, util.NewUserError(codes.InvalidArgument, err.Error())
	}

	if err := c.ValidateWorkflowExecution(namespace, finalBytes); err != nil {
		log.WithFields(log.Fields{
			"Namespace":        namespace,
			"WorkflowTemplate": workflowTemplate,
			"Error":            err.Error(),
		}).Error("Workflow could not be validated.")
		return nil, util.NewUserError(codes.InvalidArgument, err.Error())
	}

	latest, err := c.getArgoWorkflowTemplate(namespace, workflowTemplate.UID, "latest")
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace":        namespace,
			"WorkflowTemplate": workflowTemplate,
			"Error":            err.Error(),
		}).Error("Could not get latest argo workflow template")

		return nil, err
	}

	incrementedVersion, err := number.IncrementStringInt(latest.Labels["onepanel.io/version"])

	delete(latest.Labels, "onepanel.io/version_latest")

	if _, err := c.ArgoprojV1alpha1().WorkflowTemplates(namespace).Update(latest); err != nil {
		return nil, err
	}

	updatedTemplate, err := createArgoWorkflowTemplate(workflowTemplate, incrementedVersion)
	if err != nil {
		return nil, err
	}

	updatedTemplate.TypeMeta = v1.TypeMeta{}
	updatedTemplate.ObjectMeta.ResourceVersion = ""
	updatedTemplate.ObjectMeta.SetSelfLink("")

	// todo - all the error messages
	if _, err := c.ArgoprojV1alpha1().WorkflowTemplates(namespace).Create(updatedTemplate); err != nil {
		return nil, err
	}

	//
	//workflowTemplate, err := c.createWorkflowTemplateVersion(namespace, workflowTemplate)
	//if err != nil {
	//	log.WithFields(log.Fields{
	//		"Namespace":        namespace,
	//		"WorkflowTemplate": workflowTemplate,
	//		"Error":            err.Error(),
	//	}).Error("Could not create workflow template version.")
	//	return nil, util.NewUserErrorWrap(err, "Workflow template")
	//}
	//if workflowTemplate == nil {
	//	return nil, util.NewUserError(codes.NotFound, "Workflow template not found.")
	//}

	return workflowTemplate, nil
}

func (c *Client) UpdateWorkflowTemplateVersion(namespace string, workflowTemplate *WorkflowTemplate) (*WorkflowTemplate, error) {
	// validate workflow template
	if err := c.ValidateWorkflowExecution(namespace, workflowTemplate.GetManifestBytes()); err != nil {
		log.WithFields(log.Fields{
			"Namespace":        namespace,
			"WorkflowTemplate": workflowTemplate,
			"Error":            err.Error(),
		}).Error("Workflow could not be validated.")
		return nil, util.NewUserError(codes.InvalidArgument, err.Error())
	}

	originalWorkflowTemplate, err := c.getWorkflowTemplate(namespace, workflowTemplate.UID, workflowTemplate.Version)
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace":        namespace,
			"WorkflowTemplate": workflowTemplate,
			"Error":            err.Error(),
		}).Error("Could not get workflow template.")
		return nil, util.NewUserError(codes.Unknown, "Could not update workflow template version.")
	}

	workflowTemplate.ID = originalWorkflowTemplate.ID
	workflowTemplate, err = c.updateWorkflowTemplateVersion(workflowTemplate)
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace":        namespace,
			"WorkflowTemplate": workflowTemplate,
			"Error":            err.Error(),
		}).Error("Could not update workflow template version.")
		return nil, util.NewUserErrorWrap(err, "Workflow template")
	}
	if workflowTemplate == nil {
		return nil, util.NewUserError(codes.NotFound, "Workflow template not found.")
	}

	return workflowTemplate, nil
}

// If version is 0, it returns the latest.
func (c *Client) GetWorkflowTemplate(namespace, uid string, version int32) (workflowTemplate *WorkflowTemplate, err error) {
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

func (c *Client) GetWorkflowTemplateByName(namespace, name string, version int32) (workflowTemplate *WorkflowTemplate, err error) {
	workflowTemplate, err = c.getWorkflowTemplateByName(namespace, name, version)
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

func (c *Client) CloneWorkflowTemplate() (workflowTemplate *WorkflowTemplate, err error) {

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

func (c *Client) ListWorkflowTemplates(namespace string) (workflowTemplateVersions []*WorkflowTemplate, err error) {
	workflowTemplateVersions, err = c.listWorkflowTemplates(namespace)
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace": namespace,
			"Error":     err.Error(),
		}).Error("Workflow templates not found.")
		return nil, util.NewUserError(codes.NotFound, "Workflow templates not found.")
	}

	return
}

func (c *Client) ArchiveWorkflowTemplate(namespace, uid string) (archived bool, err error) {
	workflowTemplate, err := c.getWorkflowTemplate(namespace, uid, 0)
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

	archived, err = c.archiveWorkflowTemplate(namespace, uid)
	if !archived || err != nil {
		if err != nil {
			log.WithFields(log.Fields{
				"Namespace": namespace,
				"UID":       uid,
				"Error":     err.Error(),
			}).Error("Archive Workflow Template failed.")
		}
		return false, util.NewUserError(codes.Unknown, "Unable to archive workflow template.")
	}

	return
}

func createArgoWorkflowTemplate(workflowTemplate *WorkflowTemplate, version string) (*v1alpha1.WorkflowTemplate, error) {
	var argoWft *v1alpha1.WorkflowTemplate
	var jsonOpts []argojson.JSONOpt
	jsonOpts = append(jsonOpts, argojson.DisallowUnknownFields)

	finalBytes, err := WrapSpecInK8s(workflowTemplate.GetManifestBytes())
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(finalBytes, &argoWft)
	if err != nil {
		return nil, err
	}

	newUuid, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}
	argoWft.Name = newUuid.String()
	argoWft.Labels = map[string]string{
		"onepanel.io/workflow_template":     workflowTemplate.Name,
		"onepanel.io/workflow_template_uid": workflowTemplate.UID,
		"onepanel.io/version":               version,
		"onepanel.io/version_latest":        "true",
	}

	return argoWft, nil
}

// version "latest" will get the latest version.
func (c *Client) getArgoWorkflowTemplate(namespace, workflowTemplateUid, version string) (*v1alpha1.WorkflowTemplate, error) {
	labelSelect := fmt.Sprintf("onepanel.io/workflow_template_uid=%v", workflowTemplateUid)
	if version == "latest" {
		labelSelect += ",onepanel.io/version_latest=true"
	} else {
		labelSelect += fmt.Sprintf(",onepanel.io/version=%v", version)
	}

	workflowTemplates, err := c.ArgoprojV1alpha1().WorkflowTemplates(namespace).List(v1.ListOptions{
		LabelSelector: labelSelect,
	})
	if err != nil {
		return nil, err
	}

	templates := workflowTemplates.Items
	if templates.Len() > 1 {
		return nil, errors.New("not unique result")
	}

	return &templates[0], nil
}
