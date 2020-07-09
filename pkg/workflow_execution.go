package v1

import (
	"bufio"
	"cloud.google.com/go/storage"
	"encoding/json"
	"errors"
	"fmt"
	sq "github.com/Masterminds/squirrel"
	"github.com/onepanelio/core/pkg/util/label"
	"github.com/onepanelio/core/pkg/util/pagination"
	"github.com/onepanelio/core/pkg/util/ptr"
	uid2 "github.com/onepanelio/core/pkg/util/uid"
	"golang.org/x/net/context"
	"gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/watch"
	"net/http"
	"strconv"
	"strings"
	"time"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/argo/workflow/templateresolution"
	argoutil "github.com/argoproj/argo/workflow/util"
	"github.com/argoproj/argo/workflow/validate"
	argojson "github.com/argoproj/pkg/json"
	"github.com/onepanelio/core/pkg/util"
	"github.com/onepanelio/core/pkg/util/env"
	"github.com/onepanelio/core/pkg/util/s3"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

var (
	readEndOffset                   = env.GetEnv("ARTIFACT_RERPOSITORY_OBJECT_RANGE", "-102400")
	workflowTemplateUIDLabelKey     = "onepanel.io/workflow-template-uid"
	workflowTemplateVersionLabelKey = "onepanel.io/workflow-template-version"
)

func typeWorkflow(wf *wfv1.Workflow) (workflow *WorkflowExecution) {
	manifest, err := json.Marshal(wf)
	if err != nil {
		return
	}
	workflow = &WorkflowExecution{
		UID:       string(wf.UID),
		CreatedAt: wf.CreationTimestamp.UTC(),
		Name:      wf.Name,
		Manifest:  string(manifest),
	}

	return
}

func UnmarshalWorkflows(wfBytes []byte, strict bool) (wfs []wfv1.Workflow, err error) {
	if len(wfBytes) == 0 {
		return nil, fmt.Errorf("UnmarshalWorkflows unable to work on empty bytes")
	}

	var wf wfv1.Workflow
	var jsonOpts []argojson.JSONOpt
	if strict {
		jsonOpts = append(jsonOpts, argojson.DisallowUnknownFields)
	}

	wfBytes, err = filterOutCustomTypesFromManifest(wfBytes)
	if err != nil {
		return
	}

	err = argojson.Unmarshal(wfBytes, &wf, jsonOpts...)
	if err == nil {
		return []wfv1.Workflow{wf}, nil
	}
	wfs, err = common.SplitWorkflowYAMLFile(wfBytes, strict)
	if err == nil {
		return
	}

	return
}

// appendArtifactRepositoryConfigIfMissing appends default artifact repository config to artifacts that have a key.
// Artifacts that contain anything other than key are skipped.
func injectArtifactRepositoryConfig(artifact *wfv1.Artifact, namespaceConfig *NamespaceConfig) {
	if artifact.S3 != nil && artifact.S3.Key != "" && artifact.S3.Bucket == "" {
		s3Config := namespaceConfig.ArtifactRepository.S3
		artifact.S3.Endpoint = s3Config.Endpoint
		artifact.S3.Bucket = s3Config.Bucket
		artifact.S3.Region = s3Config.Region
		artifact.S3.Insecure = ptr.Bool(s3Config.Insecure)
		artifact.S3.SecretKeySecret = s3Config.SecretKeySecret
		artifact.S3.AccessKeySecret = s3Config.AccessKeySecret
	}

	// Default to no compression for artifacts
	artifact.Archive = &wfv1.ArchiveStrategy{
		None: &wfv1.NoneStrategy{},
	}
}

func (c *Client) injectAutomatedFields(namespace string, wf *wfv1.Workflow, opts *WorkflowExecutionOptions) (err error) {
	if opts.PodGCStrategy == nil {
		if wf.Spec.PodGC == nil {
			//TODO - Load this data from onepanel config-map or secret
			podGCStrategy := env.GetEnv("ARGO_POD_GC_STRATEGY", "OnPodCompletion")
			strategy := PodGCStrategy(podGCStrategy)
			wf.Spec.PodGC = &wfv1.PodGC{
				Strategy: strategy,
			}
		}
	} else {
		wf.Spec.PodGC = &wfv1.PodGC{
			Strategy: *opts.PodGCStrategy,
		}
	}

	// Get artifact repository config from current namespace
	wf.Spec.ArtifactRepositoryRef = &wfv1.ArtifactRepositoryRef{
		ConfigMap: "onepanel",
		Key:       "artifactRepository",
	}

	// Create dev/shm volume
	wf.Spec.Volumes = append(wf.Spec.Volumes, corev1.Volume{
		Name: "sys-dshm",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{
				Medium: corev1.StorageMediumMemory,
			},
		},
	})

	systemConfig, err := c.GetSystemConfig()
	if err != nil {
		return err
	}
	namespaceConfig, err := c.GetNamespaceConfig(namespace)
	if err != nil {
		return err
	}
	for i, template := range wf.Spec.Templates {
		// Do not inject Istio sidecars in workflows
		if template.Metadata.Annotations == nil {
			wf.Spec.Templates[i].Metadata.Annotations = make(map[string]string)
		}
		wf.Spec.Templates[i].Metadata.Annotations["sidecar.istio.io/inject"] = "false"

		if template.Container == nil {
			continue
		}

		// Mount dev/shm
		wf.Spec.Templates[i].Container.VolumeMounts = append(template.Container.VolumeMounts, corev1.VolumeMount{
			Name:      "sys-dshm",
			MountPath: "/dev/shm",
		})

		// Always add output artifacts for metrics but make them optional
		wf.Spec.Templates[i].Outputs.Artifacts = append(template.Outputs.Artifacts, wfv1.Artifact{
			Name:     "sys-metrics",
			Path:     "/tmp/sys-metrics.json",
			Optional: true,
			Archive: &wfv1.ArchiveStrategy{
				None: &wfv1.NoneStrategy{},
			},
		})

		// Extend artifact credentials if only key is provided
		for j, artifact := range template.Outputs.Artifacts {
			injectArtifactRepositoryConfig(&artifact, namespaceConfig)
			wf.Spec.Templates[i].Outputs.Artifacts[j] = artifact
		}

		for j, artifact := range template.Inputs.Artifacts {
			injectArtifactRepositoryConfig(&artifact, namespaceConfig)
			wf.Spec.Templates[i].Inputs.Artifacts[j] = artifact
		}

		//Generate ENV vars from secret, if there is a container present in the workflow
		//Get template ENV vars, avoid over-writing them with secret values
		env.AddDefaultEnvVarsToContainer(template.Container)
		env.PrependEnvVarToContainer(template.Container, "ONEPANEL_API_URL", systemConfig["ONEPANEL_API_URL"])
		env.PrependEnvVarToContainer(template.Container, "ONEPANEL_FQDN", systemConfig["ONEPANEL_FQDN"])
		env.PrependEnvVarToContainer(template.Container, "ONEPANEL_DOMAIN", systemConfig["ONEPANEL_DOMAIN"])
		env.PrependEnvVarToContainer(template.Container, "ONEPANEL_PROVIDER_TYPE", systemConfig["PROVIDER_TYPE"])
		env.PrependEnvVarToContainer(template.Container, "ONEPANEL_RESOURCE_NAMESPACE", "{{workflow.namespace}}")
		env.PrependEnvVarToContainer(template.Container, "ONEPANEL_RESOURCE_UID", "{{workflow.name}}")
	}

	return
}

