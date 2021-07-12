package v1

import (
	"fmt"
	v1 "k8s.io/api/core/v1"

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

// GetNamespace gets the namespace from the cluster if it exists
func (c *Client) GetNamespace(name string) (namespace *Namespace, err error) {
	ns, err := c.CoreV1().Namespaces().Get(name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	namespace = &Namespace{
		Name:   ns.Name,
		Labels: ns.Labels,
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

func (c *Client) CreateNamespace(name string) (namespace *Namespace, err error) {
	createNamespace := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"istio-injection":       "enabled",
				onepanelEnabledLabelKey: "true",
			},
		},
	}

	k8Namespace, err := c.CoreV1().Namespaces().Create(createNamespace)
	if err != nil {
		return
	}

	namespace = &Namespace{
		Name: k8Namespace.Name,
	}

	return
}
