package v1

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	sq "github.com/Masterminds/squirrel"
	"github.com/ghodss/yaml"
	"github.com/hashicorp/go-uuid"
	"github.com/onepanelio/core/api"
	"github.com/onepanelio/core/pkg/util/label"
	"io"
	"io/ioutil"
	"regexp"
	"sort"
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
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
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

	addSecretValsToTemplate := true
	secret, err := c.GetSecret(namespace, "onepanel-default-env")
	if err != nil {
		var statusError *k8serrors.StatusError
		if errors.As(err, &statusError) {
			if statusError.ErrStatus.Reason == "NotFound" {
				addSecretValsToTemplate = false
			} else {
				return err
			}
		} else {
			return err
		}
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

		if !addSecretValsToTemplate {
			continue
		}

		//Generate ENV vars from secret, if there is a container present in the workflow
		//Get template ENV vars, avoid over-writing them with secret values
		for key, value := range secret.Data {
			decodedValue, errDecode := base64.StdEncoding.DecodeString(value)
			if errDecode != nil {
				return errDecode
			}
			addEnvToTemplate(&template, key, string(decodedValue))
		}
		sysConfig, sysErr := c.GetSystemConfig()
		if sysErr != nil {
			return sysErr
		}

		addEnvToTemplate(&template, "ONEPANEL_API_URL", sysConfig["ONEPANEL_API_URL"])
		addEnvToTemplate(&template, "ONEPANEL_HOST", sysConfig["ONEPANEL_HOST"])
		addEnvToTemplate(&template, "PROVIDER_TYPE", sysConfig["PROVIDER_TYPE"])
	}

	return
}

func addEnvToTemplate(template *wfv1.Template, key string, value string) {
	//Flag to prevent over-writing user's envs
	overwriteUserEnv := true
	for _, templateEnv := range template.Container.Env {
		if templateEnv.Name == key {
			overwriteUserEnv = false
			break
		}
	}
	if overwriteUserEnv {
		template.Container.Env = append(template.Container.Env, corev1.EnvVar{
			Name:  key,
			Value: value,
		})
	}
}

func (c *Client) createWorkflow(namespace string, workflowTemplateId *uint64, wf *wfv1.Workflow, opts *WorkflowExecutionOptions) (createdWorkflow *wfv1.Workflow, err error) {
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
		wf.ObjectMeta.Labels = *opts.Labels
	}

	if err = c.injectAutomatedFields(namespace, wf, opts); err != nil {
		return nil, err
	}

	wfExecUid, err := uuid.GenerateUUID()
	if err != nil {
		return nil, err
	}
	err = InjectExitHandlerWorkflowExecutionStatistic(wf, namespace, wfExecUid, workflowTemplateId)
	if err != nil {
		return nil, err
	}

	//Create an entry for workflow_executions statistic
	//CURL code will hit the API endpoint that will update the db row
	err = c.InsertPreWorkflowExecutionStatistic(namespace, wfExecUid, int64(*workflowTemplateId), time.Now())
	if err != nil {
		return nil, err
	}
	createdWorkflow, err = c.ArgoprojV1alpha1().Workflows(namespace).Create(wf)
	if err != nil {
		//Remove the workflow execution statistic entry, because this workflow never ran
		err = c.DeletePreWorkflowExecutionStatistic(wfExecUid)
		if err != nil {
			return nil, err
		}
		return nil, err
	}
	err = c.AddNameToWorkflowExecutionStatisticPostCreation(createdWorkflow.Name, wfExecUid)
	if err != nil {
		return nil, err
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
	}

	return
}

