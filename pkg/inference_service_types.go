package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"time"
)

const inferenceServiceResource = "InferenceServices"

// MachineResources are the cpu/memory limits
type MachineResources struct {
	CPU    string `json:"cpu,omitempty"`
	Memory string `json:"memory,omitempty"`
}

// Resources represent the machine resource requests/limits
type Resources struct {
	Limits   *MachineResources `json:"limits,omitempty"`
	Requests *MachineResources `json:"requests,omitempty"`
}

// Predictor contains information on what type of predictor we are using, and what resources it has available
type Predictor struct {
	Name           string            `json:"-"`
	RuntimeVersion string            `json:"runtimeVersion,omitempty"`
	StorageURI     string            `json:"storageUri"`
	Resources      *Resources        `json:"resources,omitempty"`
	NodeSelector   map[string]string `json:"nodeSelector,omitempty"`
}

// SetResources will set the cpu/memory requests/limits for the predictor. Empty strings are ignored
func (p *Predictor) SetResources(minCPU, maxCPU, minMemory, maxMemory string) {
	if minCPU == "" && maxCPU == "" && minMemory == "" && maxMemory == "" {
		return
	}

	p.Resources = &Resources{}
	if minCPU != "" || minMemory != "" {
		p.Resources.Requests = &MachineResources{
			CPU:    minCPU,
			Memory: minMemory,
		}
	}

	if maxCPU != "" || maxMemory != "" {
		p.Resources.Limits = &MachineResources{
			CPU:    maxCPU,
			Memory: maxMemory,
		}
	}
}

// SetNodeSelector will set the node selector to the input label: selector value
func (p *Predictor) SetNodeSelector(label, selector string) {
	p.NodeSelector = map[string]string{
		label: selector,
	}
}

// Env is a name/value environment variable
type Env struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// TransformerContainer is a container specific to a Transformer
type TransformerContainer struct {
	Image     string     `json:"image"`
	Name      string     `json:"name"`
	Env       []Env      `json:"env"`
	Resources *Resources `json:"resources,omitempty"`
}

// Transformer is a unit that can convert model input and output to different formats in json
type Transformer struct {
	Containers []TransformerContainer `json:"containers"`
}

// InferenceService represents the information necessary to deploy an inference service
type InferenceService struct {
	Name      string
	Namespace string

	Transformer *Transformer
	Predictor   *Predictor
}

// InferenceServiceStatus represents information about an InferenceService
type InferenceServiceStatus struct {
	Ready      bool
	Conditions []inferenceServiceCondition
	PredictURL string
}

type inferenceServiceCondition struct {
	LastTransitionTime time.Time `json:"lastTransitionTime"`
	Status             string    `json:"status"`
	Type               string    `json:"type"`
}

type inferenceServiceAddress struct {
	URL string `json:"url"`
}

type inferenceServiceStatus struct {
	Conditions []inferenceServiceCondition `json:"conditions"`
	Address    inferenceServiceAddress     `json:"address"`
	URL        string                      `json:"url"`
}

// Ready returns true if there is a condition called Ready: true.
func (m *inferenceServiceStatus) Ready() bool {
	for _, condition := range m.Conditions {
		if condition.Type == "Ready" && condition.Status == "True" {
			return true
		}
	}

	return false
}

type k8sModel struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`
	Status            inferenceServiceStatus `json:"status,omitempty"`
}

// DeepCopyObject is a stub to support the interface
func (k k8sModel) DeepCopyObject() runtime.Object {
	panic("implement me")
}

// ToResource converts the InferenceService into a KFServing spec
func (m *InferenceService) ToResource() interface{} {
	spec := map[string]interface{}{
		"predictor": map[string]interface{}{
			m.Predictor.Name: m.Predictor,
		},
	}

	if m.Transformer != nil {
		spec["transformer"] = m.Transformer
	}

	resource := map[string]interface{}{
		"apiVersion": "serving.kubeflow.org/v1beta1",
		"kind":       "InferenceService",
		"metadata": map[string]string{
			"namespace": m.Namespace,
			"name":      m.Name,
		},
		"spec": spec,
	}

	return resource
}
