package kube

import (
	argoprojv1alpha1 "github.com/argoproj/argo/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type ListOptions = v1.ListOptions

type Client struct {
	*kubernetes.Clientset
	argoprojV1alpha1 *argoprojv1alpha1.ArgoprojV1alpha1Client
}

func (c *Client) ArgoprojV1alpha1() argoprojv1alpha1.ArgoprojV1alpha1Interface {
	return c.argoprojV1alpha1
}

func NewClient(configPath ...string) (client *Client) {
	var (
		err    error
		config *rest.Config
	)

	if len(configPath) == 0 {
		config, err = rest.InClusterConfig()
	} else {
		config, err = clientcmd.BuildConfigFromFlags("", configPath[0])
	}
	if err != nil {
		panic(err)
	}

	return &Client{Clientset: kubernetes.NewForConfigOrDie(config), argoprojV1alpha1: argoprojv1alpha1.NewForConfigOrDie(config)}
}
