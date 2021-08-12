package v1

import (
	"fmt"
	"github.com/onepanelio/core/pkg/util"
	"google.golang.org/grpc/codes"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"strings"
)

func modelRestClient() (*rest.RESTClient, error) {
	config := *NewConfig()
	config.GroupVersion = &schema.GroupVersion{Group: "serving.kubeflow.org", Version: "v1beta1"}
	config.APIPath = "/apis"
	config.NegotiatedSerializer = scheme.Codecs.WithoutConversion()

	return rest.RESTClientFor(&config)
}

// CreateInferenceService creates an InferenceService with KFServing
func (c *Client) CreateInferenceService(deployment *InferenceService) error {
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
		Resource(inferenceServiceResource).
		Body(data).
		Do().
		Error()

	if err != nil && strings.Contains(err.Error(), "already exists") {
		return util.NewUserError(codes.AlreadyExists, fmt.Sprintf("InferenceService with name '%v' already exists", deployment.Name))
	}

	return err
}

// GetModelStatus returns the model's status
func (c *Client) GetModelStatus(namespace, name string) (*InferenceServiceStatus, error) {
	restClient, err := modelRestClient()
	if err != nil {
		return nil, err
	}

	result := &k8sModel{}

	err = restClient.Get().
		Namespace(namespace).
		Name(name).
		Resource(inferenceServiceResource).
		Do().
		Into(result)

	if err != nil && strings.Contains(err.Error(), "not found") {
		return nil, util.NewUserError(codes.NotFound, "not found")
	}

	predictURL := result.Status.URL
	suffixIndex := strings.LastIndex(result.Status.Address.URL, "cluster.local")
	if suffixIndex >= 0 {
		predictURL += result.Status.Address.URL[suffixIndex+13:]
	}

	status := &InferenceServiceStatus{
		Conditions: result.Status.Conditions,
		Ready:      result.Status.Ready(),
		PredictURL: predictURL,
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
		Resource(inferenceServiceResource).
		Do().
		Error()
}
