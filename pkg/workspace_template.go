package v1

import (
	"database/sql"
	"encoding/json"
	"fmt"
	sq "github.com/Masterminds/squirrel"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/asaskevich/govalidator"
	"github.com/onepanelio/core/pkg/util"
	"github.com/onepanelio/core/pkg/util/pagination"
	"github.com/onepanelio/core/pkg/util/ptr"
	"google.golang.org/grpc/codes"
	networking "istio.io/api/networking/v1alpha3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
	"sigs.k8s.io/yaml"
)

func parseWorkspaceSpec(template string) (spec *WorkspaceSpec, err error) {
	err = yaml.UnmarshalStrict([]byte(template), &spec)

	return
}

func generateArguments(spec *WorkspaceSpec, config map[string]string) (err error) {
	if spec.Arguments == nil {
		spec.Arguments = &Arguments{
			Parameters: []Parameter{},
		}
	}

	// Resource action parameter
	spec.Arguments.Parameters = append(spec.Arguments.Parameters, Parameter{
		Name:        "sys-name",
		Type:        "input.text",
		Value:       ptr.String("name"),
		DisplayName: ptr.String("Workspace name"),
		Hint:        ptr.String("Must be between 3-30 characters, contain only alphanumeric or `-` characters"),
		Required:    true,
	})

	// TODO: These can be removed when lint validation of workflows work
	// Resource action parameter
	spec.Arguments.Parameters = append(spec.Arguments.Parameters, Parameter{
		Name:  "sys-resource-action",
		Value: ptr.String("apply"),
		Type:  "input.hidden",
	})
	// Workspace action
	spec.Arguments.Parameters = append(spec.Arguments.Parameters, Parameter{
		Name:  "sys-workspace-action",
		Value: ptr.String("create"),
		Type:  "input.hidden",
	})
	// Host
	spec.Arguments.Parameters = append(spec.Arguments.Parameters, Parameter{
		Name:  "sys-host",
		Value: ptr.String(config["ONEPANEL_DOMAIN"]),
		Type:  "input.hidden",
	})

	// Node pool parameter and options
	var options []*ParameterOption
	if err = yaml.Unmarshal([]byte(config["applicationNodePoolOptions"]), &options); err != nil {
		return
	}
	spec.Arguments.Parameters = append(spec.Arguments.Parameters, Parameter{
		Name:        "sys-node-pool",
		Value:       ptr.String(options[0].Value),
		Type:        "select.select",
		Options:     options,
		DisplayName: ptr.String("Node pool"),
		Hint:        ptr.String("Name of node pool or group"),
		Required:    true,
	})

	// Volume size parameters
	volumeClaimsMapped := make(map[string]bool)
	for _, c := range spec.Containers {
		for _, v := range c.VolumeMounts {
			if volumeClaimsMapped[v.Name] {
				continue
			}

			spec.Arguments.Parameters = append(spec.Arguments.Parameters, Parameter{
				Name:        fmt.Sprintf("sys-%v-volume-size", v.Name),
				Type:        "input.number",
				Value:       ptr.String("20480"),
				DisplayName: ptr.String(fmt.Sprintf("Disk size for \"%v\"", v.Name)),
				Hint:        ptr.String(fmt.Sprintf("Disk size in MB for volume mounted at `%v`", v.MountPath)),
				Required:    true,
			})

			volumeClaimsMapped[v.Name] = true
		}
	}

	return
}

func createServiceManifest(spec *WorkspaceSpec) (serviceManifest string, err error) {
	service := corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "{{workflow.parameters.sys-uid}}",
		},
		Spec: corev1.ServiceSpec{
			Ports: spec.Ports,
			Selector: map[string]string{
				"app": "{{workflow.parameters.sys-uid}}",
			},
		},
	}
	serviceManifestBytes, err := yaml.Marshal(service)
	if err != nil {
		return
	}
	serviceManifest = string(serviceManifestBytes)

	return
}

