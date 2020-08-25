package v1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClient_CreateSecret(t *testing.T) {
	c := DefaultTestClient()

	err := c.CreateSecret("namespace", &Secret{
		Name: "name",
	})
	assert.Nil(t, err)
}

func TestClient_GetSecret(t *testing.T) {
	c := DefaultTestClient()

	err := c.CreateSecret("namespace", &Secret{
		Name: "name",
	})
	assert.Nil(t, err)

	s, err := c.GetSecret("namespace", "name")
	assert.Nil(t, err)

	assert.NotNil(t, s)
	assert.Equal(t, s.Name, "name")
}
