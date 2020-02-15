package manager

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/onepanelio/core/util/logging"
	log "github.com/sirupsen/logrus"

	"github.com/onepanelio/core/kube"
	"github.com/onepanelio/core/model"
	"github.com/onepanelio/core/s3"
	"github.com/onepanelio/core/util"
	"github.com/onepanelio/core/util/env"
	"google.golang.org/grpc/codes"
)

var (
	readEndOffset                   = env.GetEnv("ARTIFACT_RERPOSITORY_OBJECT_RANGE", "-102400")
	workflowTemplateUIDLabelKey     = labelKeyPrefix + "workflow-template-uid"
	workflowTemplateVersionLabelKey = labelKeyPrefix + "workflow-template-version"
)

func (r *ResourceManager) CreateWorkflow(namespace string, workflow *model.Workflow) (*model.Workflow, error) {
	workflowTemplate, err := r.GetWorkflowTemplate(namespace, workflow.WorkflowTemplate.UID, workflow.WorkflowTemplate.Version)
	if err != nil {
		logging.Logger.Log.WithFields(log.Fields{
			"Namespace": namespace,
			"Workflow":  workflow,
			"Error":     err.Error(),
		}).Error("Error with getting workflow template.")
		return nil, util.NewUserError(codes.NotFound, "Error with getting workflow template.")
	}

	// TODO: Need to pull system parameters from k8s config/secret here, example: HOST
	opts := &kube.WorkflowOptions{}
	re, _ := regexp.Compile(`[^a-zA-Z0-9-]{1,}`)
	opts.GenerateName = strings.ToLower(re.ReplaceAllString(workflowTemplate.Name, `-`)) + "-"
	for _, param := range workflow.Parameters {
		opts.Parameters = append(opts.Parameters, kube.WorkflowParameter{
			Name:  param.Name,
			Value: param.Value,
		})
	}
	if opts.Labels == nil {
		opts.Labels = &map[string]string{}
	}
	(*opts.Labels)[workflowTemplateUIDLabelKey] = workflowTemplate.UID
	(*opts.Labels)[workflowTemplateVersionLabelKey] = fmt.Sprint(workflowTemplate.Version)
	createdWorkflows, err := r.NewKubeClient().CreateWorkflow(namespace, workflowTemplate.GetManifestBytes(), opts)
	if err != nil {
		logging.Logger.Log.WithFields(log.Fields{
			"Namespace": namespace,
			"Workflow":  workflow,
			"Error":     err.Error(),
		}).Error("Error creating workflow.")
		return nil, util.NewUserError(codes.Unknown, "Error creating workflow.")
	}

	workflow.Name = createdWorkflows[0].Name
	workflow.CreatedAt = createdWorkflows[0].CreationTimestamp.UTC()
	workflow.UID = string(createdWorkflows[0].ObjectMeta.UID)
	workflow.WorkflowTemplate = workflowTemplate
	// Manifests could get big, don't return them in this case.
	workflow.WorkflowTemplate.Manifest = ""

	return workflow, nil
}

