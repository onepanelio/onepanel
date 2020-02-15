package kube

import (
	argoprojv1alpha1 "github.com/argoproj/argo/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Config = rest.Config

type Client struct {
	kubernetes.Interface
	argoprojV1alpha1 argoprojv1alpha1.ArgoprojV1alpha1Interface
}

func (c *Client) ArgoprojV1alpha1() argoprojv1alpha1.ArgoprojV1alpha1Interface {
	return c.argoprojV1alpha1
}

func NewConfig() (config *Config) {
	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{}).ClientConfig()
	if err != nil {
		panic(err)
	}

	return
}

func NewClient(config *Config, token string) (client *Client, err error) {
	config.BearerToken = ""
	config.BearerTokenFile = ""
	config.Username = ""
	config.Password = ""
	config.CertData = nil
	config.CertFile = ""
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
