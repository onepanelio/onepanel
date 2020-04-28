package v1

import (
	"database/sql"
	"fmt"
	sq "github.com/Masterminds/squirrel"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	v1 "github.com/onepanelio/core/pkg/apis/core/v1"
	"github.com/onepanelio/core/pkg/util"
	"github.com/onepanelio/core/pkg/util/pagination"
	"github.com/onepanelio/core/pkg/util/ptr"
	"google.golang.org/grpc/codes"
	networking "istio.io/api/networking/v1alpha3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

func parseWorkspaceSpec(template string) (spec *v1.WorkspaceSpec, err error) {
	err = yaml.UnmarshalStrict([]byte(template), &spec)

	return
}

func generateArguments(spec *v1.WorkspaceSpec, config map[string]string) (err error) {
	if spec.Arguments == nil {
		spec.Arguments = &v1.Arguments{
			Parameters: []v1.Parameter{},
		}
	}

	// Resource action parameter
	spec.Arguments.Parameters = append(spec.Arguments.Parameters, v1.Parameter{
		Name:     "sys-name",
		Type:     "input.text",
		Value:    ptr.String("name"),
		Required: true,
	})

	// Resource action parameter
	spec.Arguments.Parameters = append(spec.Arguments.Parameters, v1.Parameter{
		Name:  "sys-resource-action",
		Value: ptr.String("apply"),
		Type:  "input.hidden",
	})

	// Workspace action
	spec.Arguments.Parameters = append(spec.Arguments.Parameters, v1.Parameter{
		Name:  "sys-workspace-action",
		Value: ptr.String("create"),
		Type:  "input.hidden",
	})

	// Node pool parameter and options
	var options []*v1.ParameterOption
	if err = yaml.Unmarshal([]byte(config["applicationNodePoolOptions"]), &options); err != nil {
		return
	}
	spec.Arguments.Parameters = append(spec.Arguments.Parameters, v1.Parameter{
		Name:     "sys-node-pool",
		Value:    ptr.String(options[0].Value),
		Type:     "select.select",
		Options:  options,
		Required: true,
	})

	// Volume size parameters
	volumeClaimsMapped := make(map[string]bool)
	for _, c := range spec.Containers {
		for _, v := range c.VolumeMounts {
			if volumeClaimsMapped[v.Name] {
				continue
			}

			spec.Arguments.Parameters = append(spec.Arguments.Parameters, v1.Parameter{
				Name:     fmt.Sprintf("sys-%v-volume-size", v.Name),
				Type:     "input.number",
				Value:    ptr.String("20480"),
				Required: true,
			})

			volumeClaimsMapped[v.Name] = true
		}
	}

	return
}

func createServiceManifest(spec *v1.WorkspaceSpec) (serviceManifest string, err error) {
	service := corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "{{workflow.parameters.sys-name}}",
		},
		Spec: corev1.ServiceSpec{
			Ports: spec.Ports,
			Selector: map[string]string{
				"app": "{{workflow.parameters.sys-name}}",
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

func createVirtualServiceManifest(spec *v1.WorkspaceSpec, config map[string]string) (virtualServiceManifest string, err error) {
	for _, h := range spec.Routes {
		for _, r := range h.Route {
			r.Destination.Host = "{{workflow.parameters.sys-name}}"
		}
	}
	virtualService := map[string]interface{}{
		"apiVersion": "networking.istio.io/v1alpha3",
		"kind":       "VirtualService",
		"metadata": metav1.ObjectMeta{
			Name: "{{workflow.parameters.sys-name}}",
		},
		"spec": networking.VirtualService{
			Http:     spec.Routes,
			Gateways: []string{"istio-system/ingressgateway"},
			Hosts:    []string{fmt.Sprintf("{{workflow.parameters.sys-name}}-{{workflow.namespace}}.%v", config["ONEPANEL_DOMAIN"])},
		},
	}
	virtualServiceManifestBytes, err := yaml.Marshal(virtualService)
	if err != nil {
		return
	}
	virtualServiceManifest = string(virtualServiceManifestBytes)

	return
}

func createStatefulSetManifest(workspaceSpec *v1.WorkspaceSpec, config map[string]string) (statefulSetManifest string, err error) {
	var volumeClaims []map[string]interface{}
	volumeClaimsMapped := make(map[string]bool)
	for _, c := range workspaceSpec.Containers {
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
			Name: "{{workflow.parameters.sys-name}}",
		},
		"spec": map[string]interface{}{
			"replicas":    1,
			"serviceName": "{{workflow.parameters.sys-name}}",
			"selector": &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "{{workflow.parameters.sys-name}}",
				},
			},
			"template": corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "{{workflow.parameters.sys-name}}",
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

func unmarshalWorkflowTemplate(spec *v1.WorkspaceSpec, serviceManifest, virtualServiceManifest, containersManifest string) (workflowTemplateSpecManifest string, err error) {
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
  name: {{inputs.parameters.sys-pvc-name}}-{{workflow.parameters.sys-name}}-0
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
	if spec.PostExecutionWorkflow != nil {
		templates = append(templates, spec.PostExecutionWorkflow.Templates...)
	}

	// TODO: Consider storing this as a Go template in a "settings" database table
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
	sb := sb.Select("wt.id", "wt.created_at", "wt.uid", "wt.name").
		From("workspace_templates wt").
		Where(sq.Eq{
			"wt.namespace": namespace,
		})

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

	virtualServiceManifest, err := createVirtualServiceManifest(workspaceSpec, config)
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
		Manifest: string(workflowTemplateManifest),
	}

	return workflowTemplate, nil
}

// GetWorkspaceTemplateWorkflowTemplate generates and returns a workflowTemplate for a given workspaceTemplate manifest
func (c *Client) GetWorkspaceTemplateWorkflowTemplate(workspaceTemplate *WorkspaceTemplate) (*WorkspaceTemplate, error) {
	workflowTemplate, err := c.generateWorkspaceTemplateWorkflowTemplate(workspaceTemplate)
	if err != nil {
		return nil, err
	}
	workspaceTemplate.WorkflowTemplate = workflowTemplate

	return workspaceTemplate, nil
}

// CreateWorkspaceTemplate creates a template for Workspaces
func (c *Client) CreateWorkspaceTemplate(namespace string, workspaceTemplate *WorkspaceTemplate) (*WorkspaceTemplate, error) {
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

func (c *Client) ListWorkspaceTemplates(namespace string, paginator *pagination.PaginationRequest) (workspaceTemplates []*WorkspaceTemplate, err error) {
	workspaceTemplates = make([]*WorkspaceTemplate, 0)
	sb := c.workspaceTemplatesSelectBuilder(namespace).
		OrderBy("wt.created_at DESC")
	paginator.ApplyToSelect(&sb)

	query, args, err := sb.ToSql()
	if err != nil {
		return nil, err
	}

	if err := c.DB.Select(&workspaceTemplates, query, args...); err != nil {
		return nil, err
	}

	return
}

func (c *Client) CountWorkspaceTemplates(namespace string) (count int, err error) {
	err = sb.Select("count(*)").
		From("workspace_templates wt").
		Where(sq.Eq{
			"wt.namespace": namespace,
		}).RunWith(c.DB.DB).
		QueryRow().
		Scan(&count)

	return
}
