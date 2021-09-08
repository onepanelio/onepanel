package v1

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/argoproj/argo/persist/sqldb"
	"github.com/argoproj/argo/workflow/hydrator"
	"github.com/google/uuid"
	"github.com/onepanelio/core/pkg/util/gcs"
	"github.com/onepanelio/core/pkg/util/label"
	"github.com/onepanelio/core/pkg/util/ptr"
	"github.com/onepanelio/core/pkg/util/request"
	"github.com/onepanelio/core/pkg/util/types"
	uid2 "github.com/onepanelio/core/pkg/util/uid"
	"gopkg.in/yaml.v2"
	networking "istio.io/api/networking/v1alpha3"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/watch"
	yaml2 "sigs.k8s.io/yaml"

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

// envVarValueInSidecars returns true if any of the sidecars contain an environment variable with the input name and value
// false otherwise
func envVarValueInSidecars(sidecars []wfv1.UserContainer, name, value string) bool {
	for _, s := range sidecars {
		for _, e := range s.Env {
			if e.Name == name && e.Value == value {
				return true
			}
		}
	}

	return false
}

// hasEnvVarValue returns true if any of the env vars have the given name and value
// false otherwise
func hasEnvVarValue(envVars []corev1.EnvVar, name, value string) bool {
	for _, e := range envVars {
		if e.Name == name && e.Value == value {
			return true
		}
	}

	return false
}

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

// WorkflowExecutionFilter represents the available ways we can filter WorkflowExecutions
type WorkflowExecutionFilter struct {
	Labels []*Label
	Phase  string // empty string means none
}

// GetLabels returns the labels in the filter
func (wf *WorkflowExecutionFilter) GetLabels() []*Label {
	return wf.Labels
}

func applyWorkflowExecutionFilter(sb sq.SelectBuilder, request *request.Request) (sq.SelectBuilder, error) {
	if !request.HasFilter() {
		return sb, nil
	}

	filter, ok := request.Filter.(WorkflowExecutionFilter)
	if !ok {
		return sb, nil
	}

	// template, name are reserved labels.
	// we query the columns on the appropriate tables instead
	finalLabels := make([]*Label, 0)
	for _, label := range filter.Labels {
		if label.Key == "template" {
			sb = sb.Where(sq.And{
				sq.Expr("wt.name ILIKE ?", "%"+label.Value+"%"),
			})
		} else if label.Key == "name" {
			sb = sb.Where(sq.And{
				sq.Expr("we.name ILIKE ?", "%"+label.Value+"%"),
			})
		} else {
			finalLabels = append(finalLabels, label)
		}
	}
	filter.Labels = finalLabels

	sb, err := ApplyLabelSelectQuery("we.labels", sb, &filter)
	if err != nil {
		return sb, err
	}

	switch filter.Phase {
	case "":
		return sb, nil
	case "running":
		sb = sb.Where(sq.Eq{
			"we.finished_at": nil,
			"we.phase":       []string{"Running", "Pending"},
		})
	case "completed":
		sb = sb.Where(sq.NotEq{
			"we.finished_at": nil,
		}).Where(sq.Eq{
			"we.phase": "Succeeded",
		})
	case "failed":
		sb = sb.Where(sq.NotEq{
			"we.finished_at": nil,
		}).Where(sq.Eq{
			"we.phase": []string{"Failed", "Error"},
		})
	case "stopped":
		sb = sb.Where(sq.Eq{
			"we.phase": "Terminated",
		})
	default:
		return sb, fmt.Errorf("unknown workflow execution phase filter '%v'", filter.Phase)
	}

	return sb, nil
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

// getWorkflowsFromWorkflowTemplate parses the WorkflowTemplate manifest and returns the argo workflows from it
func getWorkflowsFromWorkflowTemplate(wt *WorkflowTemplate) (wfs []wfv1.Workflow, err error) {
	manifest, err := wt.GetWorkflowManifestBytes()
	if err != nil {
		return nil, err
	}

	wfs, err = UnmarshalWorkflows(manifest, true)

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
		artifact.S3.SecretKeySecret = corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: s3Config.SecretKeySecret.Name,
			},
			Key: s3Config.SecretKeySecret.Key,
		}
		artifact.S3.AccessKeySecret = corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: s3Config.AccessKeySecret.Name,
			},
			Key: s3Config.AccessKeySecret.Key,
		}
	}

	if artifact.GCS != nil && namespaceConfig.ArtifactRepository.GCS != nil {
		gcsConfig := namespaceConfig.ArtifactRepository.GCS
		artifact.GCS.Bucket = gcsConfig.Bucket
		artifact.GCS.Key = gcsConfig.KeyFormat
		artifact.GCS.ServiceAccountKeySecret.Name = "onepanel"
		artifact.GCS.ServiceAccountKeySecret.Key = "artifactRepositoryGCSServiceAccountKey"
	}

	// Default to no compression for artifacts
	artifact.Archive = &wfv1.ArchiveStrategy{
		None: &wfv1.NoneStrategy{},
	}
}

// injectHostPortAndResourcesToContainer adds a hostPort to the template container, if a nodeSelector is present.
// Kubernetes will ensure that multiple containers with the same hostPort do not share the same node.
func (c *Client) injectHostPortAndResourcesToContainer(template *wfv1.Template, opts *WorkflowExecutionOptions, config SystemConfig) error {
	if template.NodeSelector == nil {
		return nil
	}

	ports := []corev1.ContainerPort{
		{Name: "node-capturer", HostPort: 80, ContainerPort: 80},
	}

	// Add resource limits for GPUs
	nodePoolVal := ""
	for _, v := range template.NodeSelector {
		nodePoolVal = v
		break
	}
	if strings.Contains(nodePoolVal, "{{workflow.") {
		parts := strings.Split(strings.Replace(nodePoolVal, "}}", "", -1), ".")
		paramName := parts[len(parts)-1]
		for _, parameter := range opts.Parameters {
			if parameter.Name == paramName {
				nodePoolVal = *parameter.Value
			}
		}
	}
	n, err := config.NodePoolOptionByValue(nodePoolVal)
	if err != nil {
		return nil
	}
	if template.Container != nil {
		template.Container.Ports = ports
		if n != nil && n.Resources.Limits != nil {
			template.Container.Resources = n.Resources
		}
	}
	if template.Script != nil {
		template.Script.Container.Ports = ports
		if n != nil && n.Resources.Limits != nil {
			template.Script.Container.Resources = n.Resources
		}
	}
	return nil
}

