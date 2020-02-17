package kube

import (
	"encoding/base64"
	"github.com/onepanelio/core/model"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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

func (c *Client) SecretExists(namespace, name string) (*model.Secret, error) {
	secretK8s, err := c.CoreV1().Secrets(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return &model.Secret{}, err
	}
	retSecret := model.Secret{
		Name: secretK8s.Name,
	}
	return &retSecret, nil
}

func (c *Client) GetSecret(namespace, name string) (*model.Secret, error) {
	foundSecret, err := c.CoreV1().Secrets(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	if foundSecret != nil {
		return &model.Secret{
			Name: foundSecret.Name,
			Data: convertSecretToMap(foundSecret),
		}, nil
	}
	return nil, nil
}

func (c *Client) ListSecrets(namespace string) (secrets []*model.Secret, err error) {
	secretsList, err := c.CoreV1().Secrets(namespace).List(metav1.ListOptions{})
	if err != nil {
		return
	}

	for _, secret := range secretsList.Items {
		secretModel := model.Secret{
			Name: secret.Name,
			Data: convertSecretToMap(&secret),
		}
		secrets = append(secrets, &secretModel)
	}

	return
}

func (c *Client) DeleteSecret(namespace, name string) (err error) {
	return c.CoreV1().Secrets(namespace).Delete(name, &metav1.DeleteOptions{})
}

func (c *Client) DeleteSecretKey(namespace, name string, payload []byte) (err error) {
	_, err = c.CoreV1().Secrets(namespace).Patch(name, types.JSONPatchType, payload)
	return
}

func (c *Client) AddSecretKeyValue(namespace, name string, payload []byte) (err error) {
	_, err = c.CoreV1().Secrets(namespace).Patch(name, types.JSONPatchType, payload)
	return
}

func (c *Client) UpdateSecretKeyValue(namespace string, name string, payload []byte) (err error) {
	_, err = c.CoreV1().Secrets(namespace).Patch(name, types.JSONPatchType, payload)
	return
}

func convertSecretToMap(foundSecret *corev1.Secret) (modelData map[string]string) {
	modelData = make(map[string]string)
	for secretKey, secretData := range foundSecret.Data {
		modelData[secretKey] = base64.StdEncoding.EncodeToString(secretData)
	}
	return modelData
}
