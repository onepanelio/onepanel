package mocks

import (
	v1alpha1 "github.com/argoproj/argo/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	"github.com/argoproj/argo/pkg/client/clientset/versioned/typed/workflow/v1alpha1/fake"
	rest "k8s.io/client-go/rest"
	"k8s.io/client-go/testing"
)

type ArgoMock struct {
	argo *fake.FakeArgoprojV1alpha1
}

func NewArgo(k8fake *testing.Fake) ArgoMock {
	return ArgoMock{
		argo: &fake.FakeArgoprojV1alpha1{
			Fake: k8fake,
		},
	}
}

func (c ArgoMock) CronWorkflows(namespace string) v1alpha1.CronWorkflowInterface {
	return c.argo.CronWorkflows(namespace)
}

func (c ArgoMock) Workflows(namespace string) v1alpha1.WorkflowInterface {
	return c.argo.Workflows(namespace)
}

func (c ArgoMock) WorkflowTemplates(namespace string) v1alpha1.WorkflowTemplateInterface {
	return c.argo.WorkflowTemplates(namespace)
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c ArgoMock) RESTClient() rest.Interface {
	return c.argo.RESTClient()
}
