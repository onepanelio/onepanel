package v1

import (
	"fmt"
	"github.com/onepanelio/core/pkg/util"
	"google.golang.org/grpc/codes"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"strings"
)

const istioVirtualServiceResource = "VirtualServices"

func istioModelRestClient() (*rest.RESTClient, error) {
	config := *NewConfig()
	config.GroupVersion = &schema.GroupVersion{Group: "networking.istio.io", Version: "v1alpha3"}
	config.APIPath = "/apis"
	config.NegotiatedSerializer = scheme.Codecs.WithoutConversion()

	return rest.RESTClientFor(&config)
}

// CreateVirtualService creates an istio virtual service
func (c *Client) CreateVirtualService(namespace string, data interface{}) error {
	restClient, err := istioModelRestClient()
	if err != nil {
		return err
	}

	err = restClient.Post().
		Namespace(namespace).
		Resource(istioVirtualServiceResource).
		Body(data).
		Do().
		Error()

	if err != nil && strings.Contains(err.Error(), "already exists") {
		return util.NewUserError(codes.AlreadyExists, fmt.Sprintf("VirtualService already exists"))
	}

	return err
}
