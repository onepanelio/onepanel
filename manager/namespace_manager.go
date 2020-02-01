package manager

import (
	"fmt"

	"github.com/onepanelio/core/model"
)

var defaultNamespaceLabelKey = labelKeyPrefix + "is-default-namespace"

func (r *ResourceManager) GetDefaultNamespace() (namespaces []*model.Namespace, err error) {
	return r.kubeClient.ListNamespaces(model.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", defaultNamespaceLabelKey, "true"),
	})
}