func (r *ResourceManager) GetWorkflow(namespace, name string) (workflow *model.Workflow, err error) {
	wf, err := r.NewKubeClient().GetWorkflow(namespace, name)
	if err != nil {
		logging.Logger.Log.WithFields(log.Fields{
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
		logging.Logger.Log.WithFields(log.Fields{
			"Namespace": namespace,
			"Name":      name,
			"Error":     err.Error(),
		}).Error("Invalid version number.")
		return nil, util.NewUserError(codes.InvalidArgument, "Invalid version number.")
	}
	workflowTemplate, err := r.GetWorkflowTemplate(namespace, uid, int32(version))
	if err != nil {
		logging.Logger.Log.WithFields(log.Fields{
			"Namespace": namespace,
			"Name":      name,
			"Error":     err.Error(),
		}).Error("Cannot get Workflow Template.")
		return nil, util.NewUserError(codes.NotFound, "Cannot get Workflow Template.")
	}

	// TODO: Do we need to parse parameters into workflow.Parameters?
	manifest, err := json.Marshal(wf)
	if err != nil {
		logging.Logger.Log.WithFields(log.Fields{
			"Namespace": namespace,
			"Name":      name,
			"Error":     err.Error(),
		}).Error("Invalid status.")
		return nil, util.NewUserError(codes.InvalidArgument, "Invalid status.")
	}
	workflow = &model.Workflow{
		UID:              string(wf.UID),
		CreatedAt:        wf.CreationTimestamp.UTC(),
		Name:             wf.Name,
		Phase:            model.WorkflowPhase(wf.Status.Phase),
		StartedAt:        wf.Status.StartedAt.UTC(),
		FinishedAt:       wf.Status.FinishedAt.UTC(),
		Manifest:         string(manifest),
		WorkflowTemplate: workflowTemplate,
	}

	return
}

func (r *ResourceManager) WatchWorkflow(namespace, name string) (<-chan *model.Workflow, error) {
	_, err := r.GetWorkflow(namespace, name)
	if err != nil {
		logging.Logger.Log.WithFields(log.Fields{
			"Namespace": namespace,
			"Name":      name,
			"Error":     err.Error(),
		}).Error("Workflow template not found.")
		return nil, util.NewUserError(codes.NotFound, "Workflow not found.")
	}

	watcher, err := r.NewKubeClient().WatchWorkflow(namespace, name)
	if err != nil {
		logging.Logger.Log.WithFields(log.Fields{
			"Namespace": namespace,
			"Name":      name,
			"Error":     err.Error(),
		}).Error("Watch Workflow error.")
		return nil, util.NewUserError(codes.Unknown, "Error with watching workflow.")
	}

	var workflow *kube.Workflow
	workflowWatcher := make(chan *model.Workflow)
	ticker := time.NewTicker(time.Second)
	go func() {
		for {
			select {
			case next := <-watcher.ResultChan():
				workflow, _ = next.Object.(*kube.Workflow)
			case <-ticker.C:
			}

			if workflow == nil {
				continue
			}
			manifest, err := json.Marshal(workflow)
			if err != nil {
				logging.Logger.Log.WithFields(log.Fields{
					"Namespace": namespace,
					"Name":      name,
					"Workflow":  workflow,
					"Error":     err.Error(),
				}).Error("Error with trying to JSON Marshal workflow.Status.")
				continue
			}
			workflowWatcher <- &model.Workflow{
				UID:      string(workflow.UID),
				Name:     workflow.Name,
				Manifest: string(manifest),
			}

			if !workflow.Status.FinishedAt.IsZero() {
				break
			}
		}
		close(workflowWatcher)
		watcher.Stop()
	}()

	return workflowWatcher, nil
}

func (r *ResourceManager) GetWorkflowLogs(namespace, name, podName, containerName string) (<-chan *model.LogEntry, error) {
	wf, err := r.NewKubeClient().GetWorkflow(namespace, name)
	if err != nil {
		logging.Logger.Log.WithFields(log.Fields{
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
		config, err = r.getNamespaceConfig(namespace)
		if err != nil {
			logging.Logger.Log.WithFields(log.Fields{
				"Namespace":     namespace,
				"Name":          name,
				"PodName":       podName,
				"ContainerName": containerName,
				"Error":         err.Error(),
			}).Error("Can't get configuration.")
			return nil, util.NewUserError(codes.PermissionDenied, "Can't get configuration.")
		}

		s3Client, err = r.getS3Client(namespace, config)
		if err != nil {
			logging.Logger.Log.WithFields(log.Fields{
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
		stream, err = r.NewKubeClient().GetPodLogs(namespace, podName, containerName)
	}
	// TODO: Catch exact kubernetes error
	//Todo: Can above todo be removed with the logging error?
	if err != nil {
		logging.Logger.Log.WithFields(log.Fields{
			"Namespace":     namespace,
			"Name":          name,
			"PodName":       podName,
			"ContainerName": containerName,
			"Error":         err.Error(),
		}).Error("Error with logs.")
		return nil, util.NewUserError(codes.NotFound, "Log not found.")
	}

	logWatcher := make(chan *model.LogEntry)
	go func() {
		scanner := bufio.NewScanner(stream)
		for scanner.Scan() {
			text := scanner.Text()
			parts := strings.Split(text, " ")
			timestamp, err := time.Parse(time.RFC3339, parts[0])
			if err != nil {
				logWatcher <- &model.LogEntry{Content: text}
			} else {
				logWatcher <- &model.LogEntry{
					Timestamp: timestamp,
					Content:   strings.Join(parts[1:], " "),
				}
			}

		}
		close(logWatcher)
	}()

	return logWatcher, err
}

func (r *ResourceManager) GetWorkflowMetrics(namespace, name, podName string) (metrics []*model.Metric, err error) {
	_, err = r.NewKubeClient().GetWorkflow(namespace, name)
	if err != nil {
		return nil, util.NewUserError(codes.NotFound, "Workflow not found.")
	}

	var (
		stream   io.ReadCloser
		s3Client *s3.Client
		config   map[string]string
	)

	config, err = r.getNamespaceConfig(namespace)
	if err != nil {
		logging.Logger.Log.WithFields(log.Fields{
			"Namespace": namespace,
			"Name":      name,
			"PodName":   podName,
			"Error":     err.Error(),
		}).Error("Can't get configuration.")
		return nil, util.NewUserError(codes.PermissionDenied, "Can't get configuration.")
	}

	s3Client, err = r.getS3Client(namespace, config)
	if err != nil {
		logging.Logger.Log.WithFields(log.Fields{
			"Namespace": namespace,
			"Name":      name,
			"PodName":   podName,
			"Error":     err.Error(),
		}).Error("Can't connect to S3 storage.")
		return nil, util.NewUserError(codes.PermissionDenied, "Can't connect to S3 storage.")
	}

	opts := s3.GetObjectOptions{}
	stream, err = s3Client.GetObject(config[artifactRepositoryBucketKey], "artifacts/"+namespace+"/"+name+"/"+podName+"/metrics.json", opts)
	if err != nil {
		logging.Logger.Log.WithFields(log.Fields{
			"Namespace": namespace,
			"Name":      name,
			"PodName":   podName,
			"Error":     err.Error(),
		}).Error("Metrics do not exist.")
		return nil, util.NewUserError(codes.NotFound, "Metrics do not exist.")
	}
	content, err := ioutil.ReadAll(stream)
	if err != nil {
		logging.Logger.Log.WithFields(log.Fields{
			"Namespace": namespace,
			"Name":      name,
			"PodName":   podName,
			"Error":     err.Error(),
		}).Error("Unknown.")
		return nil, util.NewUserError(codes.Unknown, "Unknown error.")
	}

	if err = json.Unmarshal(content, &metrics); err != nil {
		logging.Logger.Log.WithFields(log.Fields{
			"Namespace": namespace,
			"Name":      name,
			"PodName":   podName,
			"Error":     err.Error(),
		}).Error("Error parsing metrics.")
		return nil, util.NewUserError(codes.InvalidArgument, "Error parsing metrics.")
	}

	return
}

func (r *ResourceManager) ListWorkflows(namespace, workflowTemplateUID, workflowTemplateVersion string) (workflows []*model.Workflow, err error) {
	opts := &kube.WorkflowOptions{}
	if workflowTemplateUID != "" {
		labelSelect := fmt.Sprintf("%s=%s", workflowTemplateUIDLabelKey, workflowTemplateUID)

		if workflowTemplateVersion != "" {
			labelSelect = fmt.Sprintf("%s,%s=%s", labelSelect, workflowTemplateVersionLabelKey, workflowTemplateVersion)
		}

		opts.ListOptions = &kube.ListOptions{
			LabelSelector: labelSelect,
		}
	}
	wfs, err := r.NewKubeClient().ListWorkflows(namespace, opts)
	if err != nil {
		logging.Logger.Log.WithFields(log.Fields{
			"Namespace":               namespace,
			"WorkflowTemplateUID":     workflowTemplateUID,
			"WorkflowTemplateVersion": workflowTemplateVersion,
			"Error":                   err.Error(),
		}).Error("Workflows not found.")
		return nil, util.NewUserError(codes.NotFound, "Workflows not found.")
	}
	sort.Slice(wfs, func(i, j int) bool {
		ith := wfs[i].CreationTimestamp.Time
		jth := wfs[j].CreationTimestamp.Time
		//Most recent first
		return ith.After(jth)
	})

	for _, wf := range wfs {
		workflows = append(workflows, &model.Workflow{
			Name:       wf.ObjectMeta.Name,
			UID:        string(wf.ObjectMeta.UID),
			Phase:      model.WorkflowPhase(wf.Status.Phase),
			StartedAt:  wf.Status.StartedAt.UTC(),
			FinishedAt: wf.Status.FinishedAt.UTC(),
			CreatedAt:  wf.CreationTimestamp.UTC(),
		})
	}

	return
}

func (r *ResourceManager) ResubmitWorkflow(namespace, name string) (workflow *model.Workflow, err error) {
	workflow, err = r.NewKubeClient().ResubmitWorkflow(namespace, name, false)
	if err != nil {
		return nil, util.NewUserError(codes.Unknown, "Could not resubmit workflow.")
	}

	return
}

func (r *ResourceManager) TerminateWorkflow(namespace, name string) (err error) {
	if err = r.NewKubeClient().TerminateWorkflow(namespace, name); err != nil {
		logging.Logger.Log.WithFields(log.Fields{
			"Namespace": namespace,
			"Name":      name,
			"Error":     err.Error(),
		}).Error("Could not terminate workflow.")
		return util.NewUserError(codes.Unknown, "Could not terminate workflow.")
	}

	return
}

func (r *ResourceManager) CreateWorkflowTemplate(namespace string, workflowTemplate *model.WorkflowTemplate) (*model.WorkflowTemplate, error) {
	allowed, err := r.NewKubeClient().IsAuthorized(namespace, "create", "argoproj.io", "workflow", "")
	if err != nil {
		logging.Logger.Log.WithFields(log.Fields{
			"Namespace":        namespace,
			"WorkflowTemplate": workflowTemplate,
			"Error":            err.Error(),
		}).Error("IsAuthorized failed.")
		return nil, util.NewUserError(codes.Unknown, "Could not create workflow template.")
	}
	if !allowed {
		return nil, util.NewUserError(codes.PermissionDenied, "Permission denied.")
	}

	// validate workflow template
	if err := r.NewKubeClient().ValidateWorkflow(namespace, workflowTemplate.GetManifestBytes()); err != nil {
		logging.Logger.Log.WithFields(log.Fields{
			"Namespace":        namespace,
			"WorkflowTemplate": workflowTemplate,
			"Error":            err.Error(),
		}).Error("Workflow could not be validated.")
		return nil, util.NewUserError(codes.InvalidArgument, err.Error())
	}

	workflowTemplate, err = r.workflowRepository.CreateWorkflowTemplate(namespace, workflowTemplate)
	if err != nil {
		logging.Logger.Log.WithFields(log.Fields{
			"Namespace":        namespace,
			"WorkflowTemplate": workflowTemplate,
			"Error":            err.Error(),
		}).Error("Could not create workflow template.")
		return nil, util.NewUserErrorWrap(err, "Workflow template")
	}

	return workflowTemplate, nil
}

func (r *ResourceManager) CreateWorkflowTemplateVersion(namespace string, workflowTemplate *model.WorkflowTemplate) (*model.WorkflowTemplate, error) {
	allowed, err := r.NewKubeClient().IsAuthorized(namespace, "create", "argoproj.io", "workflow", "")
	if err != nil {
		logging.Logger.Log.WithFields(log.Fields{
			"Namespace":        namespace,
			"WorkflowTemplate": workflowTemplate,
			"Error":            err.Error(),
		}).Error("IsAuthorized failed.")
		return nil, util.NewUserError(codes.Unknown, "Could not create template version.")
	}
	if !allowed {
		return nil, util.NewUserError(codes.PermissionDenied, "Permission denied.")
	}

	// validate workflow template
	if err := r.NewKubeClient().ValidateWorkflow(namespace, workflowTemplate.GetManifestBytes()); err != nil {
		logging.Logger.Log.WithFields(log.Fields{
			"Namespace":        namespace,
			"WorkflowTemplate": workflowTemplate,
			"Error":            err.Error(),
		}).Error("Workflow could not be validated.")
		return nil, util.NewUserError(codes.InvalidArgument, err.Error())
	}

	if err := r.workflowRepository.RemoveIsLatestFromWorkflowTemplateVersions(workflowTemplate); err != nil {
		logging.Logger.Log.WithFields(log.Fields{
			"Namespace":        namespace,
			"WorkflowTemplate": workflowTemplate,
			"Error":            err.Error(),
		}).Error("Could not remove IsLatest from workflow template versions.")
		return nil, util.NewUserError(codes.Unknown, "Unable to Create Workflow Template Version.")
	}

	workflowTemplate, err = r.workflowRepository.CreateWorkflowTemplateVersion(namespace, workflowTemplate)
	if err != nil {
		logging.Logger.Log.WithFields(log.Fields{
			"Namespace":        namespace,
			"WorkflowTemplate": workflowTemplate,
			"Error":            err.Error(),
		}).Error("Could not create workflow template version.")
		return nil, util.NewUserErrorWrap(err, "Workflow template")
	}
	if workflowTemplate == nil {
		return nil, util.NewUserError(codes.NotFound, "Workflow template not found.")
	}

	return workflowTemplate, nil
}

func (r *ResourceManager) UpdateWorkflowTemplateVersion(namespace string, workflowTemplate *model.WorkflowTemplate) (*model.WorkflowTemplate, error) {
	allowed, err := r.NewKubeClient().IsAuthorized(namespace, "update", "argoproj.io", "workflow", "")
	if err != nil {
		logging.Logger.Log.WithFields(log.Fields{
			"Namespace":        namespace,
			"WorkflowTemplate": workflowTemplate,
			"Error":            err.Error(),
		}).Error("IsAuthorized failed.")
		return nil, util.NewUserError(codes.Unknown, "Could not update workflow template version.")
	}
	if !allowed {
		return nil, util.NewUserError(codes.PermissionDenied, "Permission denied.")
	}

	// validate workflow template
	if err := r.NewKubeClient().ValidateWorkflow(namespace, workflowTemplate.GetManifestBytes()); err != nil {
		logging.Logger.Log.WithFields(log.Fields{
			"Namespace":        namespace,
			"WorkflowTemplate": workflowTemplate,
			"Error":            err.Error(),
		}).Error("Workflow could not be validated.")
		return nil, util.NewUserError(codes.InvalidArgument, err.Error())
	}

	originalWorkflowTemplate, err := r.workflowRepository.GetWorkflowTemplate(namespace, workflowTemplate.UID, workflowTemplate.Version)
	if err != nil {
		logging.Logger.Log.WithFields(log.Fields{
			"Namespace":        namespace,
			"WorkflowTemplate": workflowTemplate,
			"Error":            err.Error(),
		}).Error("Could not get workflow template.")
		return nil, util.NewUserError(codes.Unknown, "Could not update workflow template version.")
	}

	workflowTemplate.ID = originalWorkflowTemplate.ID
	workflowTemplate, err = r.workflowRepository.UpdateWorkflowTemplateVersion(workflowTemplate)
	if err != nil {
		logging.Logger.Log.WithFields(log.Fields{
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

func (r *ResourceManager) GetWorkflowTemplate(namespace, uid string, version int32) (workflowTemplate *model.WorkflowTemplate, err error) {
	allowed, err := r.NewKubeClient().IsAuthorized(namespace, "get", "argoproj.io", "workflow", "")
	if err != nil {
		logging.Logger.Log.WithFields(log.Fields{
			"Namespace":        namespace,
			"WorkflowTemplate": workflowTemplate,
			"Error":            err.Error(),
		}).Error("IsAuthorized failed.")
		return nil, util.NewUserError(codes.Unknown, "Unknown error.")
	}
	if !allowed {
		return nil, util.NewUserError(codes.PermissionDenied, "Permission denied.")
	}

	workflowTemplate, err = r.workflowRepository.GetWorkflowTemplate(namespace, uid, version)
	if err != nil {
		logging.Logger.Log.WithFields(log.Fields{
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

func (r *ResourceManager) ListWorkflowTemplateVersions(namespace, uid string) (workflowTemplateVersions []*model.WorkflowTemplate, err error) {
	allowed, err := r.NewKubeClient().IsAuthorized(namespace, "list", "argoproj.io", "workflow", "")
	if err != nil {
		logging.Logger.Log.WithFields(log.Fields{
			"Namespace": namespace,
			"UID":       uid,
			"Error":     err.Error(),
		}).Error("IsAuthorized failed.")
		return nil, util.NewUserError(codes.Unknown, "Unknown error.")
	}
	if !allowed {
		return nil, util.NewUserError(codes.PermissionDenied, "Permission denied.")
	}

	workflowTemplateVersions, err = r.workflowRepository.ListWorkflowTemplateVersions(namespace, uid)
	if err != nil {
		logging.Logger.Log.WithFields(log.Fields{
			"Namespace": namespace,
			"UID":       uid,
			"Error":     err.Error(),
		}).Error("Workflow template versions not found.")
		return nil, util.NewUserError(codes.NotFound, "Workflow template versions not found.")
	}

	return
}

func (r *ResourceManager) ListWorkflowTemplates(namespace string) (workflowTemplateVersions []*model.WorkflowTemplate, err error) {
	allowed, err := r.NewKubeClient().IsAuthorized(namespace, "list", "argoproj.io", "workflow", "")
	if err != nil {
		logging.Logger.Log.WithFields(log.Fields{
			"Namespace": namespace,
			"Error":     err.Error(),
		}).Error("IsAuthorized failed.")
		return nil, util.NewUserError(codes.Unknown, "Unable to list workflow templates.")
	}
	if !allowed {
		return nil, util.NewUserError(codes.PermissionDenied, "Permission denied.")
	}

	workflowTemplateVersions, err = r.workflowRepository.ListWorkflowTemplates(namespace)
	if err != nil {
		logging.Logger.Log.WithFields(log.Fields{
			"Namespace": namespace,
			"Error":     err.Error(),
		}).Error("Workflow templates not found.")
		return nil, util.NewUserError(codes.NotFound, "Workflow templates not found.")
	}

	return
}

func (r *ResourceManager) ArchiveWorkflowTemplate(namespace, uid string) (archived bool, err error) {
	allowed, err := r.NewKubeClient().IsAuthorized(namespace, "delete", "argoproj.io", "workflow", "")
	if err != nil {
		logging.Logger.Log.WithFields(log.Fields{
			"Namespace": namespace,
			"UID":       uid,
			"Error":     err.Error(),
		}).Error("IsAuthorized failed.")
		return false, util.NewUserError(codes.Unknown, "Unable to archive workflow template.")
	}
	if !allowed {
		return false, util.NewUserError(codes.PermissionDenied, "Permission denied.")
	}

	workflowTemplate, err := r.workflowRepository.GetWorkflowTemplate(namespace, uid, 0)
	if err != nil {
		logging.Logger.Log.WithFields(log.Fields{
			"Namespace": namespace,
			"UID":       uid,
			"Error":     err.Error(),
		}).Error("Get Workflow Template failed.")
		return false, util.NewUserError(codes.Unknown, "Unable to archive workflow template.")
	}
	if workflowTemplate == nil {
		return false, util.NewUserError(codes.NotFound, "Workflow template not found.")
	}

	archived, err = r.workflowRepository.ArchiveWorkflowTemplate(namespace, uid)
	if !archived || err != nil {
		if err != nil {
			logging.Logger.Log.WithFields(log.Fields{
				"Namespace": namespace,
				"UID":       uid,
				"Error":     err.Error(),
			}).Error("Archive Workflow Template failed.")
		}
		return false, util.NewUserError(codes.Unknown, "Unable to archive workflow template.")
	}

	return
}