func (c *Client) ArchiveWorkflowExecution(namespace, uid string) error {
	_, err := sb.Update("workflow_executions").Set("is_archived", true).Where(sq.Eq{
		"uid":       uid,
		"namespace": namespace,
	}).RunWith(c.DB).Exec()
	if err != nil {
		return err
	}

	err = c.ArgoprojV1alpha1().Workflows(namespace).Delete(uid, nil)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil
		}
		return err
	}

	return nil
}

/*
	Name is == to UID, no user friendly name.
	Workflow execution name == uid, example: name = my-friendly-wf-name-8skjz, uid = my-friendly-wf-name-8skjz
*/
func (c *Client) createWorkflow(namespace string, workflowTemplateId uint64, workflowTemplateVersionId uint64, wf *wfv1.Workflow, opts *WorkflowExecutionOptions) (newDbId uint64, createdWorkflow *wfv1.Workflow, err error) {
	if opts == nil {
		opts = &WorkflowExecutionOptions{}
	}

	if opts.Name != "" {
		wf.ObjectMeta.Name = opts.Name
	}
	if opts.GenerateName != "" {
		wf.ObjectMeta.GenerateName = opts.GenerateName
	}
	if opts.Entrypoint != "" {
		wf.Spec.Entrypoint = opts.Entrypoint
	}
	if opts.ServiceAccount != "" {
		wf.Spec.ServiceAccountName = opts.ServiceAccount
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

		for _, param := range wf.Spec.Arguments.Parameters {
			if _, ok := passedParams[param.Name]; ok {
				// this parameter was overridden via command line
				continue
			}
			newParams = append(newParams, param)
		}
		wf.Spec.Arguments.Parameters = newParams
	}
	if opts.Labels != nil {
		wf.ObjectMeta.Labels = opts.Labels
	}

	err = injectWorkflowExecutionStatusCaller(wf, wfv1.NodeRunning)
	if err != nil {
		return 0, nil, err
	}

	err = injectExitHandlerWorkflowExecutionStatistic(wf, &workflowTemplateId)
	if err != nil {
		return 0, nil, err
	}

	if err = c.injectAutomatedFields(namespace, wf, opts); err != nil {
		return 0, nil, err
	}

	createdWorkflow, err = c.ArgoprojV1alpha1().Workflows(namespace).Create(wf)
	if err != nil {
		return 0, nil, err
	}

	uid, err := uid2.GenerateUID(createdWorkflow.Name, 63)
	if err != nil {
		return 0, nil, err
	}
	//Create an entry for workflow_executions statistic
	//CURL code will hit the API endpoint that will update the db row
	newDbId, err = c.insertPreWorkflowExecutionStatistic(namespace, createdWorkflow.Name, workflowTemplateVersionId, createdWorkflow.CreationTimestamp.UTC(), uid, opts.Parameters)
	if err != nil {
		return 0, nil, err
	}

	return
}

func (c *Client) ValidateWorkflowExecution(namespace string, manifest []byte) (err error) {
	manifest, err = filterOutCustomTypesFromManifest(manifest)
	if err != nil {
		return
	}

	workflows, err := UnmarshalWorkflows(manifest, true)
	if err != nil {
		return
	}

	wftmplGetter := templateresolution.WrapWorkflowTemplateInterface(c.ArgoprojV1alpha1().WorkflowTemplates(namespace))
	for _, wf := range workflows {
		c.injectAutomatedFields(namespace, &wf, &WorkflowExecutionOptions{})
		_, err = validate.ValidateWorkflow(wftmplGetter, &wf, validate.ValidateOpts{})
		if err != nil {
			return
		}

		// Check that entrypoint and onExit templates are DAGs
		for _, t := range wf.Spec.Templates {
			if t.Name == wf.Spec.Entrypoint && t.DAG == nil {
				return errors.New("\"entrypoint\" template should be a DAG")
			}

			if wf.Spec.OnExit != "" && t.Name == wf.Spec.OnExit && t.DAG == nil {
				return errors.New("\"onExit\" template should be a DAG")
			}
		}
	}

	return
}

// CreateWorkflowExecution creates an argo workflow execution and related resources.
// Note that the workflow template is loaded from the database/k8s, so workflow.WorkflowTemplate.Manifest is not used.
// Required:
//  * workflow.Parameters
//  * workflow.Labels (optional)
func (c *Client) CreateWorkflowExecution(namespace string, workflow *WorkflowExecution, workflowTemplate *WorkflowTemplate) (*WorkflowExecution, error) {
	opts := &WorkflowExecutionOptions{
		Labels:     make(map[string]string),
		Parameters: workflow.Parameters,
	}

	nameUID, err := uid2.GenerateUID(workflowTemplate.Name, 63)
	if err != nil {
		return nil, err
	}
	opts.GenerateName = nameUID + "-"

	opts.Labels[workflowTemplateUIDLabelKey] = workflowTemplate.UID
	opts.Labels[workflowTemplateVersionLabelKey] = fmt.Sprint(workflowTemplate.Version)
	label.MergeLabelsPrefix(opts.Labels, workflow.Labels, label.TagPrefix)

	// @todo we need to enforce the below requirement in API.
	//UX will prevent multiple workflows
	manifest, err := workflowTemplate.GetWorkflowManifestBytes()
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace":        namespace,
			"WorkflowTemplate": workflowTemplate,
			"Error":            err.Error(),
		}).Error("Error with getting WorkflowManifest from workflow template")
		return nil, err
	}

	workflows, err := UnmarshalWorkflows(manifest, true)
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace": namespace,
			"Workflow":  workflow,
			"Error":     err.Error(),
		}).Error("Error parsing workflow.")
		return nil, err
	}

	id, createdWorkflow, err := c.createWorkflow(namespace, workflowTemplate.ID, workflowTemplate.WorkflowTemplateVersionID, &workflows[0], opts)
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace": namespace,
			"Workflow":  workflow,
			"Error":     err.Error(),
		}).Error("Error parsing workflow.")
		return nil, err
	}

	if _, err := c.InsertLabels(TypeWorkflowExecution, id, workflow.Labels); err != nil {
		return nil, err
	}

	if createdWorkflow == nil {
		err = errors.New("unable to create workflow")
		log.WithFields(log.Fields{
			"Namespace":        namespace,
			"WorkflowTemplate": workflowTemplate,
			"Error":            err.Error(),
		}).Error("Error parsing workflow.")

		return nil, err
	}

	workflow.ID = id
	workflow.Name = createdWorkflow.Name
	workflow.CreatedAt = createdWorkflow.CreationTimestamp.UTC()
	workflow.UID = createdWorkflow.Name
	workflow.WorkflowTemplate = workflowTemplate

	return workflow, nil
}

