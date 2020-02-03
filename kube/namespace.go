package kube

import (
	"github.com/onepanelio/core/model"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *Client) ListNamespaces(opts model.ListOptions) (namespaces []*model.Namespace, err error) {
	namespaceList, err := c.CoreV1().Namespaces().List(metav1.ListOptions{
		LabelSelector: opts.LabelSelector,
		FieldSelector: opts.FieldSelector,
	})
	if err != nil {
		return
	}

	for _, ns := range namespaceList.Items {
		namespaces = append(namespaces, &model.Namespace{
			Name:   ns.Name,
			Labels: ns.Labels,
		})
	}

	return
}