func createVirtualServiceManifest(spec *WorkspaceSpec) (virtualServiceManifest string, err error) {
	for _, h := range spec.Routes {
		for _, r := range h.Route {
			r.Destination.Host = "{{workflow.parameters.sys-uid}}"
		}
	}
	virtualService := map[string]interface{}{
		"apiVersion": "networking.istio.io/v1alpha3",
		"kind":       "VirtualService",
		"metadata": metav1.ObjectMeta{
			Name: "{{workflow.parameters.sys-uid}}",
		},
		"spec": networking.VirtualService{
			Http:     spec.Routes,
			Gateways: []string{"istio-system/ingressgateway"},
			Hosts:    []string{"{{workflow.parameters.sys-host}}"},
		},
	}
	virtualServiceManifestBytes, err := yaml.Marshal(virtualService)
	if err != nil {
		return
	}
	virtualServiceManifest = string(virtualServiceManifestBytes)

	return
}

func createStatefulSetManifest(workspaceSpec *WorkspaceSpec, config map[string]string) (statefulSetManifest string, err error) {
	var volumeClaims []map[string]interface{}
	volumeClaimsMapped := make(map[string]bool)
	for i, c := range workspaceSpec.Containers {
		workspaceSpec.Containers[i].EnvFrom = append(workspaceSpec.Containers[i].EnvFrom, corev1.EnvFromSource{
			SecretRef: &corev1.SecretEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: DefaultEnvironmentVariableSecret,
				},
				Optional: ptr.Bool(true),
			},
		})

		for _, v := range c.VolumeMounts {
			if volumeClaimsMapped[v.Name] {
				continue
			}

			volumeClaims = append(volumeClaims, map[string]interface{}{
				"metadata": metav1.ObjectMeta{
					Name: v.Name,
				},
				"spec": map[string]interface{}{
					"accessModes": []corev1.PersistentVolumeAccessMode{
						"ReadWriteOnce",
					},
					"storageClassName": ptr.String("onepanel"),
					"resources": map[string]interface{}{
						"requests": map[string]string{
							"storage": fmt.Sprintf("{{workflow.parameters.sys-%v-volume-size}}Mi", v.Name),
						},
					},
				},
			})

			volumeClaimsMapped[v.Name] = true
		}
	}

	statefulSet := map[string]interface{}{
		"apiVersion": "apps/v1",
		"kind":       "StatefulSet",
		"metadata": metav1.ObjectMeta{
			Name: "{{workflow.parameters.sys-uid}}",
		},
		"spec": map[string]interface{}{
			"replicas":    1,
			"serviceName": "{{workflow.parameters.sys-uid}}",
			"selector": &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "{{workflow.parameters.sys-uid}}",
				},
			},
			"template": corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "{{workflow.parameters.sys-uid}}",
					},
				},
				Spec: corev1.PodSpec{
					NodeSelector: map[string]string{
						config["applicationNodePoolLabel"]: "{{workflow.parameters.sys-node-pool}}",
					},
					Containers: workspaceSpec.Containers,
				},
			},
			"volumeClaimTemplates": volumeClaims,
		},
	}
	statefulSetManifestBytes, err := yaml.Marshal(statefulSet)
	if err != nil {
		return
	}
	statefulSetManifest = string(statefulSetManifestBytes)

	return
}