func (c *Client) CreateWorkflowExecution(namespace string, workflow *WorkflowExecution) (*WorkflowExecution, error) {
	workflowTemplate, err := c.GetWorkflowTemplate(namespace, workflow.WorkflowTemplate.UID, workflow.WorkflowTemplate.Version)
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace": namespace,
			"Workflow":  workflow,
			"Error":     err.Error(),
		}).Error("Error with getting workflow template.")
		return nil, util.NewUserError(codes.NotFound, "Error with getting workflow template.")
	}

	// TODO: Need to pull system parameters from k8s config/secret here, example: HOST
	opts := &WorkflowExecutionOptions{}
	re, _ := regexp.Compile(`[^a-zA-Z0-9-]{1,}`)
	opts.GenerateName = strings.ToLower(re.ReplaceAllString(workflowTemplate.Name, `-`)) + "-"
	for _, param := range workflow.Parameters {
		opts.Parameters = append(opts.Parameters, WorkflowExecutionParameter{
			Name:  param.Name,
			Value: param.Value,
		})
	}

	if opts.Labels == nil {
		opts.Labels = &map[string]string{}
	}
	(*opts.Labels)[workflowTemplateUIDLabelKey] = workflowTemplate.UID
	(*opts.Labels)[workflowTemplateVersionLabelKey] = fmt.Sprint(workflowTemplate.Version)
	label.MergeLabelsPrefix(*opts.Labels, workflow.Labels, label.TagPrefix)

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

	var createdWorkflows []*wfv1.Workflow
	for _, wf := range workflows {
		createdWorkflow, err := c.createWorkflow(namespace, &workflowTemplate.ID, &wf, opts)
		if err != nil {
			log.WithFields(log.Fields{
				"Namespace": namespace,
				"Workflow":  workflow,
				"Error":     err.Error(),
			}).Error("Error parsing workflow.")
			return nil, err
		}
		createdWorkflows = append(createdWorkflows, createdWorkflow)
	}

	if createdWorkflows == nil {
		err = errors.New("unable to create workflow")
		log.WithFields(log.Fields{
			"Namespace":        namespace,
			"WorkflowTemplate": workflowTemplate,
			"Error":            err.Error(),
		}).Error("Error parsing workflow.")

		return nil, err
	}

	workflow.Name = createdWorkflows[0].Name
	workflow.CreatedAt = createdWorkflows[0].CreationTimestamp.UTC()
	workflow.UID = string(createdWorkflows[0].ObjectMeta.UID)
	workflow.WorkflowTemplate = workflowTemplate
	// Manifests could get big, don't return them in this case.
	workflow.WorkflowTemplate.Manifest = ""

	return workflow, nil
}

func (c *Client) InsertPreWorkflowExecutionStatistic(namespace, uid string, workflowTemplateID int64, createdAt time.Time) error {
	tx, err := c.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	insertMap := sq.Eq{
		"workflow_template_id": workflowTemplateID,
		"uid":                  uid,
		"name":                 uid,
		"namespace":            namespace,
		"created_at":           createdAt.UTC(),
	}

	_, err = sb.Insert("workflow_executions").
		SetMap(insertMap).RunWith(tx).Exec()
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return err
}

func (c *Client) AddNameToWorkflowExecutionStatisticPostCreation(name, uid string) error {
	tx, err := c.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	updateMap := sq.Eq{
		"name": name,
	}

	_, err = sb.Update("workflow_executions").
		SetMap(updateMap).Where(sq.Eq{"uid": uid}).RunWith(tx).Exec()
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return err
}

func (c *Client) DeletePreWorkflowExecutionStatistic(uid string) error {
	tx, err := c.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = sb.Delete("workflow_executions").Where(
		sq.Eq{"uid": uid}).RunWith(tx).Exec()
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return err
}

func (c *Client) AddWorkflowExecutionStatistic(namespace, name string, workflowTemplateID int64, createdAt time.Time, workflowOutcomeIsSuccess bool) (err error) {
	tx, err := c.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	insertMap := sq.Eq{
		"workflow_template_id": workflowTemplateID,
		"name":                 name,
		"namespace":            namespace,
		"created_at":           createdAt.UTC(),
	}

	if workflowOutcomeIsSuccess {
		insertMap["finished_at"] = time.Now().UTC()
	} else {
		insertMap["failed_at"] = time.Now().UTC()
	}

	_, err = sb.Insert("workflow_executions").
		SetMap(insertMap).RunWith(tx).Exec()
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return err
}

