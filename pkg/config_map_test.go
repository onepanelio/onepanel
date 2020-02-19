package v1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateConfigMap(t *testing.T) {
	c := NewTestClient()

	err := c.CreateConfigMap("namespace", &ConfigMap{
		Name: "name",
	})
	assert.Nil(t, err)
}

func TestGetConfigMap(t *testing.T) {
	c := NewTestClient()

	err := c.CreateConfigMap("namespace", &ConfigMap{
		Name: "name",
	})
	assert.Nil(t, err)

	s, err := c.GetConfigMap("namespace", "name")
	assert.Nil(t, err)

	assert.NotNil(t, s)
	assert.Equal(t, s.Name, "name")
}
