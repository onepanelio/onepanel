package argo

import (
	wfclientset "github.com/argoproj/argo/pkg/client/clientset/versioned"
	"github.com/argoproj/argo/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Client struct {
	v1alpha1.WorkflowInterface
}

func NewClient(namespace string, configPath ...string) (client *Client, err error) {
	var config *rest.Config
	if len(configPath) == 0 {
		config, err = rest.InClusterConfig()
	} else {
		config, err = clientcmd.BuildConfigFromFlags("", configPath[0])
	}
	if err != nil {
		return
	}

	wfclient := wfclientset.NewForConfigOrDie(config).ArgoprojV1alpha1().Workflows(namespace)
	client = &Client{WorkflowInterface: wfclient}

	return
}