func unmarshalWorkflowTemplate(spec *WorkspaceSpec, serviceManifest, virtualServiceManifest, containersManifest string) (workflowTemplateSpecManifest string, err error) {
	var volumeClaimItems []wfv1.Item
	volumeClaimsMapped := make(map[string]bool)
	for _, c := range spec.Containers {
		for _, v := range c.VolumeMounts {
			if volumeClaimsMapped[v.Name] {
				continue
			}

			volumeClaimItems = append(volumeClaimItems, wfv1.Item{Type: wfv1.String, StrVal: v.Name})

			volumeClaimsMapped[v.Name] = true
		}
	}

	deletePVCManifest := `apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: {{inputs.parameters.sys-pvc-name}}-{{workflow.parameters.sys-uid}}-0
`
	templates := []wfv1.Template{
		{
			Name: "workspace",
			DAG: &wfv1.DAGTemplate{
				Tasks: []wfv1.DAGTask{
					{
						Name:     "service",
						Template: "service-resource",
					},
					{
						Name:         "virtual-service",
						Template:     "virtual-service-resource",
						Dependencies: []string{"service"},
					},
					{
						Name:         "stateful-set",
						Template:     "stateful-set-resource",
						Dependencies: []string{"virtual-service"},
						When:         "{{workflow.parameters.sys-workspace-action}} == create || {{workflow.parameters.sys-workspace-action}} == update",
					},
					{
						Name:         "delete-stateful-set",
						Template:     "delete-stateful-set-resource",
						Dependencies: []string{"virtual-service"},
						When:         "{{workflow.parameters.sys-workspace-action}} == pause || {{workflow.parameters.sys-workspace-action}} == delete",
					},
					{
						Name:         "delete-pvc",
						Template:     "delete-pvc-resource",
						Dependencies: []string{"delete-stateful-set"},
						Arguments: wfv1.Arguments{
							Parameters: []wfv1.Parameter{
								{
									Name:  "sys-pvc-name",
									Value: ptr.String("{{item}}"),
								},
							},
						},
						When:      "{{workflow.parameters.sys-workspace-action}} == delete",
						WithItems: volumeClaimItems,
					},
					{
						Name:         "sys-set-phase-running",
						Template:     "sys-update-status",
						Dependencies: []string{"stateful-set"},
						Arguments: wfv1.Arguments{
							Parameters: []wfv1.Parameter{
								{
									Name:  "sys-workspace-phase",
									Value: ptr.String(string(WorkspaceRunning)),
								},
							},
						},
						When: "{{workflow.parameters.sys-workspace-action}} == create || {{workflow.parameters.sys-workspace-action}} == update",
					},
					{
						Name:         "sys-set-phase-paused",
						Template:     "sys-update-status",
						Dependencies: []string{"delete-stateful-set"},
						Arguments: wfv1.Arguments{
							Parameters: []wfv1.Parameter{
								{
									Name:  "sys-workspace-phase",
									Value: ptr.String(string(WorkspacePaused)),
								},
							},
						},
						When: "{{workflow.parameters.sys-workspace-action}} == pause",
					},
					{
						Name:         "sys-set-phase-terminated",
						Template:     "sys-update-status",
						Dependencies: []string{"delete-pvc"},
						Arguments: wfv1.Arguments{
							Parameters: []wfv1.Parameter{
								{
									Name:  "sys-workspace-phase",
									Value: ptr.String(string(WorkspaceTerminated)),
								},
							},
						},
						When: "{{workflow.parameters.sys-workspace-action}} == delete",
					},
					{
						Name:         spec.PostExecutionWorkflow.Entrypoint,
						Template:     spec.PostExecutionWorkflow.Entrypoint,
						Dependencies: []string{"stateful-set", "delete-stateful-set"},
					},
				},
			},
		},
		{
			Name: "service-resource",
			Resource: &wfv1.ResourceTemplate{
				Action:   "{{workflow.parameters.sys-resource-action}}",
				Manifest: serviceManifest,
			},
		},
		{
			Name: "virtual-service-resource",
			Resource: &wfv1.ResourceTemplate{
				Action:   "{{workflow.parameters.sys-resource-action}}",
				Manifest: virtualServiceManifest,
			},
		},
		{
			Name: "stateful-set-resource",
			Resource: &wfv1.ResourceTemplate{
				Action:           "{{workflow.parameters.sys-resource-action}}",
				Manifest:         containersManifest,
				SuccessCondition: "status.readyReplicas > 0",
			},
		},
		{
			Name: "delete-stateful-set-resource",
			Resource: &wfv1.ResourceTemplate{
				Action:   "{{workflow.parameters.sys-resource-action}}",
				Manifest: containersManifest,
			},
		},
		{
			Name: "delete-pvc-resource",
			Inputs: wfv1.Inputs{
				Parameters: []wfv1.Parameter{{Name: "sys-pvc-name"}},
			},
			Resource: &wfv1.ResourceTemplate{
				Action:   "{{workflow.parameters.sys-resource-action}}",
				Manifest: deletePVCManifest,
			},
		},
	}
	// Add curl template
	curlPath := fmt.Sprintf("/apis/v1beta1/{{workflow.namespace}}/workspaces/{{workflow.parameters.sys-uid}}/status")
	status := map[string]interface{}{
		"phase": "{{inputs.parameters.sys-workspace-phase}}",
	}
	statusBytes, err := json.Marshal(status)
	if err != nil {
		return
	}
	inputs := wfv1.Inputs{
		Parameters: []wfv1.Parameter{
			{Name: "sys-workspace-phase"},
		},
	}
	curlNodeTemplate, err := getCURLNodeTemplate("sys-update-status", http.MethodPut, curlPath, string(statusBytes), inputs)
	if err != nil {
		return
	}
	templates = append(templates, *curlNodeTemplate)
	// Add postExecutionWorkflow if it exists
	if spec.PostExecutionWorkflow != nil {
		templates = append(templates, spec.PostExecutionWorkflow.Templates...)
	}

	workflowTemplateSpec := map[string]interface{}{
		"arguments":  spec.Arguments,
		"entrypoint": "workspace",
		"templates":  templates,
	}

	workflowTemplateSpecManifestBytes, err := yaml.Marshal(workflowTemplateSpec)
	if err != nil {
		return
	}
	workflowTemplateSpecManifest = string(workflowTemplateSpecManifestBytes)

	return
}