func (c *Client) GetWorkflowExecution(namespace, name string) (workflow *WorkflowExecution, err error) {
	wf, err := c.ArgoprojV1alpha1().Workflows(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace": namespace,
			"Name":      name,
			"Error":     err.Error(),
		}).Error("Workflow not found.")
		return nil, util.NewUserError(codes.NotFound, "Workflow not found.")
	}

	uid := wf.ObjectMeta.Labels[workflowTemplateUIDLabelKey]
	version, err := strconv.ParseInt(
		wf.ObjectMeta.Labels[workflowTemplateVersionLabelKey],
		10,
		32,
	)
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace": namespace,
			"Name":      name,
			"Error":     err.Error(),
		}).Error("Invalid version number.")
		return nil, util.NewUserError(codes.InvalidArgument, "Invalid version number.")
	}
	workflowTemplate, err := c.GetWorkflowTemplate(namespace, uid, int32(version))
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace": namespace,
			"Name":      name,
			"Error":     err.Error(),
		}).Error("Cannot get Workflow Template.")
		return nil, util.NewUserError(codes.NotFound, "Cannot get Workflow Template.")
	}

	// TODO: Do we need to parse parameters into workflow.Parameters?
	manifest, err := json.Marshal(wf)
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace": namespace,
			"Name":      name,
			"Error":     err.Error(),
		}).Error("Invalid status.")
		return nil, util.NewUserError(codes.InvalidArgument, "Invalid status.")
	}
	workflow = &WorkflowExecution{
		UID:              string(wf.UID),
		CreatedAt:        wf.CreationTimestamp.UTC(),
		Name:             wf.Name,
		Phase:            WorkflowExecutionPhase(wf.Status.Phase),
		StartedAt:        wf.Status.StartedAt.UTC(),
		FinishedAt:       wf.Status.FinishedAt.UTC(),
		Manifest:         string(manifest),
		WorkflowTemplate: workflowTemplate,
	}

	return
}

func (c *Client) ListWorkflowExecutions(namespace, workflowTemplateUID, workflowTemplateVersion string) (workflows []*WorkflowExecution, err error) {
	opts := &WorkflowExecutionOptions{}
	if workflowTemplateUID != "" {
		labelSelect := fmt.Sprintf("%s=%s", workflowTemplateUIDLabelKey, workflowTemplateUID)

		if workflowTemplateVersion != "" {
			labelSelect = fmt.Sprintf("%s,%s=%s", labelSelect, workflowTemplateVersionLabelKey, workflowTemplateVersion)
		}

		opts.ListOptions = &ListOptions{
			LabelSelector: labelSelect,
		}
	}
	workflowList, err := c.ArgoprojV1alpha1().Workflows(namespace).List(*opts.ListOptions)
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace":               namespace,
			"WorkflowTemplateUID":     workflowTemplateUID,
			"WorkflowTemplateVersion": workflowTemplateVersion,
			"Error":                   err.Error(),
		}).Error("Workflows not found.")
		return nil, util.NewUserError(codes.NotFound, "Workflows not found.")
	}

	wfs := workflowList.Items
	sort.Slice(wfs, func(i, j int) bool {
		ith := wfs[i].CreationTimestamp.Time
		jth := wfs[j].CreationTimestamp.Time
		//Most recent first
		return ith.After(jth)
	})

	for _, wf := range wfs {
		execution := &WorkflowExecution{
			Name:       wf.ObjectMeta.Name,
			UID:        string(wf.ObjectMeta.UID),
			Phase:      WorkflowExecutionPhase(wf.Status.Phase),
			StartedAt:  wf.Status.StartedAt.UTC(),
			FinishedAt: wf.Status.FinishedAt.UTC(),
			CreatedAt:  wf.CreationTimestamp.UTC(),
		}

		versionString, ok := wf.Labels[workflowTemplateVersionLabelKey]
		if ok {
			versionNumber, err := strconv.Atoi(versionString)
			if err == nil {
				execution.WorkflowTemplate = &WorkflowTemplate{
					Version: int32(versionNumber),
				}
			} else {
				log.WithFields(log.Fields{
					"Namespace":            namespace,
					"WorkflowTemplateUID":  workflowTemplateUID,
					"WorkflowExecutionUID": wf.UID,
					"Error":                "Unable to get workflow template version",
				}).Error("Unable to get workflow template version")
			}
		}

		workflows = append(workflows, execution)
	}

	return
}

