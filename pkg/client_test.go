package v1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

var (
	fakeSystemSecret = &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "onepanel",
			Namespace: "onepanel",
		},
	}

	fakeSystemConfigMap = &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "onepanel",
			Namespace: "onepanel",
		},
		Data: map[string]string{
			"ONEPANEL_HOST":            "demo.onepanel.site",
			"applicationNodePoolLabel": "beta.kubernetes.io/instance-type",
		},
	}
)

func NewTestClient(objects ...runtime.Object) (client *Client) {
	return &Client{Interface: fake.NewSimpleClientset(objects...)}
}
