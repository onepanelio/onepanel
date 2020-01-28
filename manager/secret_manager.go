package manager

import (
	"github.com/onepanelio/core/model"
	apiv1 "k8s.io/api/core/v1"
)

func (r *ResourceManager) CreateSecret(namespace string, secret *model.Secret) (err error) {
	return r.kubeClient.CreateSecret(namespace, secret)
}

func (r *ResourceManager) GetSecret(namespace, name string) (secret *apiv1.Secret, err error) {
	secret, err = r.kubeClient.GetSecret(namespace, name)

	return
}