func (c *Client) WatchWorkflowExecution(namespace, name string) (<-chan *WorkflowExecution, error) {
	_, err := c.GetWorkflowExecution(namespace, name)
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace": namespace,
			"Name":      name,
			"Error":     err.Error(),
		}).Error("Workflow template not found.")
		return nil, util.NewUserError(codes.NotFound, "Workflow not found.")
	}

	fieldSelector, _ := fields.ParseSelector(fmt.Sprintf("metadata.name=%s", name))
	watcher, err := c.ArgoprojV1alpha1().Workflows(namespace).Watch(metav1.ListOptions{
		FieldSelector: fieldSelector.String(),
	})
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace": namespace,
			"Name":      name,
			"Error":     err.Error(),
		}).Error("Watch Workflow error.")
		return nil, util.NewUserError(codes.Unknown, "Error with watching workflow.")
	}

	workflowWatcher := make(chan *WorkflowExecution)
	ticker := time.NewTicker(time.Second)
	go func() {
		var workflow *wfv1.Workflow
		ok := true

		for {
			select {
			case next := <-watcher.ResultChan():
				workflow, ok = next.Object.(*wfv1.Workflow)
			case <-ticker.C:
			}

			if workflow == nil && ok {
				continue
			}

			if workflow == nil && !ok {
				workflow, err = c.ArgoprojV1alpha1().Workflows(namespace).Get(name, metav1.GetOptions{})
				if err != nil {
					log.WithFields(log.Fields{
						"Namespace": namespace,
						"Name":      name,
						"Workflow":  workflow,
						"Error":     err.Error(),
					}).Error("Unable to get workflow.")

					break
				}

				if workflow == nil {
					break
				}
			}

			manifest, err := json.Marshal(workflow)
			if err != nil {
				log.WithFields(log.Fields{
					"Namespace": namespace,
					"Name":      name,
					"Workflow":  workflow,
					"Error":     err.Error(),
				}).Error("Error with trying to JSON Marshal workflow.Status.")
				break
			}

			workflowWatcher <- &WorkflowExecution{
				CreatedAt:  workflow.CreationTimestamp.UTC(),
				StartedAt:  workflow.Status.StartedAt.UTC(),
				FinishedAt: workflow.Status.FinishedAt.UTC(),
				UID:        string(workflow.UID),
				Name:       workflow.Name,
				Manifest:   string(manifest),
			}

			if !workflow.Status.FinishedAt.IsZero() || !ok {
				break
			}
		}

		close(workflowWatcher)
		watcher.Stop()
		ticker.Stop()
	}()

	return workflowWatcher, nil
}