func (c *Client) createWorkspaceTemplate(namespace string, workspaceTemplate *WorkspaceTemplate) (*WorkspaceTemplate, error) {
	uid, err := workspaceTemplate.GenerateUID()
	if err != nil {
		return nil, err
	}

	tx, err := c.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	workspaceTemplate.WorkflowTemplate.IsSystem = true
	workspaceTemplate.WorkflowTemplate, err = c.CreateWorkflowTemplate(namespace, workspaceTemplate.WorkflowTemplate)
	if err != nil {
		return nil, err
	}
	workspaceTemplate.Version = workspaceTemplate.WorkflowTemplate.Version
	workspaceTemplate.IsLatest = true

	err = sb.Insert("workspace_templates").
		SetMap(sq.Eq{
			"uid":                  uid,
			"name":                 workspaceTemplate.Name,
			"namespace":            namespace,
			"workflow_template_id": workspaceTemplate.WorkflowTemplate.ID,
		}).
		Suffix("RETURNING id, created_at").
		RunWith(tx).
		QueryRow().Scan(&workspaceTemplate.ID, &workspaceTemplate.CreatedAt)
	if err != nil {
		_, err := c.ArchiveWorkflowTemplate(namespace, workspaceTemplate.WorkflowTemplate.UID)
		return nil, util.NewUserErrorWrap(err, "Workspace template")
	}

	_, err = sb.Insert("workspace_template_versions").
		SetMap(sq.Eq{
			"version":               workspaceTemplate.Version,
			"is_latest":             workspaceTemplate.IsLatest,
			"manifest":              workspaceTemplate.Manifest,
			"workspace_template_id": workspaceTemplate.ID,
		}).
		RunWith(tx).
		Exec()
	if err != nil {
		_, err := c.ArchiveWorkflowTemplate(namespace, workspaceTemplate.WorkflowTemplate.UID)
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		_, err := c.ArchiveWorkflowTemplate(namespace, workspaceTemplate.WorkflowTemplate.UID)
		return nil, err
	}

	return workspaceTemplate, nil
}

func (c *Client) workspaceTemplatesSelectBuilder(namespace string) sq.SelectBuilder {
	sb := sb.Select("wt.id", "wt.created_at", "wt.uid", "wt.name", `wt.workflow_template_id "workflow_template.id"`).
		From("workspace_templates wt").
		Where(sq.Eq{
			"wt.namespace": namespace,
		})

	return sb
}

func (c *Client) workspaceTemplateVersionsSelectBuilder(namespace, uid string) sq.SelectBuilder {
	sb := c.workspaceTemplatesSelectBuilder(namespace).
		Columns("wtv.created_at \"created_at\"", "wtv.version", "wtv.manifest", "wft.id \"workflow_template.id\"", "wft.uid \"workflow_template.uid\"", "wftv.version \"workflow_template.version\"", "wftv.manifest \"workflow_template.manifest\"").
		Join("workspace_template_versions wtv ON wtv.workspace_template_id = wt.id").
		Join("workflow_templates wft ON wft.id = wt.workflow_template_id").
		Join("workflow_template_versions wftv ON wftv.workflow_template_id = wft.id").
		Where(sq.Eq{"wt.uid": uid})

	return sb
}

