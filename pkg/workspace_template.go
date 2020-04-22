package v1

import (
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"log"

	networking "istio.io/api/networking/v1alpha3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

func parseServicePorts(template string) (servicePorts []corev1.ServicePort, err error) {
	if err = yaml.UnmarshalStrict([]byte(template), &servicePorts); err != nil {
		return
	}

	return
}

func parseHTTPRoutes(template string) (HTTPRoutes []*networking.HTTPRoute, err error) {
	if err = yaml.UnmarshalStrict([]byte(template), &HTTPRoutes); err != nil {
		return
	}

	return
}

func parseVolumeClaims(template string) (persistentVolumeClaims []corev1.PersistentVolumeClaim, err error) {
	if err = yaml.UnmarshalStrict([]byte(template), &persistentVolumeClaims); err != nil {
		return
	}

	return
}

func parseContainers(template string) (containers []corev1.Container, err error) {
	if err = yaml.UnmarshalStrict([]byte(template), &containers); err != nil {
		return
	}

	return
}

func createServiceManifest(portsManifest string) (serviceManifest string, err error) {
	servicePorts, err := parseServicePorts(portsManifest)
	if err != nil {
		return
	}
	service := corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "{{workflow.parameters.name}}",
		},
		Spec: corev1.ServiceSpec{
			Ports: servicePorts,
			Selector: map[string]string{
				"app": "{{workflow.parameters.name}}",
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

func createVirtualServiceManifest(routesManifest string) (virtualServiceManifest string, err error) {
	httpRoutes, err := parseHTTPRoutes(routesManifest)
	if err != nil {
		return
	}

	for _, hr := range httpRoutes {
		for _, r := range hr.Route {
			r.Destination.Host = "{{workflow.parameters.name}}"
		}
	}

	virtualService := struct {
		metav1.TypeMeta
		metav1.ObjectMeta
		Spec networking.VirtualService `json:"spec"`
	}{
		metav1.TypeMeta{
			APIVersion: "networking.istio.io/v1alpha3",
			Kind:       "VirtualService",
		},
		metav1.ObjectMeta{
			Name: "{{workflow.parameters.name}}",
		},
		networking.VirtualService{
			Http: httpRoutes,
		},
	}

	virtualServiceManifestBytes, err := yaml.Marshal(virtualService)
	if err != nil {
		return
	}
	virtualServiceManifest = string(virtualServiceManifestBytes)

	return
}

func unmarshalWorkflowTemplate(serviceManifest, virtualServiceManifest string) (workflowTemplateManifest string, err error) {
	workflowTemplate := wfv1.WorkflowTemplate{
		Spec: wfv1.WorkflowTemplateSpec{
			WorkflowSpec: wfv1.WorkflowSpec{
				Arguments: wfv1.Arguments{},
				Templates: []wfv1.Template{
					{
						Name: "create-workspace",
						DAG: &wfv1.DAGTemplate{
							Tasks: []wfv1.DAGTask{
								{
									Name:     "create-service",
									Template: "create-service-resource",
								},
								{
									Name:     "create-virtual-service",
									Template: "create-virtual-service-resource",
								},
							},
						},
					},
					{
						Name: "create-service-resource",
						Resource: &wfv1.ResourceTemplate{
							Action:   "{{workflow.parameters.action}}",
							Manifest: serviceManifest,
						},
					},
					{
						Name: "create-virtual-service-resource",
						Resource: &wfv1.ResourceTemplate{
							Action:   "{{workflow.parameters.action}}",
							Manifest: virtualServiceManifest,
						},
					},
				},
			},
		},
	}
	workflowTemplateManifestBytes, err := yaml.Marshal(workflowTemplate)
	if err != nil {
		return
	}
	workflowTemplateManifest = string(workflowTemplateManifestBytes)

	return
}

// CreateWorkspaceTemplate creates a template for Workspaces
func (c *Client) CreateWorkspaceTemplate(namespace string, workspaceTemplate WorkspaceTemplate) (err error) {
	//parameters := workspaceTemplate.Spec.Parameters
	serviceManifest, err := createServiceManifest(workspaceTemplate.PortsManifest)
	if err != nil {
		return
	}

	virtualServiceManifest, err := createVirtualServiceManifest(workspaceTemplate.RoutesManifest)
	if err != nil {
		return
	}

	workflowTemplateManifest, err := unmarshalWorkflowTemplate(serviceManifest, virtualServiceManifest)
	if err != nil {
		return
	}

	//_, err = c.CreateWorkflowTemplate(namespace, &WorkflowTemplate{
	//	Name: "test",
	//	Manifest: string(workflowManifest),
	//})
	log.Print(string(workflowTemplateManifest))

	return
}
