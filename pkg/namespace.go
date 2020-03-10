package v1

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var onepanelEnabledLabelKey = "onepanel.io/enabled"

func (c *Client) ListOnepanelEnabledNamespaces() (namespaces []*Namespace, err error) {
	namespaceList, err := c.CoreV1().Namespaces().List(metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", onepanelEnabledLabelKey, "true"),
	})
	if err != nil {
		return
	}

	for _, ns := range namespaceList.Items {
		namespaces = append(namespaces, &Namespace{
			Name:   ns.Name,
			Labels: ns.Labels,
		})
	}

	return
}

func (c *Client) ListNamespaces() (namespaces []*Namespace, err error) {
	namespaceList, err := c.CoreV1().Namespaces().List(metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", onepanelEnabledLabelKey, "true"),
	})
	if err != nil {
		return
	}

	for _, ns := range namespaceList.Items {
		namespaces = append(namespaces, &Namespace{
			Name:   ns.Name,
			Labels: ns.Labels,
		})
	}

	return
}
