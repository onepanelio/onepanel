package kube

import (
	"github.com/onepanelio/core/model"
	corev1 "k8s.io/api/core/v1"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
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

func (c *Client) SecretExists(namespace string, secretName string) (exists bool, err error) {
	var foundSecret *apiv1.Secret
	foundSecret, err = c.CoreV1().Secrets(namespace).Get(secretName, metav1.GetOptions{})
	if err != nil {
		if err, ok := err.(*errors.StatusError); ok {
			if err.ErrStatus.Reason == "NotFound" {
				return false, nil
			}
			return false, err
		}
		return false, err
	}
	if foundSecret != nil {
		return true, nil
	}
	return false, nil
}

func (c *Client) GetSecret(namespace string, secretName string) (secret *apiv1.Secret, err error) {
	var foundSecret *apiv1.Secret
	foundSecret, err = c.CoreV1().Secrets(namespace).Get(secretName, metav1.GetOptions{})
	if err != nil {
		if err, ok := err.(*errors.StatusError); ok {
			if err.ErrStatus.Reason == "NotFound" {
				return nil, nil
			}
			return nil, err
		}
		return nil, err
	}
	if foundSecret != nil {
		return foundSecret, nil
	}
	return nil, nil
}

func (c *Client) GetSecrets(namespace string) (secrets []apiv1.Secret, err error) {
	listedSecrets, err := c.CoreV1().Secrets(namespace).List(metav1.ListOptions{})
	if err != nil {
		return
	}
	for _, secret := range listedSecrets.Items {
		secrets = append(secrets, secret)
	}
	return
}

func (c *Client) DeleteSecret(namespace string, secretName string) (deleted bool, err error) {
	err = c.CoreV1().Secrets(namespace).Delete(secretName, &metav1.DeleteOptions{})
	if err != nil {
		return false, err
	}
	return true, nil
}

func (c *Client) DeleteSecretKey(namespace string, secretName string, key string) (deleted bool, err error) {
	//Check if the secret has the key to delete
	secretFound, secretFindErr := c.GetSecret(namespace, secretName)
	if secretFindErr != nil {
		return false, secretFindErr
	}
	secretDataKeyExists := false
	for secretDataKey := range secretFound.Data {
		if secretDataKey == key {
			secretDataKeyExists = true
			break
		}
	}

	if secretDataKeyExists {
		//  patchStringPath specifies a patch operation for a secret key.
		type patchStringPath struct {
			Op   string `json:"op"`
			Path string `json:"path"`
		}
		payload := []patchStringPath{{
			Op:   "remove",
			Path: "/data/" + key,
		}}
		payloadBytes, _ := json.Marshal(payload)
		_, errSecret := c.CoreV1().Secrets(namespace).Patch(secretName, types.JSONPatchType, payloadBytes)
		if errSecret != nil {
			return false, errSecret
		}
		return true, nil
	}
	return true, nil
}

func (c *Client) UpdateSecretKeyValue() {

}
