package v1

import (
	"fmt"
	"github.com/onepanelio/core/pkg/util"
	"google.golang.org/grpc/codes"
	"io/ioutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
)

var onepanelEnabledLabelKey = "onepanel.io/enabled"

func replaceVariables(filepath string, replacements map[string]string) (string, error) {
	data, err := ioutil.ReadFile(filepath)
	if err != nil {
		return "", err
	}

	dataStr := string(data)
	for key, value := range replacements {
		dataStr = strings.ReplaceAll(dataStr, key, value)
	}

	return dataStr, nil
}

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

// ListNamespaces lists all of the onepanel enabled namespaces
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

// CreateNamespace creates a new namespace in the system
func (c *Client) CreateNamespace(sourceNamespace, name string) (namespace *Namespace, err error) {
	return nil, util.NewUserError(codes.FailedPrecondition, "Creating namespaces is not supported in the community edition")
}
