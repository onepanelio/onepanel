package kube

import (
	"testing"

	"github.com/onepanelio/core/model"
	"github.com/stretchr/testify/assert"
)

func TestCreateConfigMap(t *testing.T) {
	c := NewTestClient()

	err := c.CreateConfigMap("namespace", &model.ConfigMap{
		Name: "name",
	})
	assert.Nil(t, err)
}

func TestGetConfigMap(t *testing.T) {
	c := NewTestClient()

	err := c.CreateConfigMap("namespace", &model.ConfigMap{
		Name: "name",
	})
	assert.Nil(t, err)

	s, err := c.GetConfigMap("namespace", "name")
	assert.Nil(t, err)

	assert.NotNil(t, s)
	assert.Equal(t, s.Name, "name")
}