func (c *Client) CloneWorkflowExecution(namespace, uid string) (*WorkflowExecution, error) {
	// TODO do you need the and template here?
	workflowExecution, err := c.getWorkflowExecutionAndTemplate(namespace, uid)
	if err != nil {
		return nil, err
	}

	workflowTemplate, err := c.GetWorkflowTemplate(namespace, workflowExecution.WorkflowTemplate.UID, workflowExecution.WorkflowTemplate.Version)
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace": namespace,
			"Workflow":  workflowExecution,
			"Error":     err.Error(),
		}).Error("Error with getting workflow template.")
		return nil, util.NewUserError(codes.NotFound, "Error with getting workflow template.")
	}

	return c.CreateWorkflowExecution(namespace, workflowExecution, workflowTemplate)
}

func (c *Client) insertPreWorkflowExecutionStatistic(namespace, name string, workflowTemplateVersionId uint64, createdAt time.Time, uid string, parameters []Parameter) (newId uint64, err error) {
	tx, err := c.DB.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	parametersJSON, err := json.Marshal(parameters)
	if err != nil {
		return 0, err
	}

	insertMap := sq.Eq{
		"uid":                          uid,
		"workflow_template_version_id": workflowTemplateVersionId,
		"name":                         name,
		"namespace":                    namespace,
		"created_at":                   createdAt.UTC(),
		"phase":                        wfv1.NodePending,
		"parameters":                   string(parametersJSON),
		"is_archived":                  false,
	}

	err = sb.Insert("workflow_executions").
		SetMap(insertMap).
		Suffix("RETURNING id").
		RunWith(tx).
		QueryRow().
		Scan(&newId)

	if err != nil {
		return 0, err
	}
	err = tx.Commit()
	if err != nil {
		return 0, err
	}
	return newId, err
}

func (c *Client) FinishWorkflowExecutionStatisticViaExitHandler(namespace, name string, workflowTemplateID int64, phase wfv1.NodePhase, startedAt time.Time) (err error) {
	tx, err := c.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	updateMap := sq.Eq{
		"started_at":  startedAt.UTC(),
		"name":        name,
		"namespace":   namespace,
		"finished_at": time.Now().UTC(),
		"phase":       phase,
	}

	_, err = sb.Update("workflow_executions").
		SetMap(updateMap).Where(sq.Eq{"name": name}).RunWith(tx).Exec()
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return err
}

func (c *Client) CronStartWorkflowExecutionStatisticInsert(namespace, uid string, workflowTemplateID int64) (err error) {
	query, args, err := c.workflowTemplatesSelectBuilder(namespace).
		Where(sq.Eq{
			"wt.id": workflowTemplateID,
		}).
		ToSql()
	if err != nil {
		return err
	}

	workflowTemplate := &WorkflowTemplate{}
	if err := c.DB.Get(workflowTemplate, query, args...); err != nil {
		return err
	}

	query, args, err = c.cronWorkflowSelectBuilder(namespace, workflowTemplate.UID).ToSql()
	if err != nil {
		return err
	}

	cronWorkflow := &CronWorkflow{}
	if err := c.DB.Get(cronWorkflow, query, args...); err != nil {
		return err
	}

	cronLabels, err := c.GetDbLabels(TypeCronWorkflow, cronWorkflow.ID)
	if err != nil {
		return err
	}

	tx, err := c.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	parametersJSON, err := cronWorkflow.GetParametersFromWorkflowSpecJSON()
	if err != nil {
		return err
	}

	insertMap := sq.Eq{
		"uid":                          uid,
		"workflow_template_version_id": cronWorkflow.WorkflowTemplateVersionID,
		"name":                         uid,
		"namespace":                    namespace,
		"phase":                        wfv1.NodeRunning,
		"started_at":                   time.Now().UTC(),
		"cron_workflow_id":             cronWorkflow.ID,
		"parameters":                   string(parametersJSON),
	}

	workflowExecutionId := uint64(0)
	err = sb.Insert("workflow_executions").
		SetMap(insertMap).
		Suffix("RETURNING id").
		RunWith(tx).
		QueryRow().
		Scan(&workflowExecutionId)
	if err != nil {
		return err
	}

	if len(cronLabels) > 0 {
		labelsMapped := LabelsToMapping(cronLabels...)
		_, err = c.InsertLabelsBuilder(TypeWorkflowExecution, workflowExecutionId, labelsMapped).
			RunWith(tx).
			Exec()
		if err != nil {
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		return err
	}
	return err
}

func (c *Client) GetWorkflowExecution(namespace, uid string) (workflow *WorkflowExecution, err error) {
	workflow = &WorkflowExecution{}

	query, args, err := sb.Select(getWorkflowExecutionColumns("we", "")...).
		Columns(getWorkflowTemplateColumns("wt", "workflow_template")...).
		Columns(`wtv.manifest "workflow_template.manifest"`).
		From("workflow_executions we").
		Join("workflow_template_versions wtv ON wtv.id = we.workflow_template_version_id").
		Join("workflow_templates wt ON wt.id = wtv.workflow_template_id").
		Where(sq.Eq{
			"wt.namespace":   namespace,
			"we.name":        uid,
			"we.is_archived": false,
		}).
		ToSql()
	if err != nil {
		return nil, err
	}
	if err := c.DB.Get(workflow, query, args...); err != nil {
		return nil, err
	}

	wf, err := c.ArgoprojV1alpha1().Workflows(namespace).Get(uid, metav1.GetOptions{})
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace": namespace,
			"UID":       uid,
			"Error":     err.Error(),
		}).Error("Workflow not found.")
		return nil, util.NewUserError(codes.NotFound, "Workflow not found.")
	}

	uidLabel := wf.ObjectMeta.Labels[workflowTemplateUIDLabelKey]
	version, err := strconv.ParseInt(
		wf.ObjectMeta.Labels[workflowTemplateVersionLabelKey],
		10,
		32,
	)
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace": namespace,
			"UID":       uid,
			"Error":     err.Error(),
		}).Error("Invalid version number.")
		return nil, util.NewUserError(codes.InvalidArgument, "Invalid version number.")
	}
	workflowTemplate, err := c.GetWorkflowTemplate(namespace, uidLabel, version)
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace": namespace,
			"UID":       uid,
			"Error":     err.Error(),
		}).Error("Cannot get Workflow Template.")
		return nil, util.NewUserError(codes.NotFound, "Cannot get Workflow Template.")
	}

	manifest, err := json.Marshal(wf)
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace": namespace,
			"UID":       uid,
			"Error":     err.Error(),
		}).Error("Invalid status.")
		return nil, util.NewUserError(codes.InvalidArgument, "Invalid status.")
	}

	workflow.Manifest = string(manifest)
	workflow.WorkflowTemplate = workflowTemplate

	return
}

