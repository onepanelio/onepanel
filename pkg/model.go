package v1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

func modelRestClient() (*rest.RESTClient, error) {
	config := *NewConfig()
	config.GroupVersion = &schema.GroupVersion{Group: "serving.kubeflow.org", Version: "v1beta1"}
	config.APIPath = "/apis"
	config.NegotiatedSerializer = scheme.Codecs.WithoutConversion()

	return rest.RESTClientFor(&config)
}

// DeployModel creates an InferenceService with KFServing
func (c *Client) DeployModel(deployment *ModelDeployment) error {
	resource := deployment.ToResource()

	data, err := json.Marshal(resource)
	if err != nil {
		return err
	}

	restClient, err := modelRestClient()
	if err != nil {
		return err
	}

	err = restClient.Post().
		Namespace(deployment.Namespace).
		Name(deployment.Name).
		Resource(modelResource).
		Body(data).
		Do().
		Error()

	return err
}

// GetModelStatus returns the model's status
func (c *Client) GetModelStatus(namespace, name string) (*ModelStatus, error) {
	restClient, err := modelRestClient()
	if err != nil {
		return nil, err
	}

	result := &k8sModel{}

	err = restClient.Get().
		Namespace(namespace).
		Name(name).
		Resource(modelResource).
		Do().
		Into(result)

	status := &ModelStatus{
		Conditions: result.Status.Conditions,
		Ready:      result.Status.Ready(),
	}

	return status, err
}

// DeleteModel deletes the model
func (c *Client) DeleteModel(namespace, name string) error {
	restClient, err := modelRestClient()
	if err != nil {
		return err
	}

	return restClient.Delete().
		Namespace(namespace).
		Name(name).
		Resource(modelResource).
		Do().
		Error()
}
