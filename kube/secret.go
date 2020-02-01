package kube

import (
	"github.com/onepanelio/core/model"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *Client) CreateSecret(namespace string, secret *model.Secret) (err error) {
	_, err = c.CoreV1().Secrets(namespace).Create(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: secret.Name,
		},
		StringData: secret.Data,
	})

	return
}

func (c *Client) GetSecret(namespace, name string) (secret *model.Secret, err error) {
	s, err := c.CoreV1().Secrets(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return
	}

	data := make(map[string]string)
	for key := range s.Data {
		data[key] = string(s.Data[key])
	}
	secret = &model.Secret{
		Name: name,
		Data: data,
	}

	return
}