func (c *Client) ListWorkflowExecutions(namespace, workflowTemplateUID, workflowTemplateVersion string, paginator *pagination.PaginationRequest) (workflows []*WorkflowExecution, err error) {
	sb := workflowExecutionsSelectBuilder(namespace, workflowTemplateUID, workflowTemplateVersion).
		OrderBy("we.created_at DESC")
	sb = *paginator.ApplyToSelect(&sb)

	if err := c.DB.Selectx(&workflows, sb); err != nil {
		return nil, err
	}

	return
}

func (c *Client) CountWorkflowExecutions(namespace, workflowTemplateUID, workflowTemplateVersion string) (count int, err error) {
	err = workflowExecutionsSelectBuilderNoColumns(namespace, workflowTemplateUID, workflowTemplateVersion).
		Columns("COUNT(*)").
		RunWith(c.DB.DB).
		QueryRow().
		Scan(&count)

	return
}

func (c *Client) WatchWorkflowExecution(namespace, uid string) (<-chan *WorkflowExecution, error) {
	_, err := c.GetWorkflowExecution(namespace, uid)
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace": namespace,
			"UID":       uid,
			"Error":     err.Error(),
		}).Errorf("Workflow execution not found for namespace: %v, uid: %v).", namespace, uid)
		return nil, util.NewUserError(codes.NotFound, "Workflow not found.")
	}

	fieldSelector, _ := fields.ParseSelector(fmt.Sprintf("metadata.name=%s", uid))
	watcher, err := c.ArgoprojV1alpha1().Workflows(namespace).Watch(metav1.ListOptions{
		FieldSelector: fieldSelector.String(),
	})
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace": namespace,
			"UID":       uid,
			"Error":     err.Error(),
		}).Error("Watch Workflow error.")
		return nil, util.NewUserError(codes.Unknown, "Error with watching workflow.")
	}

	workflowWatcher := make(chan *WorkflowExecution)
	go func() {
		var next watch.Event
		done := false

		for !done {
			for next = range watcher.ResultChan() {
				workflow, ok := next.Object.(*wfv1.Workflow)
				if !ok {
					done = true
					break
				}
				if workflow == nil {
					continue
				}

				manifest, err := json.Marshal(workflow)
				if err != nil {
					log.WithFields(log.Fields{
						"Namespace": namespace,
						"UID":       uid,
						"Workflow":  workflow,
						"Error":     err.Error(),
					}).Error("Error with trying to JSON Marshal workflow.Status.")
					done = true
					break
				}

				workflowWatcher <- &WorkflowExecution{
					CreatedAt:  workflow.CreationTimestamp.UTC(),
					StartedAt:  ptr.Time(workflow.Status.StartedAt.UTC()),
					FinishedAt: ptr.Time(workflow.Status.FinishedAt.UTC()),
					UID:        string(workflow.UID),
					Name:       workflow.Name,
					Manifest:   string(manifest),
				}

				if !workflow.Status.FinishedAt.IsZero() {
					done = true
					break
				}
			}

			// We want to continue to watch the workflow until it is done, or an error occurred
			// If it is not done, create a new watch and continue watching.
			if !done {
				workflow, err := c.ArgoprojV1alpha1().Workflows(namespace).Get(uid, metav1.GetOptions{})
				if err != nil {
					log.WithFields(log.Fields{
						"Namespace": namespace,
						"UID":       uid,
						"Workflow":  workflow,
						"Error":     err.Error(),
					}).Error("Unable to get workflow.")

					done = true
					break
				}

				if workflow.Status.Phase == wfv1.NodeRunning {
					watcher, err = c.ArgoprojV1alpha1().Workflows(namespace).Watch(metav1.ListOptions{
						FieldSelector: fieldSelector.String(),
					})
					if err != nil {
						log.WithFields(log.Fields{
							"Namespace": namespace,
							"UID":       uid,
							"Error":     err.Error(),
						}).Error("Watch Workflow error.")
						done = true
						break
					}
				} else {
					done = true
					break
				}
			}
		}

		watcher.Stop()
		close(workflowWatcher)
	}()

	return workflowWatcher, nil
}

func (c *Client) GetWorkflowExecutionLogs(namespace, uid, podName, containerName string) (<-chan *LogEntry, error) {
	wf, err := c.ArgoprojV1alpha1().Workflows(namespace).Get(uid, metav1.GetOptions{})
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace":     namespace,
			"UID":           uid,
			"PodName":       podName,
			"ContainerName": containerName,
			"Error":         err.Error(),
		}).Error("Workflow not found.")
		return nil, util.NewUserError(codes.NotFound, "Workflow not found.")
	}

	var (
		stream    io.ReadCloser
		s3Client  *s3.Client
		gcsClient *storage.Client
		config    *NamespaceConfig
		endOffset int
	)

	if wf.Status.Nodes[podName].Completed() {
		config, err = c.GetNamespaceConfig(namespace)
		if err != nil {
			log.WithFields(log.Fields{
				"Namespace":     namespace,
				"UID":           uid,
				"PodName":       podName,
				"ContainerName": containerName,
				"Error":         err.Error(),
			}).Error("Can't get configuration.")
			return nil, util.NewUserError(codes.NotFound, "Can't get configuration.")
		}

		switch {
		case config.ArtifactRepository.S3 != nil:
			{
				s3Client, err = c.GetS3Client(namespace, config.ArtifactRepository.S3)
				if err != nil {
					log.WithFields(log.Fields{
						"Namespace":     namespace,
						"UID":           uid,
						"PodName":       podName,
						"ContainerName": containerName,
						"Error":         err.Error(),
					}).Error("Can't connect to S3 storage.")
					return nil, util.NewUserError(codes.NotFound, "Can't connect to S3 storage.")
				}

				opts := s3.GetObjectOptions{}
				endOffset, err = strconv.Atoi(readEndOffset)
				if err != nil {
					return nil, util.NewUserError(codes.InvalidArgument, "Invalid range.")
				}
				err = opts.SetRange(0, int64(endOffset))
				if err != nil {
					log.WithFields(log.Fields{
						"Namespace":     namespace,
						"UID":           uid,
						"PodName":       podName,
						"ContainerName": containerName,
						"Error":         err.Error(),
					}).Error("Can't set range.")
					return nil, util.NewUserError(codes.NotFound, "Can't connect to S3 storage.")
				}

				key := config.ArtifactRepository.S3.FormatKey(namespace, uid, podName) + "/" + containerName + ".log"
				stream, err = s3Client.GetObject(config.ArtifactRepository.S3.Bucket, key, opts)
			}
		case config.ArtifactRepository.GCS != nil:
			{
				ctx := context.Background()
				gcsClient, err = c.GetGCSClient(namespace, config.ArtifactRepository.GCS)
				if err != nil {
					log.WithFields(log.Fields{
						"Namespace":     namespace,
						"UID":           uid,
						"PodName":       podName,
						"ContainerName": containerName,
						"Error":         err.Error(),
					}).Error("Can't connect to GCS storage.")
					return nil, util.NewUserError(codes.NotFound, "Can't connect to GCS storage.")
				}
				key := config.ArtifactRepository.GCS.FormatKey(namespace, uid, podName) + "/" + containerName + ".log"
				stream, err = gcsClient.Bucket(config.ArtifactRepository.GCS.Bucket).Object(key).NewReader(ctx)
			}
		}
	} else {
		stream, err = c.CoreV1().Pods(namespace).GetLogs(podName, &corev1.PodLogOptions{
			Container:  containerName,
			Follow:     true,
			Timestamps: true,
		}).Stream()
	}
	// TODO: Catch exact kubernetes error
	//Todo: Can above todo be removed with the logging error?
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace":     namespace,
			"UID":           uid,
			"PodName":       podName,
			"ContainerName": containerName,
			"Error":         err.Error(),
		}).Error("Error with logs.")
		return nil, util.NewUserError(codes.NotFound, "Log not found.")
	}

	logWatcher := make(chan *LogEntry)
	go func() {
		scanner := bufio.NewScanner(stream)
		for scanner.Scan() {
			text := scanner.Text()
			parts := strings.Split(text, " ")
			timestamp, err := time.Parse(time.RFC3339, parts[0])
			if err != nil {
				logWatcher <- &LogEntry{Content: text}
			} else {
				logWatcher <- &LogEntry{
					Timestamp: timestamp,
					Content:   strings.Join(parts[1:], " "),
				}
			}

		}
		close(logWatcher)
	}()

	return logWatcher, err
}