func injectEnvironmentVariables(container *corev1.Container, systemConfig SystemConfig) {
	//Generate ENV vars from secret, if there is a container present in the workflow
	//Get template ENV vars, avoid over-writing them with secret values
	env.AddDefaultEnvVarsToContainer(container)
	env.PrependEnvVarToContainer(container, "ONEPANEL_API_URL", systemConfig["ONEPANEL_API_URL"])
	env.PrependEnvVarToContainer(container, "ONEPANEL_FQDN", systemConfig["ONEPANEL_FQDN"])
	env.PrependEnvVarToContainer(container, "ONEPANEL_DOMAIN", systemConfig["ONEPANEL_DOMAIN"])
	env.PrependEnvVarToContainer(container, "ONEPANEL_PROVIDER", systemConfig["ONEPANEL_PROVIDER"])
	env.PrependEnvVarToContainer(container, "ONEPANEL_RESOURCE_NAMESPACE", "{{workflow.namespace}}")
	env.PrependEnvVarToContainer(container, "ONEPANEL_RESOURCE_UID", "{{workflow.name}}")
}

func (c *Client) injectAutomatedFields(namespace string, wf *wfv1.Workflow, opts *WorkflowExecutionOptions) (err error) {
	if opts.PodGCStrategy == nil {
		if wf.Spec.PodGC == nil {
			podGCStrategy := env.Get("ARGO_POD_GC_STRATEGY", "OnPodCompletion")
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
	}, corev1.Volume{ // Artifacts out
		Name: "tmp",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{
				Medium: corev1.StorageMediumMemory,
			},
		},
	})

	// Create artifacts out volume
	wf.Spec.Volumes = append(wf.Spec.Volumes, corev1.Volume{
		Name: "out",
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
	for i := range wf.Spec.Templates {
		template := &wf.Spec.Templates[i]

		// Do not inject Istio sidecars in workflows
		if template.Metadata.Annotations == nil {
			template.Metadata.Annotations = make(map[string]string)
		}

		//For workflows with accessible sidecars, we need istio
		//Istio does not prevent the main container from stopping
		if envVarValueInSidecars(template.Sidecars, "ONEPANEL_INTERACTIVE_SIDECAR", "true") {
			template.Metadata.Annotations["sidecar.istio.io/inject"] = "true"
			template.Metadata.Annotations["traffic.sidecar.istio.io/includeOutboundIPRanges"] = ""
		} else {
			template.Metadata.Annotations["sidecar.istio.io/inject"] = "false"
		}

		if template.Container != nil {
			// Mount dev/shm
			template.Container.VolumeMounts = append(template.Container.VolumeMounts, corev1.VolumeMount{
				Name:      "sys-dshm",
				MountPath: "/dev/shm",
			})

			template.Container.VolumeMounts = append(template.Container.VolumeMounts, corev1.VolumeMount{
				Name:      "tmp",
				MountPath: "/mnt/tmp",
			})

			err = c.injectHostPortAndResourcesToContainer(template, opts, systemConfig)
			if err != nil {
				return err
			}
			injectEnvironmentVariables(template.Container, systemConfig)
		}

		if template.Script != nil {
			err = c.injectHostPortAndResourcesToContainer(template, opts, systemConfig)
			if err != nil {
				return err
			}
			injectEnvironmentVariables(&template.Script.Container, systemConfig)
		}

		if template.Container != nil || template.Script != nil {
			// Always add output artifacts for metrics but make them optional
			template.Outputs.Artifacts = append(template.Outputs.Artifacts, wfv1.Artifact{
				Name:     "sys-metrics",
				Path:     "/mnt/tmp/sys-metrics.json",
				Optional: true,
				Archive: &wfv1.ArchiveStrategy{
					None: &wfv1.NoneStrategy{},
				},
			})

			// Extend artifact credentials if only key is provided
			for j, artifact := range template.Outputs.Artifacts {
				injectArtifactRepositoryConfig(&artifact, namespaceConfig)
				template.Outputs.Artifacts[j] = artifact
			}

			for j, artifact := range template.Inputs.Artifacts {
				injectArtifactRepositoryConfig(&artifact, namespaceConfig)
				template.Inputs.Artifacts[j] = artifact
			}

			if template.Metadata.Labels == nil {
				template.Metadata.Labels = make(map[string]string)
			}
			template.Metadata.Labels["onepanel.io/entity-type"] = "Workflow"
			template.Metadata.Labels["onepanel.io/entity-uid"] = opts.WorkflowTemplateUID
		}
	}

	return
}

// ArchiveWorkflowExecution marks a WorkflowExecution as archived in database
// and deletes the argo workflow.
//
// If the database record does not exist, we still try to delete the argo workflow record.
// No errors are returned if the records do not exist.
func (c *Client) ArchiveWorkflowExecution(namespace, uid string) error {
	_, err := sb.Update("workflow_executions").
		Set("is_archived", true).
		Where(sq.Eq{
			"uid":       uid,
			"namespace": namespace,
		}).RunWith(c.DB).
		Exec()
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

// createWorkflow creates the workflow in the database and argo.
// Name is == to UID, no user friendly name.
// Workflow execution name == uid, example: name = my-friendly-wf-name-8skjz, uid = my-friendly-wf-name-8skjz
func (c *Client) createWorkflow(namespace string, workflowTemplateID uint64, workflowTemplateVersionID uint64, wf *wfv1.Workflow, opts *WorkflowExecutionOptions, labels types.JSONLabels) (createdWorkflow *WorkflowExecution, err error) {
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
				Value: wfv1.AnyStringPtr(*param.Value),
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

	newParameters := make([]wfv1.Parameter, 0)

	// Only used for workspaces
	workspaceUID := ""
	for i := range wf.Spec.Arguments.Parameters {
		param := wf.Spec.Arguments.Parameters[i]
		if param.Name == "sys-name" {
			uid, err := GenerateWorkspaceUID(param.Value.String())
			if err != nil {
				return nil, err
			}
			workspaceUID = uid
		}
	}

	for i := range wf.Spec.Arguments.Parameters {
		param := wf.Spec.Arguments.Parameters[i]
		if param.Value != nil {
			re, reErr := regexp.Compile(`{{\s*workflow.namespace\s*}}|{{\s*workspace.namespace\s*}}`)
			if reErr != nil {
				return nil, reErr
			}

			value := re.ReplaceAllString(param.Value.String(), namespace)

			if workspaceUID != "" {
				reWorkspaceUID, reErr := regexp.Compile(`{{\s*workspace.uid\s*}}`)
				if reErr != nil {
					return nil, reErr
				}
				value = reWorkspaceUID.ReplaceAllString(value, workspaceUID)
			}
			param.Value = wfv1.AnyStringPtr(value)
		}

		newParameters = append(newParameters, param)
	}
	wf.Spec.Arguments.Parameters = newParameters

	if err = injectFilesyncerSidecar(wf); err != nil {
		return nil, err
	}

	if err = injectWorkflowExecutionStatusCaller(wf, wfv1.NodeRunning); err != nil {
		return nil, err
	}

	if err = injectExitHandlerWorkflowExecutionStatistic(wf, &workflowTemplateID); err != nil {
		return nil, err
	}

	if err = c.injectAutomatedFields(namespace, wf, opts); err != nil {
		return nil, err
	}

	newTemplateOrder, err := c.injectAccessForSidecars(namespace, wf)
	if err != nil {
		return nil, err
	}
	wf.Spec.Templates = newTemplateOrder
	createdArgoWorkflow, err := c.ArgoprojV1alpha1().Workflows(namespace).Create(wf)
	if err != nil {
		return nil, err
	}

	createdWorkflow = &WorkflowExecution{
		Name:         createdArgoWorkflow.Name,
		CreatedAt:    createdArgoWorkflow.CreationTimestamp.UTC(),
		ArgoWorkflow: createdArgoWorkflow,
		WorkflowTemplate: &WorkflowTemplate{
			WorkflowTemplateVersionID: workflowTemplateVersionID,
		},
		Parameters: opts.Parameters,
		Labels:     labels,
	}

	if err = createdWorkflow.GenerateUID(createdArgoWorkflow.Name); err != nil {
		return nil, err
	}

	//Create an entry for workflow_executions statistic
	//CURL code will hit the API endpoint that will update the db row
	if err := c.createWorkflowExecutionDB(namespace, createdWorkflow); err != nil {
		return nil, err
	}

	return
}

func (c *Client) injectAccessForSidecars(namespace string, wf *wfv1.Workflow) ([]wfv1.Template, error) {
	var newTemplateOrder []wfv1.Template
	taskSysSendStatusName := "sys-send-status"
	taskSysSendExitStats := "sys-send-exit-stats"
	for tIdx, t := range wf.Spec.Templates {
		//Inject services, virtual routes
		for si, s := range t.Sidecars {
			//If ONEPANEL_INTERACTIVE_SIDECAR is true, sidecar needs to be accessible by HTTP
			//Otherwise, we skip the sidecar
			hasInjectIstio := hasEnvVarValue(s.Env, "ONEPANEL_INTERACTIVE_SIDECAR", "true")
			if !hasInjectIstio {
				continue
			}

			if len(s.Ports) == 0 {
				msg := fmt.Sprintf("sidecar %s must have at least one port.", s.Name)
				return nil, util.NewUserError(codes.InvalidArgument, msg)
			}

			t.Sidecars[si].MirrorVolumeMounts = ptr.Bool(true)
			serviceNameUID := "s" + uuid.New().String() + "--" + namespace
			serviceNameUIDDNSCompliant, err := uid2.GenerateUID(serviceNameUID, 63)
			if err != nil {
				return nil, util.NewUserError(codes.InvalidArgument, err.Error())
			}

			serviceName := serviceNameUIDDNSCompliant + "." + *c.systemConfig.Domain()

			serviceTemplateName := uuid.New().String()
			serviceTemplateNameAdd := "sys-k8s-service-template-add-" + serviceTemplateName
			serviceTemplateNameDelete := "sys-k8s-service-template-delete-" + serviceTemplateName
			serviceTaskName := "service-" + uuid.New().String()
			serviceAddTaskName := "sys-add-" + serviceTaskName
			serviceDeleteTaskName := "sys-delete-" + serviceTaskName
			virtualServiceTemplateName := uuid.New().String()
			virtualServiceTemplateNameAdd := "sys-k8s-virtual-service-template-add-" + virtualServiceTemplateName
			virtualServiceTemplateNameDelete := "sys-k8s-virtual-service-template-delete-" + virtualServiceTemplateName
			virtualServiceTaskName := "virtual-service-" + uuid.New().String()
			virtualServiceAddTaskName := "sys-add-" + virtualServiceTaskName
			virtualServiceDeleteTaskName := "sys-delete-" + virtualServiceTaskName
			var servicePorts []corev1.ServicePort
			var routes []*networking.HTTPRoute
			for _, port := range s.Ports {
				servicePort := corev1.ServicePort{
					Name:       port.Name,
					Protocol:   port.Protocol,
					Port:       port.ContainerPort,
					TargetPort: intstr.FromInt(int(port.ContainerPort)),
				}
				servicePorts = append(servicePorts, servicePort)
				route := networking.HTTPRoute{
					Match: []*networking.HTTPMatchRequest{
						{
							Uri: &networking.StringMatch{
								MatchType: &networking.StringMatch_Prefix{
									Prefix: "/"},
							},
						},
					},
					Route: []*networking.HTTPRouteDestination{
						{
							Destination: &networking.Destination{
								Host: serviceNameUIDDNSCompliant,
								Port: &networking.PortSelector{
									Number: uint32(port.ContainerPort),
								},
							},
						},
					},
				}
				routes = append(routes, &route)
			}
			service := corev1.Service{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Service",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: serviceNameUIDDNSCompliant,
				},
				Spec: corev1.ServiceSpec{
					Ports: servicePorts,
					Selector: map[string]string{
						serviceTaskName: serviceNameUIDDNSCompliant,
					},
				},
			}
			//Istio needs to know which pod to setup the route to
			if wf.Spec.Templates[tIdx].Metadata.Labels == nil {
				wf.Spec.Templates[tIdx].Metadata.Labels = make(map[string]string)
			}
			wf.Spec.Templates[tIdx].Metadata.Labels[serviceTaskName] = serviceNameUIDDNSCompliant
			serviceManifestBytes, err := yaml2.Marshal(service)
			if err != nil {
				return nil, err
			}
			serviceManifest := string(serviceManifestBytes)
			templateServiceResource := wfv1.Template{
				Name: serviceTemplateNameAdd,
				Metadata: wfv1.Metadata{
					Annotations: map[string]string{
						"sidecar.istio.io/inject": "false",
					},
				},
				Resource: &wfv1.ResourceTemplate{
					Action:   "create",
					Manifest: serviceManifest,
				},
			}
			newTemplateOrder = append(newTemplateOrder, templateServiceResource)
			//routes
			virtualServiceNameUUID := "vs-" + uuid.New().String()
			hosts := []string{serviceName}
			wf.Spec.Templates[tIdx].Outputs.Parameters = append(wf.Spec.Templates[tIdx].Outputs.Parameters,
				wfv1.Parameter{
					Name:  "sys-sidecar-url--" + s.Name,
					Value: wfv1.AnyStringPtr(serviceName),
				},
			)
			virtualService := map[string]interface{}{
				"apiVersion": "networking.istio.io/v1alpha3",
				"kind":       "VirtualService",
				"metadata": metav1.ObjectMeta{
					Name: virtualServiceNameUUID,
				},
				"spec": networking.VirtualService{
					Http:     routes,
					Gateways: []string{"istio-system/ingressgateway"},
					Hosts:    hosts,
				},
			}

			virtualServiceManifestBytes, err := yaml2.Marshal(virtualService)
			if err != nil {
				return nil, err
			}
			virtualServiceManifest := string(virtualServiceManifestBytes)

			templateRouteResource := wfv1.Template{
				Name: virtualServiceTemplateNameAdd,
				Metadata: wfv1.Metadata{
					Annotations: map[string]string{
						"sidecar.istio.io/inject": "false",
					},
				},
				Resource: &wfv1.ResourceTemplate{
					Action:   "create",
					Manifest: virtualServiceManifest,
				},
			}
			newTemplateOrder = append(newTemplateOrder, templateRouteResource)

			for i2, t2 := range wf.Spec.Templates {
				if t2.Name == wf.Spec.Entrypoint {
					if t2.DAG != nil {
						tasks := wf.Spec.Templates[i2].DAG.Tasks
						t := tasks[0]
						sysDepFound := false
						for _, d := range t.Dependencies {
							if d == taskSysSendStatusName {
								sysDepFound = true
								wf.Spec.Templates[i2].DAG.Tasks[0].Dependencies =
									[]string{virtualServiceAddTaskName}
							}
						}
						if sysDepFound == false {
							wf.Spec.Templates[i2].DAG.Tasks[0].Dependencies = append(wf.Spec.Templates[i2].DAG.Tasks[0].Dependencies, virtualServiceAddTaskName)
						}

						wf.Spec.Templates[i2].DAG.Tasks = append(tasks, []wfv1.DAGTask{
							{
								Name:         serviceAddTaskName,
								Template:     serviceTemplateNameAdd,
								Dependencies: []string{taskSysSendStatusName},
							},
							{
								Name:         virtualServiceAddTaskName,
								Template:     virtualServiceTemplateNameAdd,
								Dependencies: []string{serviceAddTaskName},
							},
						}...)
					}
				}
			}
			//Inject clean-up for service and virtualservice
			templateServiceDeleteResource := wfv1.Template{
				Name: serviceTemplateNameDelete,
				Metadata: wfv1.Metadata{
					Annotations: map[string]string{
						"sidecar.istio.io/inject": "false",
					},
				},
				Resource: &wfv1.ResourceTemplate{
					Action:   "delete",
					Manifest: serviceManifest,
				},
			}
			newTemplateOrder = append(newTemplateOrder, templateServiceDeleteResource)

			templateRouteDeleteResource := wfv1.Template{
				Name: virtualServiceTemplateNameDelete,
				Metadata: wfv1.Metadata{
					Annotations: map[string]string{
						"sidecar.istio.io/inject": "false",
					},
				},
				Resource: &wfv1.ResourceTemplate{
					Action:   "delete",
					Manifest: virtualServiceManifest,
				},
			}

			newTemplateOrder = append(newTemplateOrder, templateRouteDeleteResource)

			dagTasks := []wfv1.DAGTask{
				{
					Name:     serviceDeleteTaskName,
					Template: serviceTemplateNameDelete,
				},
				{
					Name:         virtualServiceDeleteTaskName,
					Template:     virtualServiceTemplateNameDelete,
					Dependencies: []string{serviceDeleteTaskName},
				},
			}
			if wf.Spec.OnExit != "" {
				for _, t := range wf.Spec.Templates {
					if t.Name == wf.Spec.OnExit {
						t.DAG.Tasks = append(t.DAG.Tasks, dagTasks...)
						sysExitDepFound := false
						for dti, dt := range t.DAG.Tasks {
							if dt.Name == taskSysSendExitStats {
								sysExitDepFound = true
								t.DAG.Tasks[dti].Dependencies = append(t.DAG.Tasks[dti].Dependencies, virtualServiceDeleteTaskName)
							}
						}
						if sysExitDepFound == false {
							t.DAG.Tasks[0].Dependencies = append(t.DAG.Tasks[0].Dependencies, virtualServiceDeleteTaskName)
						}
						break
					}
				}
			} else {
				exitHandlerDAG := wfv1.Template{
					Name: "exit-handler",
					DAG: &wfv1.DAGTemplate{
						Tasks: dagTasks,
					},
				}
				wf.Spec.OnExit = "exit-handler"
				wf.Spec.Templates = append(wf.Spec.Templates, exitHandlerDAG)
			}
		}
		newTemplateOrder = append(newTemplateOrder, wf.Spec.Templates[tIdx])

	}
	return newTemplateOrder, nil
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
	clusterWftmplGetter := templateresolution.WrapClusterWorkflowTemplateInterface(c.ArgoprojV1alpha1().ClusterWorkflowTemplates())
	for _, wf := range workflows {
		if err = c.injectAutomatedFields(namespace, &wf, &WorkflowExecutionOptions{}); err != nil {
			return err
		}
		_, err = validate.ValidateWorkflow(wftmplGetter, clusterWftmplGetter, &wf, validate.ValidateOpts{})
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
// If workflow.Name is set, it is used instead of a generated name.
// If there is a parameter named "workflow-execution-name" in workflow.Parameters, it is set as the name.
func (c *Client) CreateWorkflowExecution(namespace string, workflow *WorkflowExecution, workflowTemplate *WorkflowTemplate) (*WorkflowExecution, error) {
	opts := &WorkflowExecutionOptions{
		Labels:     make(map[string]string),
		Parameters: workflow.Parameters,
	}

	if workflow.Name != "" {
		opts.Name = workflow.Name
	}

	if workflowExecutionName := workflow.GetParameterValue("workflow-execution-name"); workflowExecutionName != nil {
		opts.Name = *workflowExecutionName
	}

	nameUID, err := uid2.GenerateUID(workflowTemplate.Name, 63)
	if err != nil {
		return nil, err
	}
	opts.GenerateName = nameUID + "-"
	opts.WorkflowTemplateUID = workflowTemplate.UID

	opts.Labels[workflowTemplateUIDLabelKey] = workflowTemplate.UID
	opts.Labels[workflowTemplateVersionLabelKey] = fmt.Sprint(workflowTemplate.Version)
	label.MergeLabelsPrefix(opts.Labels, workflow.Labels, label.TagPrefix)

	workflows, err := getWorkflowsFromWorkflowTemplate(workflowTemplate)
	if err != nil {
		return nil, err
	}

	if len(workflows) != 1 {
		return nil, fmt.Errorf("workflow Template contained more than 1 workflow execution")
	}

	wf := &workflows[0]
	if wf.Spec.VolumeClaimGC == nil {
		wf.Spec.VolumeClaimGC = &wfv1.VolumeClaimGC{
			Strategy: wfv1.VolumeClaimGCOnCompletion,
		}
	}

	createdWorkflow, err := c.createWorkflow(namespace, workflowTemplate.ID, workflowTemplate.WorkflowTemplateVersionID, wf, opts, workflow.Labels)
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace": namespace,
			"Workflow":  workflow,
			"Error":     err.Error(),
		}).Error("Error parsing workflow.")
		return nil, err
	}

	workflow.ID = createdWorkflow.ID
	workflow.Name = createdWorkflow.Name
	workflow.CreatedAt = createdWorkflow.CreatedAt.UTC()
	workflow.UID = createdWorkflow.UID
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

	// We remove the name because CreateWorkflowExecution will otherwise use it to try and create an execution with that name
	workflowExecution.Name = ""
	return c.CreateWorkflowExecution(namespace, workflowExecution, workflowTemplate)
}

