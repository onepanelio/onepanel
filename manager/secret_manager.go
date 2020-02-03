package manager

import (
	"github.com/onepanelio/core/model"
	apiv1 "k8s.io/api/core/v1"
)

func (r *ResourceManager) CreateSecret(namespace string, secret *model.Secret) (err error) {
	return r.kubeClient.CreateSecret(namespace, secret)
}

func (r *ResourceManager) SecretExists(namespace string, secretName string) (exists bool, err error) {
	return r.kubeClient.SecretExists(namespace, secretName)
}

func (r *ResourceManager) GetSecret(namespace, name string) (secret *model.Secret, err error) {
	return r.kubeClient.GetSecret(namespace, name)
}

func (r *ResourceManager) GetSecrets(namespace string) (secrets []apiv1.Secret, err error) {
	return r.kubeClient.GetSecrets(namespace)
}

func (r *ResourceManager) DeleteSecret(namespace string, secretName string) (deleted bool, err error) {
	return r.kubeClient.DeleteSecret(namespace, secretName)
}

func (r *ResourceManager) DeleteSecretKey(namespace string, secretName string, key string) (deleted bool, err error) {
	return r.kubeClient.DeleteSecretKey(namespace, secretName, key)
}
