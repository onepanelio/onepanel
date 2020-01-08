package kube

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Client struct {
	*kubernetes.Clientset
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

	return &Client{Clientset: kubernetes.NewForConfigOrDie(config)}
}