func (c *Client) GetWorkflowExecutionLogs(namespace, name, podName, containerName string) (<-chan *LogEntry, error) {
	wf, err := c.ArgoprojV1alpha1().Workflows(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace":     namespace,
			"Name":          name,
			"PodName":       podName,
			"ContainerName": containerName,
			"Error":         err.Error(),
		}).Error("Workflow not found.")
		return nil, util.NewUserError(codes.NotFound, "Workflow not found.")
	}

	var (
		stream    io.ReadCloser
		s3Client  *s3.Client
		config    map[string]string
		endOffset int
	)

	if wf.Status.Nodes[podName].Completed() {
		config, err = c.GetNamespaceConfig(namespace)
		if err != nil {
			log.WithFields(log.Fields{
				"Namespace":     namespace,
				"Name":          name,
				"PodName":       podName,
				"ContainerName": containerName,
				"Error":         err.Error(),
			}).Error("Can't get configuration.")
			return nil, util.NewUserError(codes.PermissionDenied, "Can't get configuration.")
		}

		s3Client, err = c.GetS3Client(namespace, config)
		if err != nil {
			log.WithFields(log.Fields{
				"Namespace":     namespace,
				"Name":          name,
				"PodName":       podName,
				"ContainerName": containerName,
				"Error":         err.Error(),
			}).Error("Can't connect to S3 storage.")
			return nil, util.NewUserError(codes.PermissionDenied, "Can't connect to S3 storage.")
		}

		opts := s3.GetObjectOptions{}
		endOffset, err = strconv.Atoi(readEndOffset)
		if err != nil {
			return nil, util.NewUserError(codes.InvalidArgument, "Invaild range.")
		}
		opts.SetRange(0, int64(endOffset))
		stream, err = s3Client.GetObject(config[artifactRepositoryBucketKey], "artifacts/"+namespace+"/"+name+"/"+podName+"/"+containerName+".log", opts)
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
			"Name":          name,
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

func (c *Client) GetWorkflowExecutionMetrics(namespace, name, podName string) (metrics []*Metric, err error) {
	_, err = c.GetWorkflowExecution(namespace, name)
	if err != nil {
		return nil, util.NewUserError(codes.NotFound, "Workflow not found.")
	}

	var (
		stream   io.ReadCloser
		s3Client *s3.Client
		config   map[string]string
	)

	config, err = c.GetNamespaceConfig(namespace)
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace": namespace,
			"Name":      name,
			"PodName":   podName,
			"Error":     err.Error(),
		}).Error("Can't get configuration.")
		return nil, util.NewUserError(codes.PermissionDenied, "Can't get configuration.")
	}

	s3Client, err = c.GetS3Client(namespace, config)
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace": namespace,
			"Name":      name,
			"PodName":   podName,
			"Error":     err.Error(),
		}).Error("Can't connect to S3 storage.")
		return nil, util.NewUserError(codes.PermissionDenied, "Can't connect to S3 storage.")
	}

	opts := s3.GetObjectOptions{}
	stream, err = s3Client.GetObject(config[artifactRepositoryBucketKey], "artifacts/"+namespace+"/"+name+"/"+podName+"/sys-metrics.json", opts)
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace": namespace,
			"Name":      name,
			"PodName":   podName,
			"Error":     err.Error(),
		}).Error("Metrics do not exist.")
		return nil, util.NewUserError(codes.NotFound, "Metrics do not exist.")
	}
	content, err := ioutil.ReadAll(stream)
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace": namespace,
			"Name":      name,
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
			"Name":      name,
			"PodName":   podName,
			"Error":     err.Error(),
		}).Error("Error parsing metrics.")
		return nil, util.NewUserError(codes.InvalidArgument, "Error parsing metrics.")
	}

	return
}

func (c *Client) RetryWorkflowExecution(namespace, name string) (workflow *WorkflowExecution, err error) {
	wf, err := c.ArgoprojV1alpha1().Workflows(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return
	}

	wf, err = argoutil.RetryWorkflow(c, c.ArgoprojV1alpha1().Workflows(namespace), wf)

	workflow = typeWorkflow(wf)

	return
}

