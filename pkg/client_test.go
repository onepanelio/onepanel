package v1

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

func NewTestClient(objects ...runtime.Object) (client *Client) {
	return &Client{Interface: fake.NewSimpleClientset(objects...)}
}