func (c *Client) GetWorkflowExecutionMetrics(namespace, uid, podName string) (metrics []*Metric, err error) {
	_, err = c.GetWorkflowExecution(namespace, uid)
	if err != nil {
		return nil, util.NewUserError(codes.NotFound, "Workflow not found.")
	}

	var (
		stream    io.ReadCloser
		s3Client  *s3.Client
		gcsClient *storage.Client
		config    *NamespaceConfig
	)

	config, err = c.GetNamespaceConfig(namespace)
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace": namespace,
			"UID":       uid,
			"PodName":   podName,
			"Error":     err.Error(),
		}).Error("Can't get configuration.")
		return nil, util.NewUserError(codes.NotFound, "Can't get configuration.")
	}

	switch {
	case config.ArtifactRepository.S3 != nil:
		{
			s3Client, err = c.GetS3Client(namespace, config.ArtifactRepository.S3)
			if err != nil {
				log.WithFields(log.Fields{
					"Namespace": namespace,
					"UID":       uid,
					"PodName":   podName,
					"Error":     err.Error(),
				}).Error("Can't connect to S3 storage.")
				return nil, util.NewUserError(codes.NotFound, "Can't connect to S3 storage.")
			}

			opts := s3.GetObjectOptions{}

			key := config.ArtifactRepository.S3.FormatKey(namespace, uid, podName) + "/sys-metrics.json"
			stream, err = s3Client.GetObject(config.ArtifactRepository.S3.Bucket, key, opts)
			if err != nil {
				log.WithFields(log.Fields{
					"Namespace": namespace,
					"UID":       uid,
					"PodName":   podName,
					"Error":     err.Error(),
				}).Error("Metrics do not exist.")
				return nil, util.NewUserError(codes.NotFound, "Metrics do not exist.")
			}
		}
	case config.ArtifactRepository.GCS != nil:
		{
			ctx := context.Background()
			gcsClient, err = c.GetGCSClient(namespace, config.ArtifactRepository.GCS)
			if err != nil {
				log.WithFields(log.Fields{
					"Namespace": namespace,
					"UID":       uid,
					"PodName":   podName,
					"Error":     err.Error(),
				}).Error("Can't connect to GCS storage.")
				return nil, util.NewUserError(codes.NotFound, "Can't connect to GCS storage.")
			}
			key := config.ArtifactRepository.GCS.FormatKey(namespace, uid, podName) + "/sys-metrics.json"
			stream, err = gcsClient.Bucket(config.ArtifactRepository.GCS.Bucket).Object(key).NewReader(ctx)
			if err != nil {
				log.WithFields(log.Fields{
					"Namespace": namespace,
					"UID":       uid,
					"PodName":   podName,
					"Error":     err.Error(),
				}).Error("Metrics do not exist.")
				return nil, util.NewUserError(codes.NotFound, "Metrics do not exist.")
			}
		}
	}

	content, err := ioutil.ReadAll(stream)
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace": namespace,
			"UID":       uid,
			"PodName":   podName,
			"Error":     err.Error(),
		}).Error("Unknown.")
		if strings.Contains("The specified key does not exist.", err.Error()) {
			return nil, util.NewUserError(codes.NotFound, "Metrics were not found.")
		}
		return nil, util.NewUserError(codes.Unknown, "Unknown error.")
	}

	if err = json.Unmarshal(content, &metrics); err != nil {
		log.WithFields(log.Fields{
			"Namespace": namespace,
			"UID":       uid,
			"PodName":   podName,
			"Error":     err.Error(),
		}).Error("Error parsing metrics.")
		return nil, util.NewUserError(codes.InvalidArgument, "Error parsing metrics.")
	}

	return
}

func (c *Client) RetryWorkflowExecution(namespace, uid string) (workflow *WorkflowExecution, err error) {
	wf, err := c.ArgoprojV1alpha1().Workflows(namespace).Get(uid, metav1.GetOptions{})
	if err != nil {
		return
	}

	wf, err = argoutil.RetryWorkflow(c, c.ArgoprojV1alpha1().Workflows(namespace), wf)

	workflow = typeWorkflow(wf)

	return
}

func (c *Client) ResubmitWorkflowExecution(namespace, uid string) (workflow *WorkflowExecution, err error) {
	wf, err := c.ArgoprojV1alpha1().Workflows(namespace).Get(uid, metav1.GetOptions{})
	if err != nil {
		return
	}

	wf, err = argoutil.FormulateResubmitWorkflow(wf, false)
	if err != nil {
		return
	}

	wf, err = argoutil.SubmitWorkflow(c.ArgoprojV1alpha1().Workflows(namespace), c, namespace, wf, &argoutil.SubmitOpts{})
	if err != nil {
		return
	}

	workflow = typeWorkflow(wf)

	return
}

func (c *Client) ResumeWorkflowExecution(namespace, uid string) (workflow *WorkflowExecution, err error) {
	err = argoutil.ResumeWorkflow(c.ArgoprojV1alpha1().Workflows(namespace), uid, "")
	if err != nil {
		return
	}

	wf, err := c.ArgoprojV1alpha1().Workflows(namespace).Get(uid, metav1.GetOptions{})

	workflow = typeWorkflow(wf)

	return
}

func (c *Client) SuspendWorkflowExecution(namespace, uid string) (err error) {
	err = argoutil.SuspendWorkflow(c.ArgoprojV1alpha1().Workflows(namespace), uid)

	return
}

func (c *Client) TerminateWorkflowExecution(namespace, uid string) (err error) {
	query, args, err := sb.Update("workflow_executions").
		Set("phase", "Terminated").
		Set("started_at", time.Time.UTC(time.Now())).
		Set("finished_at", time.Time.UTC(time.Now())).
		Where(sq.Eq{
			"uid":       uid,
			"namespace": namespace,
		}).
		ToSql()

	if err != nil {
		return err
	}

	if _, err := c.DB.Exec(query, args...); err != nil {
		return err
	}

	err = argoutil.TerminateWorkflow(c.ArgoprojV1alpha1().Workflows(namespace), uid)

	return
}