func (c *Client) getWorkspaceTemplateByName(namespace, name string) (workspaceTemplate *WorkspaceTemplate, err error) {
	workspaceTemplate = &WorkspaceTemplate{}

	sb := c.workspaceTemplatesSelectBuilder(namespace).
		Where(sq.Eq{"wt.name": name}).
		Limit(1)
	query, args, err := sb.ToSql()
	if err != nil {
		return
	}

	if err = c.DB.Get(workspaceTemplate, query, args...); err == sql.ErrNoRows {
		err = nil
		workspaceTemplate = nil
	}

	return
}

func (c *Client) generateWorkspaceTemplateWorkflowTemplate(workspaceTemplate *WorkspaceTemplate) (workflowTemplate *WorkflowTemplate, err error) {
	if workspaceTemplate == nil || workspaceTemplate.Manifest == "" {
		return nil, util.NewUserError(codes.InvalidArgument, "Workspace template manifest is required")
	}

	config, err := c.GetSystemConfig()
	if err != nil {
		return nil, err
	}

	workspaceSpec, err := parseWorkspaceSpec(workspaceTemplate.Manifest)
	if err != nil {
		return nil, err
	}

	if err = generateArguments(workspaceSpec, config); err != nil {
		return nil, err
	}

	serviceManifest, err := createServiceManifest(workspaceSpec)
	if err != nil {
		return nil, err
	}

	virtualServiceManifest, err := createVirtualServiceManifest(workspaceSpec)
	if err != nil {
		return nil, err
	}

	containersManifest, err := createStatefulSetManifest(workspaceSpec, config)
	if err != nil {
		return nil, err
	}

	workflowTemplateManifest, err := unmarshalWorkflowTemplate(workspaceSpec, serviceManifest, virtualServiceManifest, containersManifest)
	if err != nil {
		return nil, err
	}

	workflowTemplate = &WorkflowTemplate{
		Name:     workspaceTemplate.Name,
		Manifest: workflowTemplateManifest,
	}

	return workflowTemplate, nil
}

// CreateWorkspaceTemplateWorkflowTemplate generates and returns a workflowTemplate for a given workspaceTemplate manifest
func (c *Client) GenerateWorkspaceTemplateWorkflowTemplate(workspaceTemplate *WorkspaceTemplate) (workflowTemplate *WorkflowTemplate, err error) {
	workflowTemplate, err = c.generateWorkspaceTemplateWorkflowTemplate(workspaceTemplate)
	if err != nil {
		return nil, err
	}

	return workflowTemplate, nil
}

// CreateWorkspaceTemplate creates a template for Workspaces
func (c *Client) CreateWorkspaceTemplate(namespace string, workspaceTemplate *WorkspaceTemplate) (*WorkspaceTemplate, error) {
	valid, err := govalidator.ValidateStruct(workspaceTemplate)
	if err != nil || !valid {
		return nil, util.NewUserError(codes.InvalidArgument, err.Error())
	}

	existingWorkspaceTemplate, err := c.getWorkspaceTemplateByName(namespace, workspaceTemplate.Name)
	if err != nil {
		return nil, err
	}
	if existingWorkspaceTemplate != nil {
		return nil, util.NewUserError(codes.AlreadyExists, "Workspace template already exists.")
	}

	workspaceTemplate.WorkflowTemplate, err = c.generateWorkspaceTemplateWorkflowTemplate(workspaceTemplate)
	if err != nil {
		return nil, err
	}

	workspaceTemplate, err = c.createWorkspaceTemplate(namespace, workspaceTemplate)
	if err != nil {
		return nil, err
	}

	return workspaceTemplate, nil
}

