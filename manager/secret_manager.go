package manager

import (
	"github.com/onepanelio/core/model"
)

func (r *ResourceManager) CreateSecret(namespace string, secret *model.Secret) (err error) {
	return r.kubeClient.CreateSecret(namespace, secret)
}

func (r *ResourceManager) SecretExists(namespace string, secretName string) (exists bool, err error) {
	return r.kubeClient.SecretExists(namespace, secretName)
}

func (r *ResourceManager) GetSecret(namespace string, secret *model.Secret) (secretRes *model.Secret, err error) {
	return r.kubeClient.GetSecret(namespace, secret.Name)
}

func (r *ResourceManager) ListSecrets(namespace string) (secrets []model.Secret, err error) {
	return r.kubeClient.ListSecrets(namespace)
}

func (r *ResourceManager) DeleteSecret(namespace string, secretName string) (deleted bool, err error) {
	return r.kubeClient.DeleteSecret(namespace, secretName)
}

func (r *ResourceManager) DeleteSecretKey(namespace string, secretName string, key string) (deleted bool, err error) {
	return r.kubeClient.DeleteSecretKey(namespace, secretName, key)
}

func (r *ResourceManager) AddSecretKeyValue(namespace string, secretName string, key string, value string) (inserted bool, err error) {
	return r.kubeClient.AddSecretKeyValue(namespace, secretName, key, value)
}

func (r *ResourceManager) UpdateSecretKeyValue(namespace string, secretName string, key string, value string) (updated bool, err error) {
	return r.kubeClient.UpdateSecretKeyValue(namespace, secretName, key, value)
}
