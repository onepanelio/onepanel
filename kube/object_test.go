package kube

import (
	"testing"

	"github.com/onepanelio/core/template"
)

func TestCreateVirtualService(t *testing.T) {
	c, err := NewDynamicClient("/Users/rush/.kube/config")
	if err != nil {
		t.Error(err)
		return
	}

	err = c.CreateObject(template.VirtualService, struct {
		SystemIstioGateway string
		InstanceName       string
	}{
		"istio-system/istio-ingress-gateway",
		"examples-test-2",
	})
	if err != nil {
		t.Error(err)
		return
	}

	t.Log("success")
}
