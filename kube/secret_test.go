package kube

import "testing"

import "github.com/stretchr/testify/assert"

func TestCreateSecret(t *testing.T) {
	c := NewTestClient()

	err := c.CreateSecret("namespace", &Secret{
		Name: "name",
	})
	assert.Nil(t, err)
}

func TestGetSecret(t *testing.T) {
	c := NewTestClient()

	err := c.CreateSecret("namespace", &Secret{
		Name: "name",
	})
	assert.Nil(t, err)

	s, err := c.GetSecret("namespace", "name")
	assert.Nil(t, err)

	assert.NotNil(t, s)
	assert.Equal(t, s.Name, "name")
}