// createWorkflowExecutionDB inserts a workflow execution into the database.
// Required fields
// * name
// * createdAt // we sync the argo created at with the db
// * parameters, if any
// * WorkflowTemplate.WorkflowTemplateVersionID
//
// After success, the passed in WorkflowExecution will have it's ID set to the new db record.
func (c *Client) createWorkflowExecutionDB(namespace string, workflowExecution *WorkflowExecution) (err error) {
	parametersJSON, err := json.Marshal(workflowExecution.Parameters)
	if err != nil {
		return err
	}

	if err := workflowExecution.GenerateUID(workflowExecution.Name); err != nil {
		return err
	}

	err = sb.Insert("workflow_executions").
		SetMap(sq.Eq{
			"UID":                          workflowExecution.UID,
			"workflow_template_version_id": workflowExecution.WorkflowTemplate.WorkflowTemplateVersionID,
			"name":                         workflowExecution.Name,
			"namespace":                    namespace,
			"created_at":                   workflowExecution.CreatedAt.UTC(),
			"phase":                        wfv1.NodePending,
			"parameters":                   string(parametersJSON),
			"is_archived":                  false,
			"labels":                       workflowExecution.Labels,
			"metrics":                      workflowExecution.Metrics,
		}).
		Suffix("RETURNING id").
		RunWith(c.DB).
		QueryRow().
		Scan(&workflowExecution.ID)

	return
}

