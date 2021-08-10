package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"time"
)

const modelResource = "InferenceServices"

// ResourceLimits are the cpu/memory limits
type ResourceLimits struct {
	CPU    string
	Memory string
}

// NodeSelector provides a key/value to select a Node
type NodeSelector struct {
	Key   string
	Value string
}

// PredictorServer contains information on a server that serves models
type PredictorServer struct {
	Name           string
	RuntimeVersion *string
	StorageURI     string
	ResourceLimits *ResourceLimits
}

// Predictor contains information on what type of predictor we are using, and what resources it has available
type Predictor struct {
	NodeSelector *NodeSelector
	Server       PredictorServer
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

// ModelDeployment represents the information necessary to deploy a model
type ModelDeployment struct {
	Name      string
	Namespace string

	Transformer *Transformer
	Predictor   *Predictor
}

// ModelStatus represents information about a model's status
type ModelStatus struct {
	Ready      bool
	Conditions []modelCondition
}

type modelCondition struct {
	LastTransitionTime time.Time `json:"lastTransitionTime"`
	Status             string    `json:"status"`
	Type               string    `json:"type"`
}

type modelStatus struct {
	Conditions []modelCondition `json:"conditions"`
}

// Ready returns true if there is a condition called Ready: true.
func (m *modelStatus) Ready() bool {
	for _, condition := range m.Conditions {
		if condition.Type == "Ready" && condition.Status == "True" {
			return true
		}
	}

	return false
}

// TODO
// k8sModel
type k8sModel struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`
	Status            modelStatus `json:"status,omitempty"`
}

func (k k8sModel) DeepCopyObject() runtime.Object {
	panic("implement me")
}

// ToResource converts the ModelDeployment into a KFServing spec
func (m *ModelDeployment) ToResource() interface{} {
	spec := map[string]interface{}{
		"predictor": map[string]interface{}{
			m.Predictor.Server.Name: map[string]interface{}{
				"storageUri": m.Predictor.Server.StorageURI,
			},
		},
	}

	predictor := spec["predictor"].(map[string]interface{})
	predictorServer := predictor[m.Predictor.Server.Name].(map[string]interface{})

	if m.Predictor.Server.RuntimeVersion != nil {
		predictorServer["runtimeVersion"] = m.Predictor.Server.RuntimeVersion
	}

	if m.Predictor.NodeSelector != nil {
		predictor["nodeSelector"] = map[string]string{
			m.Predictor.NodeSelector.Key: m.Predictor.NodeSelector.Value,
		}
	}

	if m.Predictor.Server.ResourceLimits != nil {
		predictorServer["resources"] = map[string]string{
			"cpu":    m.Predictor.Server.ResourceLimits.CPU,
			"memory": m.Predictor.Server.ResourceLimits.Memory,
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