// GetWorkspaceTemplate return a workspaceTemplate and its corresponding workflowTemplate
func (c *Client) GetWorkspaceTemplate(namespace, uid string, version int64) (workspaceTemplate *WorkspaceTemplate, err error) {
	workspaceTemplate = &WorkspaceTemplate{}
	sb := c.workspaceTemplateVersionsSelectBuilder(namespace, uid).
		Limit(1)
	if version == 0 {
		sb = sb.Where(sq.Eq{
			"wtv.is_latest":  true,
			"wftv.is_latest": true,
		})
	} else {
		sb = sb.Where(sq.Eq{
			"wtv.version":  version,
			"wftv.version": version,
		})
	}
	query, args, err := sb.ToSql()
	if err != nil {
		return
	}
	if err = c.DB.Get(workspaceTemplate, query, args...); err == sql.ErrNoRows {
		return
	}

	return
}

// UpdateWorkspaceTemplate adds a new workspace template version
func (c *Client) UpdateWorkspaceTemplate(namespace string, workspaceTemplate *WorkspaceTemplate) (*WorkspaceTemplate, error) {
	existingWorkspaceTemplate, err := c.GetWorkspaceTemplate(namespace, workspaceTemplate.UID, workspaceTemplate.Version)
	if err != nil {
		return nil, err
	}
	if existingWorkspaceTemplate == nil {
		return nil, util.NewUserError(codes.NotFound, "Workspace template not found.")
	}
	workspaceTemplate.ID = existingWorkspaceTemplate.ID
	workspaceTemplate.Name = existingWorkspaceTemplate.UID

	updatedWorkflowTemplate, err := c.generateWorkspaceTemplateWorkflowTemplate(workspaceTemplate)
	if err != nil {
		return nil, err
	}
	updatedWorkflowTemplate.ID = existingWorkspaceTemplate.WorkflowTemplate.ID
	updatedWorkflowTemplate.UID = existingWorkspaceTemplate.WorkflowTemplate.UID

	tx, err := c.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	workflowTemplateVersion, err := c.CreateWorkflowTemplateVersion(namespace, updatedWorkflowTemplate)
	if err != nil {
		return nil, err
	}

	workspaceTemplate.Version = workflowTemplateVersion.Version
	workspaceTemplate.IsLatest = true

	_, err = sb.Update("workspace_template_versions").
		SetMap(sq.Eq{"is_latest": false}).
		Where(sq.Eq{
			"workspace_template_id": workspaceTemplate.ID,
		}).
		RunWith(tx).
		Exec()
	if err != nil {
		return nil, err
	}

	_, err = sb.Insert("workspace_template_versions").
		SetMap(sq.Eq{
			"version":               workspaceTemplate.Version,
			"is_latest":             workspaceTemplate.IsLatest,
			"manifest":              workspaceTemplate.Manifest,
			"workspace_template_id": workspaceTemplate.ID,
		}).
		RunWith(tx).
		Exec()
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return workspaceTemplate, nil
}

func (c *Client) ListWorkspaceTemplates(namespace string, paginator *pagination.PaginationRequest) (workspaceTemplates []*WorkspaceTemplate, err error) {
	sb := c.workspaceTemplatesSelectBuilder(namespace).
		OrderBy("wt.created_at DESC")
	sb = *paginator.ApplyToSelect(&sb)

	query, args, err := sb.ToSql()
	if err != nil {
		return nil, err
	}

	if err := c.DB.Select(&workspaceTemplates, query, args...); err != nil {
		return nil, err
	}

	return
}

func (c *Client) ListWorkspaceTemplateVersions(namespace, uid string) (workspaceTemplates []*WorkspaceTemplate, err error) {
	sb := c.workspaceTemplateVersionsSelectBuilder(namespace, uid).
		Options("DISTINCT ON (wtv.version) wtv.version,").
		Where(sq.Eq{
			"wt.is_archived":  false,
			"wft.is_archived": false,
		}).
		OrderBy("wtv.version DESC")
	query, args, err := sb.ToSql()
	if err != nil {
		return
	}
	if err = c.DB.Select(&workspaceTemplates, query, args...); err != nil {
		return
	}

	return
}

func (c *Client) CountWorkspaceTemplates(namespace string) (count int, err error) {
	err = sb.Select("count(*)").
		From("workspace_templates wt").
		Where(sq.Eq{
			"wt.namespace": namespace,
		}).RunWith(c.DB).
		QueryRow().
		Scan(&count)

	return
}