func (c *Client) FinishWorkflowExecutionStatisticViaExitHandler(namespace, name string, phase wfv1.NodePhase, startedAt time.Time) (err error) {
	_, err = sb.Update("workflow_executions").
		SetMap(sq.Eq{
			"started_at":  startedAt.UTC(),
			"name":        name,
			"namespace":   namespace,
			"finished_at": time.Now().UTC(),
			"phase":       phase,
		}).
		Where(sq.And{
			sq.Eq{"name": name},
			sq.NotEq{"phase": "Terminated"},
		}).
		RunWith(c.DB).
		Exec()

	return err
}

func (c *Client) CronStartWorkflowExecutionStatisticInsert(namespace, uid string, workflowTemplateID int64) (err error) {
	queryWt := c.workflowTemplatesSelectBuilder(namespace).
		Where(sq.Eq{
			"wt.id": workflowTemplateID,
		})

	workflowTemplate := &WorkflowTemplate{}
	if err := c.DB.Getx(workflowTemplate, queryWt); err != nil {
		return err
	}

	queryCw := c.cronWorkflowSelectBuilder(namespace, workflowTemplate.UID)

	cronWorkflow := &CronWorkflow{}
	if err := c.DB.Getx(cronWorkflow, queryCw); err != nil {
		return err
	}

	parametersJSON, err := cronWorkflow.GetParametersFromWorkflowSpecJSON()
	if err != nil {
		return err
	}

	workflowExecutionID := uint64(0)
	err = sb.Insert("workflow_executions").
		SetMap(sq.Eq{
			"uid":                          uid,
			"workflow_template_version_id": cronWorkflow.WorkflowTemplateVersionID,
			"name":                         uid,
			"namespace":                    namespace,
			"phase":                        wfv1.NodeRunning,
			"started_at":                   time.Now().UTC(),
			"cron_workflow_id":             cronWorkflow.ID,
			"parameters":                   string(parametersJSON),
			"labels":                       cronWorkflow.Labels,
			"metrics":                      Metrics{},
		}).
		Suffix("RETURNING id").
		RunWith(c.DB).
		QueryRow().
		Scan(&workflowExecutionID)
	if err != nil {
		return err
	}

	return err
}

