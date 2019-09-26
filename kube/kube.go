package kube

import (
	"bytes"
	"html/template"

	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Client struct {
	dynamic.Interface
}

func NewClient(configPath ...string) (client *Client, err error) {
	var config *rest.Config
	if len(configPath) == 0 {
		config, err = rest.InClusterConfig()
	} else {
		config, err = clientcmd.BuildConfigFromFlags("", configPath[0])
	}
	if err != nil {
		return
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return
	}
	client = &Client{Interface: dynamicClient}

	return
}

func ParseObjectTemplate(objectTemplate string, data interface{}) (obj map[string]interface{}, err error) {
	var parsedObjectTemplate bytes.Buffer

	t, err := template.New("yaml").Parse(objectTemplate)
	if err != nil {
		return
	}

	if err = t.Execute(&parsedObjectTemplate, data); err != nil {
		return
	}

	obj = make(map[string]interface{})
	err = yaml.NewYAMLOrJSONDecoder(bytes.NewReader(parsedObjectTemplate.Bytes()), 4096).Decode(&obj)

	return
}