func (c *Client) ResubmitWorkflowExecution(namespace, name string) (workflow *WorkflowExecution, err error) {
	wf, err := c.ArgoprojV1alpha1().Workflows(namespace).Get(name, metav1.GetOptions{})
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

func (c *Client) ResumeWorkflowExecution(namespace, name string) (workflow *WorkflowExecution, err error) {
	err = argoutil.ResumeWorkflow(c.ArgoprojV1alpha1().Workflows(namespace), name, "")
	if err != nil {
		return
	}

	wf, err := c.ArgoprojV1alpha1().Workflows(namespace).Get(name, metav1.GetOptions{})

	workflow = typeWorkflow(wf)

	return
}

func (c *Client) SuspendWorkflowExecution(namespace, name string) (err error) {
	err = argoutil.SuspendWorkflow(c.ArgoprojV1alpha1().Workflows(namespace), name)

	return
}

func (c *Client) TerminateWorkflowExecution(namespace, name string) (err error) {
	err = argoutil.TerminateWorkflow(c.ArgoprojV1alpha1().Workflows(namespace), name)

	return
}

func (c *Client) GetArtifact(namespace, name, key string) (data []byte, err error) {
	config, err := c.GetNamespaceConfig(namespace)
	if err != nil {
		return
	}

	s3Client, err := c.GetS3Client(namespace, config)
	if err != nil {
		return
	}

	opts := s3.GetObjectOptions{}
	stream, err := s3Client.GetObject(config[artifactRepositoryBucketKey], key, opts)
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace": namespace,
			"Name":      name,
			"Key":       key,
			"Error":     err.Error(),
		}).Error("Metrics do not exist.")
		return
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

	s3Client, err := c.GetS3Client(namespace, config)
	if err != nil {
		return
	}

	files = make([]*File, 0)

	if len(key) > 0 {
		if string(key[len(key)-1]) != "/" {
			key += "/"
		}
	}

	doneCh := make(chan struct{})
	defer close(doneCh)
	for objInfo := range s3Client.ListObjectsV2(config[artifactRepositoryBucketKey], key, false, doneCh) {
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
			continue
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
func (c *Client) GetWorkflowExecutionLabels(namespace, name, prefix string) (labels map[string]string, err error) {
	wf, err := c.ArgoprojV1alpha1().Workflows(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace": namespace,
			"Name":      name,
			"Error":     err.Error(),
		}).Error("Workflow not found.")
		return nil, util.NewUserError(codes.NotFound, "Workflow not found.")
	}

	labels = label.FilterByPrefix(prefix, wf.Labels)
	labels = label.RemovePrefix(prefix, labels)

	return
}

func (c *Client) DeleteWorkflowExecutionLabel(namespace, name string, keysToDelete ...string) (labels map[string]string, err error) {
	wf, err := c.ArgoprojV1alpha1().Workflows(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace": namespace,
			"Name":      name,
			"Error":     err.Error(),
		}).Error("Workflow not found.")
		return nil, util.NewUserError(codes.NotFound, "Workflow not found.")
	}

	label.Delete(wf.Labels, keysToDelete...)

	return wf.Labels, nil
}

