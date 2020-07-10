package v1

import (
	"strconv"
	"testing"

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
					"onepanel.io/enabled": "true",
				},
			},
		})
	}
}

func TestClient_ListNamespace(t *testing.T) {
	c := DefaultTestClient()

	testCreateNamespace(c)

	n, err := c.ListNamespaces()
	assert.Nil(t, err)
	assert.NotEmpty(t, n)
	assert.Equal(t, len(n), 5)
}