func (c *Client) GetArtifact(namespace, uid, key string) (data []byte, err error) {
	config, err := c.GetNamespaceConfig(namespace)
	if err != nil {
		return
	}
	var (
		stream io.ReadCloser
	)
	switch {
	case config.ArtifactRepository.S3 != nil:
		{
			s3Client, err := c.GetS3Client(namespace, config.ArtifactRepository.S3)
			if err != nil {
				return nil, err
			}

			opts := s3.GetObjectOptions{}
			stream, err = s3Client.GetObject(config.ArtifactRepository.S3.Bucket, key, opts)
			if err != nil {
				log.WithFields(log.Fields{
					"Namespace": namespace,
					"UID":       uid,
					"Key":       key,
					"Error":     err.Error(),
				}).Error("Artifact does not exist.")
				return nil, err
			}
		}
	case config.ArtifactRepository.GCS != nil:
		{
			ctx := context.Background()
			gcsClient, err := c.GetGCSClient(namespace, config.ArtifactRepository.GCS)
			if err != nil {
				log.WithFields(log.Fields{
					"Namespace": namespace,
					"UID":       uid,
					"Error":     err.Error(),
				}).Error("Artifact does not exist.")
				return nil, util.NewUserError(codes.NotFound, "Artifact does not exist.")
			}
			stream, err = gcsClient.Bucket(config.ArtifactRepository.GCS.Bucket).Object(key).NewReader(ctx)
			if err != nil {
				log.WithFields(log.Fields{
					"Namespace": namespace,
					"UID":       uid,
					"Error":     err.Error(),
				}).Error("Artifact does not exist.")
				return nil, util.NewUserError(codes.NotFound, "Artifact does not exist.")
			}
		}
	}

	data, err = ioutil.ReadAll(stream)
	if err != nil {
		return
	}

	return
}

func (c *Client) ListFiles(namespace, key string) (files []*File, err error) {
	config, err := c.GetNamespaceConfig(namespace)
	if err != nil {
		return
	}

	files = make([]*File, 0)

	if len(key) > 0 {
		if string(key[len(key)-1]) != "/" {
			key += "/"
		}
	}
	switch {
	case config.ArtifactRepository.S3 != nil:
		{
			s3Client, err := c.GetS3Client(namespace, config.ArtifactRepository.S3)
			if err != nil {
				return nil, err
			}

			doneCh := make(chan struct{})
			defer close(doneCh)
			for objInfo := range s3Client.ListObjectsV2(config.ArtifactRepository.S3.Bucket, key, false, doneCh) {
				if objInfo.Key == key {
					continue
				}

				isDirectory := (objInfo.ETag == "" || strings.HasSuffix(objInfo.Key, "/")) && objInfo.Size == 0

				newFile := &File{
					Path:         objInfo.Key,
					Name:         FilePathToName(objInfo.Key),
					Extension:    FilePathToExtension(objInfo.Key),
					Size:         objInfo.Size,
					LastModified: objInfo.LastModified,
					ContentType:  objInfo.ContentType,
					Directory:    isDirectory,
				}
				files = append(files, newFile)
			}
		}
	case config.ArtifactRepository.GCS != nil:
		{
			ctx := context.Background()
			gcsClient, err := c.GetGCSClient(namespace, config.ArtifactRepository.GCS)
			if err != nil {
				return nil, err
			}
			q := &storage.Query{
				Delimiter: "",
				Prefix:    key,
				Versions:  false,
			}
			bucketFiles := gcsClient.Bucket(config.ArtifactRepository.GCS.Bucket).Objects(ctx, q)

			//iterate and get files?
			//Check for files are done. todo
			for true { //todo exit condition
				file, err := bucketFiles.Next()
				if err != nil {
					return nil, err
				}
				//todo check if Name or Prefix should be used
				if file.Name == key {
					continue
				}
				isDirectory := (file.Etag == "" || strings.HasSuffix(file.Name, "/")) && file.Size == 0

				newFile := &File{
					Path:         file.Name,
					Name:         FilePathToName(file.Name),
					Extension:    FilePathToExtension(file.Name),
					Size:         file.Size,
					LastModified: file.Updated,
					ContentType:  file.ContentType,
					Directory:    isDirectory,
				}
				files = append(files, newFile)
			}
		}
	}
	return
}

func filterOutCustomTypesFromManifest(manifest []byte) (result []byte, err error) {
	data := make(map[string]interface{})
	err = yaml.Unmarshal(manifest, &data)
	if err != nil {
		return
	}

	spec, ok := data["spec"]
	if !ok {
		return manifest, nil
	}

	specMap, ok := spec.(map[string]interface{})
	if !ok {
		return manifest, nil
	}

	arguments, ok := specMap["arguments"]
	if !ok {
		return manifest, nil
	}

	argumentsMap, ok := arguments.(map[string]interface{})
	if !ok {
		return manifest, nil
	}

	parameters, ok := argumentsMap["parameters"]
	if !ok {
		return manifest, nil
	}

	parametersList, ok := parameters.([]interface{})
	if !ok {
		return manifest, nil
	}

	// We might not want some parameters due to data structuring.
	parametersToKeep := make([]interface{}, 0)

	for _, parameter := range parametersList {
		paramMap, ok := parameter.(map[string]interface{})
		if !ok {
			continue
		}

		// If the parameter does not have a value, skip it so argo doesn't try to process it and fail.
		if _, hasValue := paramMap["value"]; !hasValue {
			paramMap["value"] = "<value>"
		}

		parametersToKeep = append(parametersToKeep, parameter)

		keysToDelete := make([]string, 0)
		for key := range paramMap {
			if key != "name" && key != "value" {
				keysToDelete = append(keysToDelete, key)
			}
		}

		for _, key := range keysToDelete {
			delete(paramMap, key)
		}
	}

	argumentsMap["parameters"] = parametersToKeep

	return yaml.Marshal(data)
}

// prefix is the label prefix.
// e.g. prefix/my-label-key: my-label-value
func (c *Client) GetWorkflowExecutionLabels(namespace, uid, prefix string) (labels map[string]string, err error) {
	wf, err := c.ArgoprojV1alpha1().Workflows(namespace).Get(uid, metav1.GetOptions{})
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace": namespace,
			"UID":       uid,
			"Error":     err.Error(),
		}).Error("Workflow not found.")
		return nil, util.NewUserError(codes.NotFound, "Workflow not found.")
	}

	labels = label.FilterByPrefix(prefix, wf.Labels)
	labels = label.RemovePrefix(prefix, labels)

	return
}

