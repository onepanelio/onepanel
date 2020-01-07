package argo

import (
	wfclientset "github.com/argoproj/argo/pkg/client/clientset/versioned"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Client struct {
	*wfclientset.Clientset
}

func NewClient(configPath ...string) *Client {
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

	return &Client{Clientset: wfclientset.NewForConfigOrDie(config)}
}
