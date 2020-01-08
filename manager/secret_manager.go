package manager

import (
	"github.com/onepanelio/core/model"
)

func (r *ResourceManager) CreateSecret(namespace string, secret *model.Secret) (err error) {
	return r.kubeClient.CreateSecret(namespace, secret)
}