func (c *Client) DeleteWorkflowExecutionLabel(namespace, uid string, keysToDelete ...string) (labels map[string]string, err error) {
	wf, err := c.ArgoprojV1alpha1().Workflows(namespace).Get(uid, metav1.GetOptions{})
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace": namespace,
			"UID":       uid,
			"Error":     err.Error(),
		}).Error("Workflow not found.")
		return nil, util.NewUserError(codes.NotFound, "Workflow not found.")
	}

	label.Delete(wf.Labels, keysToDelete...)

	return wf.Labels, nil
}

func (c *Client) DeleteWorkflowTemplateLabel(namespace, uid string, keysToDelete ...string) (labels map[string]string, err error) {
	wf, err := c.ArgoprojV1alpha1().WorkflowTemplates(namespace).Get(uid, metav1.GetOptions{})
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace": namespace,
			"UID":       uid,
			"Error":     err.Error(),
		}).Error("Workflow Template not found.")
		return nil, util.NewUserError(codes.NotFound, "Workflow Template not found.")
	}

	label.Delete(wf.Labels, keysToDelete...)

	return wf.Labels, nil
}

// prefix is the label prefix.
// we delete all labels with that prefix and set the new ones
// e.g. prefix/my-label-key: my-label-value
func (c *Client) SetWorkflowExecutionLabels(namespace, uid, prefix string, keyValues map[string]string, deleteOld bool) (workflowLabels map[string]string, err error) {
	wf, err := c.ArgoprojV1alpha1().Workflows(namespace).Get(uid, metav1.GetOptions{})
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace": namespace,
			"UID":       uid,
			"Error":     err.Error(),
		}).Error("Workflow not found.")
		return nil, util.NewUserError(codes.NotFound, "Workflow not found.")
	}

	if deleteOld {
		label.DeleteWithPrefix(wf.Labels, prefix)
	}

	label.MergeLabelsPrefix(wf.Labels, keyValues, prefix)

	wf, err = c.ArgoprojV1alpha1().Workflows(namespace).Update(wf)
	if err != nil {
		return nil, err
	}

	filteredMap := label.FilterByPrefix(prefix, wf.Labels)
	filteredMap = label.RemovePrefix(prefix, filteredMap)

	return filteredMap, nil
}

// prefix is the label prefix.
// we delete all labels with that prefix and set the new ones
// e.g. prefix/my-label-key: my-label-value
func (c *Client) SetWorkflowTemplateLabels(namespace, uid, prefix string, keyValues map[string]string, deleteOld bool) (workflowLabels map[string]string, err error) {
	wf, err := c.getArgoWorkflowTemplate(namespace, uid, "latest")
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace": namespace,
			"UID":       uid,
			"Error":     err.Error(),
		}).Error("Workflow Template not found.")
		return nil, util.NewUserError(codes.NotFound, "Workflow Template not found.")
	}

	if deleteOld {
		label.DeleteWithPrefix(wf.Labels, prefix)
	}

	if wf.Labels == nil {
		wf.Labels = make(map[string]string)
	}
	label.MergeLabelsPrefix(wf.Labels, keyValues, prefix)

	wf, err = c.ArgoprojV1alpha1().WorkflowTemplates(namespace).Update(wf)
	if err != nil {
		return nil, err
	}

	filteredMap := label.FilterByPrefix(prefix, wf.Labels)
	filteredMap = label.RemovePrefix(prefix, filteredMap)

	return filteredMap, nil
}

