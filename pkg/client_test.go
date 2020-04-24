package v1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

var (
	mockSystemSecret = &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "onepanel",
			Namespace: "onepanel",
		},
	}

	mockSystemConfigMap = &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "onepanel",
			Namespace: "onepanel",
		},
		Data: map[string]string{
			"ONEPANEL_HOST":            "demo.onepanel.site",
			"applicationNodePoolLabel": "beta.kubernetes.io/instance-type",
			"applicationNodePoolOptions": `
- name: 'CPU: 2, RAM: 8GB'
  value: 'Standard_D2s_v3'
  default: true
- name: 'CPU: 4, RAM: 16GB'
  value: 'Standard_D4s_v3'
- name: 'CPU: 8, RAM: 32GB'
  value: 'Standard_D5s_v3'
`,
		},
	}
)

func NewTestClient(objects ...runtime.Object) (client *Client) {
	return &Client{Interface: fake.NewSimpleClientset(objects...)}
}
