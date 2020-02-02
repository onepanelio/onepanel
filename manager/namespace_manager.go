package manager

import (
	"fmt"

	"github.com/onepanelio/core/model"
)

var onepanelEnabledLabelKey = labelKeyPrefix + "enabled"

func (r *ResourceManager) ListNamespaces() (namespaces []*model.Namespace, err error) {
	return r.kubeClient.ListNamespaces(model.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", onepanelEnabledLabelKey, "true"),
	})
}