func (c *Client) GetWorkflowExecutionStatisticsForTemplates(workflowTemplates ...*WorkflowTemplate) (err error) {
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
		MAX(we.created_at) last_executed,
		COUNT(*) FILTER (WHERE finished_at IS NULL AND (phase = 'Running' OR phase = 'Pending')) running,
		COUNT(*) FILTER (WHERE finished_at IS NOT NULL AND phase = 'Succeeded') completed,
		COUNT(*) FILTER (WHERE finished_at IS NOT NULL AND (phase = 'Failed' OR phase = 'Error')) failed,
		COUNT(*) FILTER (WHERE phase = 'Terminated') terminated,
		COUNT(*) total`

	query, args, err := sb.Select(statsSelect).
		From("workflow_executions we").
		Join("workflow_template_versions wtv ON wtv.id = we.workflow_template_version_id").
		Where(whereIn, ids...).
		GroupBy("wtv.workflow_template_id").
		ToSql()

	if err != nil {
		return err
	}
	result := make([]*WorkflowExecutionStatisticReport, 0)
	err = c.DB.Select(&result, query, args...)
	if err != nil {
		return err
	}

	resultMapping := make(map[uint64]*WorkflowExecutionStatisticReport)
	for i := range result {
		report := result[i]
		resultMapping[report.WorkflowTemplateId] = report
	}

	for _, workflowTemplate := range workflowTemplates {
		resultMap, ok := resultMapping[workflowTemplate.ID]
		if ok {
			workflowTemplate.WorkflowExecutionStatisticReport = resultMap
		}
	}

	return
}

/**
Will build a template that makes a CURL request to the onepanel-core API,
with statistics about the workflow that was just executed.
*/
func getCURLNodeTemplate(name, curlMethod, curlPath, curlBody string, inputs wfv1.Inputs) (template *wfv1.Template, err error) {
	host := env.GetEnv("ONEPANEL_CORE_SERVICE_HOST", "onepanel-core.onepanel.svc.cluster.local")
	if host == "" {
		err = errors.New("ONEPANEL_CORE_SERVICE_HOST is empty.")
		return
	}
	port := env.GetEnv("ONEPANEL_CORE_SERVICE_PORT", "80")
	if port == "" {
		err = errors.New("ONEPANEL_CORE_SERVICE_PORT is empty.")
		return
	}
	endpoint := fmt.Sprintf("http://%s:%s%s", host, port, curlPath)
	template = &wfv1.Template{
		Name:   name,
		Inputs: inputs,
		Container: &corev1.Container{
			Name:    "curl",
			Image:   "curlimages/curl",
			Command: []string{"sh", "-c"},
			Args: []string{
				"SERVICE_ACCOUNT_TOKEN=$(cat /var/run/secrets/kubernetes.io/serviceaccount/token) " +
					"&& curl -X " + curlMethod + " -s -o /dev/null -w '%{http_code}' " +
					"--connect-timeout 10 --retry 5 --retry-delay 5 " +
					"'" + endpoint + "' -H \"Content-Type: application/json\" -H 'Connection: keep-alive' -H 'Accept: application/json' " +
					"-H 'Authorization: Bearer '\"$SERVICE_ACCOUNT_TOKEN\"'' " +
					"--data '" + curlBody + "' --compressed",
			},
		},
	}
	return
}

func injectExitHandlerWorkflowExecutionStatistic(wf *wfv1.Workflow, workflowTemplateId *uint64) error {
	curlPath := "/apis/v1beta1/{{workflow.namespace}}/workflow_executions/{{workflow.name}}/statistics"
	statistics := map[string]interface{}{
		"workflowStatus":     "{{workflow.status}}",
		"workflowTemplateId": int64(*workflowTemplateId),
	}
	statisticsBytes, err := json.Marshal(statistics)
	if err != nil {
		return err
	}
	statsTemplate, err := getCURLNodeTemplate("sys-send-exit-stats", http.MethodPost, curlPath, string(statisticsBytes), wfv1.Inputs{})
	if err != nil {
		return err
	}

	dagTask := wfv1.DAGTask{
		Name:     statsTemplate.Name,
		Template: statsTemplate.Name,
	}
	wf.Spec.Templates = append(wf.Spec.Templates, *statsTemplate)
	if wf.Spec.OnExit != "" {
		for _, t := range wf.Spec.Templates {
			if t.Name == wf.Spec.OnExit {
				t.DAG.Tasks = append(t.DAG.Tasks, dagTask)

				break
			}
		}
	} else {
		exitHandlerDAG := wfv1.Template{
			Name: "exit-handler",
			DAG: &wfv1.DAGTemplate{
				Tasks: []wfv1.DAGTask{
					dagTask,
				},
			},
		}
		wf.Spec.OnExit = "exit-handler"
		wf.Spec.Templates = append(wf.Spec.Templates, exitHandlerDAG)
	}

	return nil
}

func injectInitHandlerWorkflowExecutionStatistic(wf *wfv1.Workflow, workflowTemplateId *uint64) error {
	curlPath := "/apis/v1beta1/{{workflow.namespace}}/workflow_executions/{{workflow.name}}/cron_start_statistics"
	statistics := map[string]interface{}{
		"workflowTemplateId": int64(*workflowTemplateId),
	}
	statisticsBytes, err := json.Marshal(statistics)
	if err != nil {
		return err
	}
	containerTemplate, err := getCURLNodeTemplate("sys-send-init-stats", http.MethodPost, curlPath, string(statisticsBytes), wfv1.Inputs{})
	if err != nil {
		return err
	}

	// Inject template as entrypoint in DAG
	wf.Spec.Templates = append(wf.Spec.Templates, *containerTemplate)
	for i, t := range wf.Spec.Templates {
		if t.Name == wf.Spec.Entrypoint {
			// DAG is always required for entrypoint templates
			if t.DAG != nil {
				for j, task := range t.DAG.Tasks {
					if task.Dependencies == nil {
						wf.Spec.Templates[i].DAG.Tasks[j].Dependencies = []string{containerTemplate.Name}
						wf.Spec.Templates[i].DAG.Tasks = append(t.DAG.Tasks, wfv1.DAGTask{
							Name:     containerTemplate.Name,
							Template: containerTemplate.Name,
						})
					}
				}
			}
			break
		}
	}

	return nil
}

// injectWorkflowExecutionStatusCaller injects a template that calls a webhook to update execution status
// It injects the template as an entrypoint template and makes the current entrypoint template a dependent.
func injectWorkflowExecutionStatusCaller(wf *wfv1.Workflow, phase wfv1.NodePhase) error {
	curlPath := "/apis/v1beta1/{{workflow.namespace}}/workflow_executions/{{workflow.name}}/status"
	status := WorkflowExecutionStatus{
		Phase: phase,
	}
	statusBytes, err := json.Marshal(status)
	if err != nil {
		return err
	}
	containerTemplate, err := getCURLNodeTemplate("sys-send-status", http.MethodPut, curlPath, string(statusBytes), wfv1.Inputs{})
	if err != nil {
		return err
	}

	// Inject template as entrypoint in DAG
	wf.Spec.Templates = append(wf.Spec.Templates, *containerTemplate)
	for i, t := range wf.Spec.Templates {
		if t.Name == wf.Spec.Entrypoint {
			// DAG is always required for entrypoint templates
			if t.DAG != nil {
				for j, task := range t.DAG.Tasks {
					if task.Dependencies == nil {
						wf.Spec.Templates[i].DAG.Tasks[j].Dependencies = []string{containerTemplate.Name}
						wf.Spec.Templates[i].DAG.Tasks = append(t.DAG.Tasks, wfv1.DAGTask{
							Name:     containerTemplate.Name,
							Template: containerTemplate.Name,
						})
					}
				}
			}
			break
		}
	}

	return nil
}

func workflowExecutionsSelectBuilderNoColumns(namespace, workflowTemplateUID, workflowTemplateVersion string) sq.SelectBuilder {
	whereMap := sq.Eq{
		"wt.namespace":   namespace,
		"wt.uid":         workflowTemplateUID,
		"we.is_archived": false,
	}
	if workflowTemplateVersion != "" {
		whereMap["wtv.version"] = workflowTemplateVersion
	}

	sb := sb.Select().
		From("workflow_executions we").
		LeftJoin("workflow_template_versions wtv ON wtv.id = we.workflow_template_version_id").
		LeftJoin("workflow_templates wt ON wt.id = wtv.workflow_template_id").
		Where(whereMap)

	return sb
}

func workflowExecutionsSelectBuilder(namespace, workflowTemplateUID, workflowTemplateVersion string) sq.SelectBuilder {
	sb := workflowExecutionsSelectBuilderNoColumns(namespace, workflowTemplateUID, workflowTemplateVersion)
	sb = sb.Columns(getWorkflowExecutionColumns("we", "")...).
		Columns(`wtv.version "workflow_template.version"`, `wtv.created_at "workflow_template.created_at"`)

	return sb
}

func (c *Client) getWorkflowExecutionAndTemplate(namespace string, uid string) (workflow *WorkflowExecution, err error) {
	query, args, err := sb.Select(getWorkflowExecutionColumns("we", "")...).
		Columns(getWorkflowTemplateColumns("wt", "workflow_template")...).
		Columns(`wtv.manifest "workflow_template.manifest"`, `wtv.version "workflow_template.version"`).
		From("workflow_executions we").
		Join("workflow_template_versions wtv ON we.workflow_template_version_id = wtv.id").
		Join("workflow_templates wt ON wtv.workflow_template_id = wt.id").
		Where(sq.Eq{
			"wt.namespace":   namespace,
			"we.name":        uid,
			"we.is_archived": false,
		}).
		ToSql()
	if err != nil {
		return nil, err
	}

	// TODO DB call

	workflow = &WorkflowExecution{}
	if err = c.DB.Get(workflow, query, args...); err != nil {
		return nil, err
	}

	workflow.Parameters = make([]Parameter, 0)
	if err := json.Unmarshal(workflow.ParametersBytes, &workflow.Parameters); err != nil {
		return nil, err
	}

	return
}

// UpdateWorkflowExecutionPhase updates workflow execution phases and times.
// `modified_at` time is always updated when this method is called.
func (c *Client) UpdateWorkflowExecutionStatus(namespace, uid string, status *WorkflowExecutionStatus) (err error) {
	fieldMap := sq.Eq{
		"phase": status.Phase,
	}
	switch status.Phase {
	case wfv1.NodeRunning:
		fieldMap["started_at"] = time.Now().UTC()
		break
	}
	_, err = sb.Update("workflow_executions").
		SetMap(fieldMap).
		Where(sq.Eq{
			"namespace": namespace,
			"uid":       uid,
		}).
		RunWith(c.DB).
		Exec()
	if err != nil {
		return util.NewUserError(codes.NotFound, "Workflow execution not found.")
	}

	return
}
