package kube

import (
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Secret struct {
	Name string
	Data map[string]string
}

func (c *Client) CreateSecret(namespace string, secret *Secret) (err error) {
	_, err = c.CoreV1().Secrets(namespace).Create(&apiv1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: secret.Name,
		},
		StringData: secret.Data,
	})

	return
}
