package v1

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
	RuntimeVersion string
	StorageURI     string
	ResourceLimits ResourceLimits
	NodeSelector   NodeSelector
}

// ModelDeployment represents the information necessary to deploy a model
type ModelDeployment struct {
	Name      string
	Namespace string

	PredictorServer PredictorServer
}

// ToResource converts the ModelDeployment into a KFServing spec
func (m *ModelDeployment) ToResource() interface{} {
	resource := map[string]interface{}{
		"apiVersion": "serving.kubeflow.org/v1beta1",
		"kind":       "InferenceService",
		"metadata": map[string]string{
			"namespace": m.Namespace,
			"name":      m.Name,
		},
		"spec": map[string]interface{}{
			"predictor": map[string]interface{}{
				"nodeSelector": map[string]string{
					m.PredictorServer.NodeSelector.Key: m.PredictorServer.NodeSelector.Value,
				},
				m.PredictorServer.Name: map[string]interface{}{
					"resources": map[string]interface{}{
						"limits": map[string]string{
							"cpu":    m.PredictorServer.ResourceLimits.CPU,
							"memory": m.PredictorServer.ResourceLimits.Memory,
						},
					},
					"runtimeVersion": m.PredictorServer.RuntimeVersion,
					"storageUri":     m.PredictorServer.StorageURI,
				},
			},
		},
	}

	return resource
}
