package kube

import (
	"strconv"
	"testing"

	"github.com/onepanelio/core/model"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func testCreateNamespace(c *Client) {
	for i := 0; i < 5; i++ {
		c.CoreV1().Namespaces().Create(&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "namespace-" + strconv.Itoa(i),
				Labels: map[string]string{
					"label": "label-" + strconv.Itoa(i),
				},
			},
		})
	}
}
func TestListNamespace(t *testing.T) {
	c := NewTestClient()

	testCreateNamespace(c)

	n, err := c.ListNamespaces(model.ListOptions{})
	assert.Nil(t, err)
	assert.NotEmpty(t, n)
	assert.Equal(t, len(n), 5)
}

func TestListNamespaceByLabel(t *testing.T) {
	c := NewTestClient()

	testCreateNamespace(c)

	n, err := c.ListNamespaces(model.ListOptions{LabelSelector: "label=label-0"})
	assert.Nil(t, err)
	assert.NotEmpty(t, n)
	assert.Equal(t, n[0].Name, "namespace-0")
}
