package manager

import (
	"github.com/onepanelio/core/model"
)

func (r *ResourceManager) ListNamespaces(opts model.ListOptions) (namespaces []*model.Namespace, err error) {
	return r.kubeClient.ListNamespaces(opts)
}