func (c *Client) DeleteWorkflowTemplateLabel(namespace, name string, keysToDelete ...string) (labels map[string]string, err error) {
	wf, err := c.ArgoprojV1alpha1().WorkflowTemplates(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace": namespace,
			"Name":      name,
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
func (c *Client) SetWorkflowExecutionLabels(namespace, name, prefix string, keyValues map[string]string, deleteOld bool) (workflowLabels map[string]string, err error) {
	wf, err := c.ArgoprojV1alpha1().Workflows(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace": namespace,
			"Name":      name,
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
func (c *Client) SetWorkflowTemplateLabels(namespace, name, prefix string, keyValues map[string]string, deleteOld bool) (workflowLabels map[string]string, err error) {
	wf, err := c.getArgoWorkflowTemplate(namespace, name, "latest")
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace": namespace,
			"Name":      name,
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
		return errors.New("GetWorkflowExecutionStatisticsForTemplates requires at least 1 id")
	}

	tx, err := c.DB.Begin()
	if err != nil {
		return err
	}

	whereIn := "workflow_template_id IN (?"
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
		MAX(created_at) last_executed,
		COUNT(*) FILTER (WHERE finished_At IS NULL AND failed_at IS NULL) running,
		COUNT(*) FILTER (WHERE finished_at IS NOT NULL) completed,
		COUNT(*) FILTER (WHERE failed_at IS NOT NULL) failed,
		COUNT(*) total`

	query, args, err := sb.Select(statsSelect).
		From("workflow_executions").
		Where(whereIn, ids...).
		GroupBy("workflow_template_id").
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
		workflowTemplate.WorkflowExecutionStatisticReport = resultMapping[workflowTemplate.ID]
	}

	return
}

/**
Will build a template that makes a CURL request to the onepanel-core API,
with statistics about the workflow that was just executed.
*/
func GetExitHandlerWorkflowStatistics(namespace, uid string, workflowTemplateId *uint64) (workflowStepName, workflowStepTemplate, workflowStepWhen string, err error, wfv1Template wfv1.Template) {
	workflowStepName = "workflow-statistics"
	workflowStepTemplate = "workflow-statistics-template"
	host := env.GetEnv("ONEPANEL_CORE_SERVICE_HOST", "")
	if host == "" {
		err = errors.New("ONEPANEL_CORE_SERVICE_HOST is empty.")
		return
	}
	port := env.GetEnv("ONEPANEL_CORE_SERVICE_PORT", "")
	if port == "" {
		err = errors.New("ONEPANEL_CORE_SERVICE_PORT is empty.")
		return
	}
	curlEndpoint := fmt.Sprintf("http://%s:%s/apis/v1beta1/%s/workflow_executions/{{workflow.name}}/statistics", host, port, namespace)

	jsonRequestStruct := api.Statistics{
		WorkflowStatus:     "{{workflow.status}}",
		Uuid:               uid,
		WorkflowTemplateId: int64(*workflowTemplateId),
	}
	jsonRequestBytes, err := json.Marshal(jsonRequestStruct)
	if err != nil {
		return "", "", "", err, wfv1.Template{}
	}
	jsonRequestStr := string(jsonRequestBytes)
	curlJSONBody := fmt.Sprintf("--data '%s'", jsonRequestStr)

	token, err := GetBearerToken(namespace)
	if err != nil {
		return "", "", "", err, wfv1.Template{}
	}

	wfv1Template = wfv1.Template{
		Name: workflowStepTemplate,
		Container: &corev1.Container{
			Image:   "curlimages/curl",
			Command: []string{"sh", "-c"},
			Args: []string{
				"curl '" + curlEndpoint + "' -H \"Content-Type: application/json\" -H 'Connection: keep-alive' -H 'Accept: application/json' " +
					"-H 'Authorization: Bearer " + token + "' " +
					curlJSONBody + " --compressed",
			},
		},
	}
	return
}

func InjectExitHandlerWorkflowExecutionStatistic(wf *wfv1.Workflow, namespace, wfExecUid string, workflowTemplateId *uint64) error {
	exitHandlerStepName, exitHandlerStepTemplate, exitHandlerStepWhen, err, exitHandlerTemplate := GetExitHandlerWorkflowStatistics(namespace, wfExecUid, workflowTemplateId)
	if err != nil {
		return err
	}
	if exitHandlerStepTemplate != "" {
		exitHandler := wfv1.Template{
			Name: "exit-handler",
			Steps: []wfv1.ParallelSteps{
				{
					Steps: []wfv1.WorkflowStep{
						{Name: exitHandlerStepName, Template: exitHandlerStepTemplate, When: exitHandlerStepWhen},
					},
				},
			},
		}
		wf.Spec.OnExit = "exit-handler"
		wf.Spec.Templates = append(wf.Spec.Templates, exitHandler, exitHandlerTemplate)
	}
	return nil
}
