package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"time"
)

const inferenceServiceResource = "InferenceServices"

// ResourceLimits are the cpu/memory limits
type ResourceLimits struct {
	CPU    string
	Memory string
}

// Predictor contains information on what type of predictor we are using, and what resources it has available
type Predictor struct {
	Name           string
	RuntimeVersion *string
	StorageURI     string
	ResourceLimits *ResourceLimits
	NodeSelector   *string
}

// Env is a name/value environment variable
type Env struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// TransformerContainer is a container specific to a Transformer
type TransformerContainer struct {
	Image string `json:"image"`
	Name  string `json:"name"`
	Env   []Env  `json:"env"`
}

// Transformer is a unit that can convert model input and output to different formats in json
type Transformer struct {
	Containers []TransformerContainer
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
}

type inferenceServiceCondition struct {
	LastTransitionTime time.Time `json:"lastTransitionTime"`
	Status             string    `json:"status"`
	Type               string    `json:"type"`
}

type inferenceServiceStatus struct {
	Conditions []inferenceServiceCondition `json:"conditions"`
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

func (k k8sModel) DeepCopyObject() runtime.Object {
	panic("implement me")
}

// ToResource converts the InferenceService into a KFServing spec
func (m *InferenceService) ToResource(nodeSelector string) interface{} {
	spec := map[string]interface{}{
		"predictor": map[string]interface{}{
			m.Predictor.Name: map[string]interface{}{
				"storageUri": m.Predictor.StorageURI,
			},
		},
	}

	predictor := spec["predictor"].(map[string]interface{})
	predictorServer := predictor[m.Predictor.Name].(map[string]interface{})

	if m.Predictor.RuntimeVersion != nil {
		predictorServer["runtimeVersion"] = m.Predictor.RuntimeVersion
	}

	if m.Predictor.NodeSelector != nil {
		predictor["nodeSelector"] = map[string]string{
			nodeSelector: *m.Predictor.NodeSelector,
		}
	}

	if m.Predictor.ResourceLimits != nil {
		predictorServer["resources"] = map[string]string{
			"cpu":    m.Predictor.ResourceLimits.CPU,
			"memory": m.Predictor.ResourceLimits.Memory,
		}
	}

	if m.Transformer != nil {
		spec["transformer"] = map[string]interface{}{
			"containers": m.Transformer.Containers,
		}
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
