package kube

import (
	argoprojv1alpha1 "github.com/argoproj/argo/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Client struct {
	kubernetes.Interface
	argoprojV1alpha1 argoprojv1alpha1.ArgoprojV1alpha1Interface
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

	return &Client{Interface: kubernetes.NewForConfigOrDie(config), argoprojV1alpha1: argoprojv1alpha1.NewForConfigOrDie(config)}
}

func GetClient(token string) (client *Client, err error) {
	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{}).ClientConfig()
	if err != nil {
		return
	}
	config.BearerToken = ""
	config.BearerTokenFile = ""
	config.Username = ""
	config.Password = ""
	if token != "" {
		config.BearerToken = token
	}

	kubeConfig, err := kubernetes.NewForConfig(config)
	if err != nil {
		return
	}

	argoConfig, err := argoprojv1alpha1.NewForConfig(config)
	if err != nil {
		return
	}

	return &Client{Interface: kubeConfig, argoprojV1alpha1: argoConfig}, nil
}
