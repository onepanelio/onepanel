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

func (r *ResourceManager) GetSecret(namespace, name string) (secret *model.Secret, err error) {
	return r.kubeClient.GetSecret(namespace, name)
}