func (c *Client) GetWorkflowExecution(namespace, uid string) (workflow *WorkflowExecution, err error) {
	workflow = &WorkflowExecution{}
	query := sb.Select(getWorkflowExecutionColumns("we")...).
		Columns(getWorkflowTemplateColumns("wt", "workflow_template")...).
		Columns(`wtv.manifest "workflow_template.manifest"`).
		From("workflow_executions we").
		Join("workflow_template_versions wtv ON wtv.id = we.workflow_template_version_id").
		Join("workflow_templates wt ON wt.id = wtv.workflow_template_id").
		Where(sq.Eq{
			"wt.namespace":   namespace,
			"we.name":        uid,
			"we.is_archived": false,
		})

	if err := c.DB.Getx(workflow, query); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

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
		64,
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

// ListWorkflowExecutions gets a list of WorkflowExecutions ordered by most recently created first.
func (c *Client) ListWorkflowExecutions(namespace, workflowTemplateUID, workflowTemplateVersion string, includeSystem bool, request *request.Request) (workflows []*WorkflowExecution, err error) {
	sb := workflowExecutionsSelectBuilder(namespace, workflowTemplateUID, workflowTemplateVersion, includeSystem)

	if request.HasSorting() {
		properties := getWorkflowExecutionColumnsMap(true)
		for _, order := range request.Sort.Properties {
			if columnName, ok := properties[order.Property]; ok {
				nullSort := "NULLS FIRST"
				if order.Direction == "desc" {
					nullSort = "NULLS LAST" // default in postgres, but let's be explicit
				}
				sb = sb.OrderBy(fmt.Sprintf("we.%v %v %v", columnName, order.Direction, nullSort))
			}
		}
	} else {
		sb = sb.OrderBy("we.created_at DESC")
	}

	sb, err = applyWorkflowExecutionFilter(sb, request)
	if err != nil {
		return nil, err
	}

	sb = *request.ApplyPaginationToSelect(&sb)
	if err := c.DB.Selectx(&workflows, sb); err != nil {
		return nil, err
	}

	return
}

// CountWorkflowExecutions returns the number of workflow executions
func (c *Client) CountWorkflowExecutions(namespace, workflowTemplateUID, workflowTemplateVersion string, includeSystem bool, request *request.Request) (count int, err error) {
	sb := workflowExecutionsSelectBuilderNoColumns(namespace, workflowTemplateUID, workflowTemplateVersion, includeSystem).
		Columns("COUNT(*)")

	sb, err = applyWorkflowExecutionFilter(sb, request)
	if err != nil {
		return
	}

	err = sb.RunWith(c.DB).
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

		timeouts := 0

		for !done {
			for next = range watcher.ResultChan() {
				watchEvent, ok := next.Object.(*metav1.Status)
				if ok {
					// If a timeout occurred, retry.
					if strings.Contains(watchEvent.Message, "Client.Timeout or context cancellation") {
						if timeouts > 5 {
							done = true
							break
						}

						timeouts++
						continue
					}

					done = true
					break
				}

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
					UID:        workflow.Name,
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

func (c *Client) GetWorkflowExecutionLogs(namespace, uid, podName, containerName string) (<-chan []*LogEntry, error) {
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
		gcsClient *gcs.Client
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
				stream, err = gcsClient.GetObject(config.ArtifactRepository.GCS.Bucket, key)
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

	logWatcher := make(chan []*LogEntry)
	go func() {
		buffer := make([]byte, 4096)
		reader := bufio.NewReader(stream)

		lastChunkSent := -1
		lastLine := ""
		for {
			bytesRead, err := reader.Read(buffer)
			if err != nil && err.Error() != "EOF" {
				break
			}
			content := lastLine + string(buffer[:bytesRead])
			lastLine = ""

			chunk := make([]*LogEntry, 0)
			lines := strings.Split(content, "\n")
			for lineIndex, line := range lines {
				if lineIndex == len(lines)-1 {
					lastLine = line
					continue
				}

				newLogEntry := LogEntryFromLine(&line)
				if newLogEntry == nil {
					continue
				}

				chunk = append(chunk, newLogEntry)
			}

			if lastChunkSent == 0 && lastLine != "" {
				newLogEntry := LogEntryFromLine(&lastLine)
				if newLogEntry != nil {
					chunk = append(chunk, newLogEntry)
					lastLine = ""
				}
			}

			if len(chunk) > 0 {
				logWatcher <- chunk
			}
			lastChunkSent = len(chunk)

			if err != nil && err.Error() == "EOF" {
				break
			}
		}

		newLogEntry := LogEntryFromLine(&lastLine)
		if newLogEntry != nil {
			logWatcher <- []*LogEntry{newLogEntry}
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
		gcsClient *gcs.Client
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
			stream, err = gcsClient.GetObject(config.ArtifactRepository.GCS.Bucket, key)
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

	hy := hydrator.New(sqldb.ExplosiveOffloadNodeStatusRepo)

	wf, err = argoutil.RetryWorkflow(c, hy, c.ArgoprojV1alpha1().Workflows(namespace), uid, true, "")

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

	wf, err = argoutil.SubmitWorkflow(c.ArgoprojV1alpha1().Workflows(namespace), c, namespace, wf, &wfv1.SubmitOpts{})
	if err != nil {
		return
	}

	workflow = typeWorkflow(wf)

	return
}

func (c *Client) ResumeWorkflowExecution(namespace, uid string) (workflow *WorkflowExecution, err error) {
	hy := hydrator.New(sqldb.ExplosiveOffloadNodeStatusRepo)
	err = argoutil.ResumeWorkflow(c.ArgoprojV1alpha1().Workflows(namespace), hy, uid, "")
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

// TerminateWorkflowExecution marks a workflows execution as terminated in DB and terminates the argo resource.
func (c *Client) TerminateWorkflowExecution(namespace, uid string) (err error) {
	_, err = sb.Update("workflow_executions").
		Set("phase", "Terminated").
		Set("started_at", time.Time.UTC(time.Now())).
		Set("finished_at", time.Time.UTC(time.Now())).
		Where(sq.Eq{
			"uid":       uid,
			"namespace": namespace,
		}).
		RunWith(c.DB).
		Exec()
	if err != nil {
		return err
	}

	hy := hydrator.New(sqldb.ExplosiveOffloadNodeStatusRepo)
	err = argoutil.StopWorkflow(c.ArgoprojV1alpha1().Workflows(namespace), hy, uid, "", "")

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

	specMap, ok := spec.(map[interface{}]interface{})
	if !ok {
		return manifest, nil
	}

	arguments, ok := specMap["arguments"]
	if !ok {
		return manifest, nil
	}

	argumentsMap, ok := arguments.(map[interface{}]interface{})
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
		paramMap, ok := parameter.(map[interface{}]interface{})
		if !ok {
			continue
		}

		// If the parameter does not have a value, skip it so argo doesn't try to process it and fail.
		if _, hasValue := paramMap["value"]; !hasValue {
			paramMap["value"] = "<value>"
		}

		parametersToKeep = append(parametersToKeep, parameter)

		keysToDelete := make([]interface{}, 0)
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

// GetWorkflowExecutionStatisticsForNamespace loads statistics on workflow executions for the provided namespace
func (c *Client) GetWorkflowExecutionStatisticsForNamespace(namespace string) (report *WorkflowExecutionStatisticReport, err error) {
	statsSelect := `
		MAX(we.created_at) last_executed,
		COUNT(*) FILTER (WHERE finished_at IS NULL AND (phase = 'Running' OR phase = 'Pending')) running,
		COUNT(*) FILTER (WHERE finished_at IS NOT NULL AND phase = 'Succeeded') completed,
		COUNT(*) FILTER (WHERE finished_at IS NOT NULL AND (phase = 'Failed' OR phase = 'Error')) failed,
		COUNT(*) FILTER (WHERE phase = 'Terminated') terminated,
		COUNT(*) total`

	query := sb.Select(statsSelect).
		From("workflow_executions we").
		LeftJoin("workflow_template_versions wtv ON we.workflow_template_version_id = wtv.id").
		LeftJoin("workflow_templates wt ON wtv.workflow_template_id = wt.id").
		Where(sq.Eq{
			"we.namespace": namespace,
			"wt.is_system": false,
		})

	report = &WorkflowExecutionStatisticReport{}
	err = c.DB.Getx(report, query)

	return
}

// GetWorkflowExecutionStatisticsForTemplates loads statistics on workflow executions for the provided
// workflowTemplates and sets it as the WorkflowExecutionStatisticReport property
func (c *Client) GetWorkflowExecutionStatisticsForTemplates(workflowTemplates ...*WorkflowTemplate) (err error) {
	if len(workflowTemplates) == 0 {
		return nil
	}

	tx, err := c.DB.Begin()
	if err != nil {
		return err
	}

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
		Where(sq.Eq{
			"wtv.workflow_template_id": ids,
		}).
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
	host := "onepanel-core.onepanel.svc.cluster.local"
	endpoint := fmt.Sprintf("http://%s%s", host, curlPath)

	template = &wfv1.Template{
		Name:   name,
		Inputs: inputs,
		Container: &corev1.Container{
			Name:    "curl",
			Image:   "curlimages/curl:7.73.0",
			Command: []string{"sh", "-c"},
			Args: []string{
				"SERVICE_ACCOUNT_TOKEN=$(cat /var/run/secrets/kubernetes.io/serviceaccount/token) " +
					"&& curl -X " + curlMethod + " -s -o /dev/null -w '%{http_code}' " +
					"--connect-timeout 10 --retry 10 --retry-delay 5 --retry-all-errors --fail " +
					"'" + endpoint + "' -H \"Content-Type: application/json\" -H 'Connection: keep-alive' -H 'Accept: application/json' " +
					"-H 'Authorization: Bearer '\"$SERVICE_ACCOUNT_TOKEN\"'' " +
					"--data '" + curlBody + "' --compressed",
			},
		},
	}
	return
}

func injectFilesyncerSidecar(wf *wfv1.Workflow) error {
	filesyncer := wfv1.UserContainer{
		Container: corev1.Container{
			Name:  "sys-filesyncer",
			Image: "onepanel/filesyncer:v0.19.0",
			Args:  []string{"server", "-server-prefix=/sys/filesyncer", "-backend=local-storage"},
			Env: []corev1.EnvVar{
				{
					Name:  "ONEPANEL_INTERACTIVE_SIDECAR",
					Value: "true",
				},
			},
			Ports: []corev1.ContainerPort{
				{
					ContainerPort: 8888,
				},
			},
		},
	}

	for i := range wf.Spec.Templates {
		template := &wf.Spec.Templates[i]

		if (template.Container != nil && len(template.Container.VolumeMounts) != 0) ||
			(template.Script != nil && len(template.Script.VolumeMounts) != 0) {
			template.Sidecars = append(template.Sidecars, filesyncer)
		}
	}

	return nil
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
					}
				}
			}
			wf.Spec.Templates[i].DAG.Tasks = append(t.DAG.Tasks, wfv1.DAGTask{
				Name:     containerTemplate.Name,
				Template: containerTemplate.Name,
			})
			break
		}
	}

	return nil
}

func workflowExecutionsSelectBuilderNoColumns(namespace, workflowTemplateUID, workflowTemplateVersion string, includeSystem bool) sq.SelectBuilder {
	whereMap := sq.Eq{
		"wt.namespace":   namespace,
		"we.is_archived": false,
	}

	if !includeSystem {
		whereMap["wt.is_system"] = false
	}

	if workflowTemplateUID != "" {
		whereMap["wt.uid"] = workflowTemplateUID

		if workflowTemplateVersion != "" {
			whereMap["wtv.version"] = workflowTemplateVersion
		}
	}

	sb := sb.Select().
		From("workflow_executions we").
		LeftJoin("workflow_template_versions wtv ON wtv.id = we.workflow_template_version_id").
		LeftJoin("workflow_templates wt ON wt.id = wtv.workflow_template_id").
		Where(whereMap)

	return sb
}

func workflowExecutionsSelectBuilder(namespace, workflowTemplateUID, workflowTemplateVersion string, includeSystem bool) sq.SelectBuilder {
	sb := workflowExecutionsSelectBuilderNoColumns(namespace, workflowTemplateUID, workflowTemplateVersion, includeSystem)
	sb = sb.Columns(getWorkflowExecutionColumns("we")...).
		Columns(`wtv.version "workflow_template.version"`, `wtv.created_at "workflow_template.created_at"`, `wt.name "workflow_template.name"`, `wt.uid "workflow_template.uid"`)

	return sb
}

func (c *Client) getWorkflowExecutionAndTemplate(namespace string, uid string) (workflow *WorkflowExecution, err error) {
	sb := sb.Select(getWorkflowExecutionColumns("we")...).
		Columns(getWorkflowTemplateColumns("wt", "workflow_template")...).
		Columns(`wtv.manifest "workflow_template.manifest"`, `wtv.version "workflow_template.version"`).
		From("workflow_executions we").
		Join("workflow_template_versions wtv ON we.workflow_template_version_id = wtv.id").
		Join("workflow_templates wt ON wtv.workflow_template_id = wt.id").
		Where(sq.Eq{
			"wt.namespace":   namespace,
			"we.name":        uid,
			"we.is_archived": false,
		})

	workflow = &WorkflowExecution{}
	if err = c.DB.Getx(workflow, sb); err != nil {
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

// AddWorkflowExecutionMetrics merges the metrics provided with the ones present in the workflow execution identified by (namespace, uid)
func (c *Client) AddWorkflowExecutionMetrics(namespace, uid string, metrics Metrics, override bool) (workflowExecution *WorkflowExecution, err error) {
	workflowExecution, err = c.GetWorkflowExecution(namespace, uid)
	if err != nil {
		return nil, err
	}
	if workflowExecution == nil {
		return nil, util.NewUserError(codes.NotFound, "Workflow execution not found")
	}

	workflowExecution.Metrics.Merge(metrics, override)

	_, err = sb.Update("workflow_executions").
		Set("metrics", workflowExecution.Metrics).
		Where(sq.Eq{
			"namespace": namespace,
			"uid":       uid,
		}).
		RunWith(c.DB).
		Exec()
	if err != nil {
		return nil, util.NewUserError(codes.Internal, "Error updating metrics.")
	}

	return
}

// UpdateWorkflowExecutionMetrics replaces the metrics of a workflow execution identified by (namespace, uid) with the input metrics.
func (c *Client) UpdateWorkflowExecutionMetrics(namespace, uid string, metrics Metrics) (workflowExecution *WorkflowExecution, err error) {
	workflowExecution, err = c.GetWorkflowExecution(namespace, uid)
	if err != nil {
		return nil, err
	}
	if workflowExecution == nil {
		return nil, util.NewUserError(codes.NotFound, "Workflow execution not found")
	}

	workflowExecution.Metrics = metrics

	_, err = sb.Update("workflow_executions").
		Set("metrics", workflowExecution.Metrics).
		Where(sq.Eq{
			"namespace": namespace,
			"uid":       uid,
		}).
		RunWith(c.DB).
		Exec()
	if err != nil {
		return nil, util.NewUserError(codes.Internal, "Error updating metrics.")
	}

	return
}

// ListWorkflowExecutionsField loads all of the distinct field values for workflow executions
func (c *Client) ListWorkflowExecutionsField(namespace, field string) (value []string, err error) {
	columnName := ""

	switch field {
	case "name":
		columnName = "we.name"
		break
	case "templateName":
		columnName = "wt.name"
		break
	default:
		return nil, fmt.Errorf("unsupported field '%v'", field)
	}

	sb := sb.Select(columnName).
		Distinct().
		From("workflow_executions we").
		Join("workflow_template_versions wtv ON we.workflow_template_version_id = wtv.id").
		Join("workflow_templates wt ON wtv.workflow_template_id = wt.id").
		Where(sq.And{sq.Eq{
			"we.namespace": namespace,
			"wt.is_system": false,
		}}).OrderBy(columnName)

	err = c.DB.Selectx(&value, sb)

	return
}

// CountWorkflowExecutionsForWorkflowTemplate returns the number of workflow executions associated with the workflow template identified by it's id.
func (c *Client) CountWorkflowExecutionsForWorkflowTemplate(workflowTemplateID uint64) (count int, err error) {
	err = sb.Select("COUNT(*)").
		From("workflow_executions we").
		Join("workflow_template_versions wtv ON we.workflow_template_version_id = wtv.id").
		Join("workflow_templates wt ON wtv.workflow_template_id = wt.id").
		Where(sq.Eq{"wt.id": workflowTemplateID}).
		RunWith(c.DB).
		QueryRow().
		Scan(&count)

	return
}
