package v1

import (
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/onepanelio/core/pkg/util/ptr"
	"k8s.io/apimachinery/pkg/api/resource"
	"log"

	networking "istio.io/api/networking/v1alpha3"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

func parsePorts(template string) (servicePorts []corev1.ServicePort, err error) {
	if err = yaml.UnmarshalStrict([]byte(template), &servicePorts); err != nil {
		return
	}

	return
}

func parseRoutes(template string) (HTTPRoutes []*networking.HTTPRoute, err error) {
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
	servicePorts, err := parsePorts(portsManifest)
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
	httpRoutes, err := parseRoutes(routesManifest)
	if err != nil {
		return
	}

	for _, h := range httpRoutes {
		for _, r := range h.Route {
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
			Http:     httpRoutes,
			Gateways: []string{"istio-system/ingressgateway"},
			// TODO: This should be generated when workspace is launched
			//Hosts: []string{"{{workflow.parameters.name}}-{{workflow.namespace}}.{{}}"},
		},
	}

	virtualServiceManifestBytes, err := yaml.Marshal(virtualService)
	if err != nil {
		return
	}
	virtualServiceManifest = string(virtualServiceManifestBytes)

	return
}

func createStatefulSetManifest(containersManifest string) (statefulSetManifest string, err error) {
	containers, err := parseContainers(containersManifest)
	if err != nil {
		return
	}

	volumeClaims := []corev1.PersistentVolumeClaim{}
	volumeClaimsMapped := make(map[string]bool)
	for _, c := range containers {
		for _, v := range c.VolumeMounts {
			if volumeClaimsMapped[v.Name] {
				continue
			}

			volumeClaims = append(volumeClaims, corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name: v.Name,
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{
						"ReadWriteOnce",
					},
					StorageClassName: ptr.String("default"),
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							// TODO: Need to get this value from {{workflow.parameters.<volume-name>-size}}
							"storage": resource.Quantity{},
						},
					},
				},
			})
			volumeClaimsMapped[v.Name] = true
		}
	}

	statefulSet := appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "StatefulSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "{{workflow.parameters.name}}",
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas:    ptr.Int32(1),
			ServiceName: "{{workflow.parameters.name}}",
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "{{workflow.parameters.name}}",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "{{workflow.parameters.name}}",
					},
				},
				Spec: corev1.PodSpec{
					// TODO: This should be generated when workspace is launched
					//NodeSelector: map[string]string{
					//	"{{}}": "{{workflow.parameters.node-pool}}",
					//},
					Containers: containers,
				},
			},
			VolumeClaimTemplates: volumeClaims,
		},
	}

	statefulSetManifestBytes, err := yaml.Marshal(statefulSet)
	if err != nil {
		return
	}
	statefulSetManifest = string(statefulSetManifestBytes)

	return
}

func unmarshalWorkflowTemplate(serviceManifest, virtualServiceManifest, containersManifest string) (workflowTemplateManifest string, err error) {
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
								{
									Name:     "create-stateful-set",
									Template: "create-stateful-set-resource",
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
					{
						Name: "create-stateful-set-resource",
						Resource: &wfv1.ResourceTemplate{
							Action:   "{{workflow.parameters.action}}",
							Manifest: containersManifest,
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

	containersManifest, err := createStatefulSetManifest(workspaceTemplate.ContainersManifest)
	if err != nil {
		return
	}

	workflowTemplateManifest, err := unmarshalWorkflowTemplate(serviceManifest, virtualServiceManifest, containersManifest)
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
