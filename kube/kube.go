package kube

import (
	"bytes"
	"html/template"

	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func newDynamicClient(configPath ...string) (client dynamic.Interface, err error) {
	var config *rest.Config

	if len(configPath) == 0 {
		config, err = rest.InClusterConfig()
	} else {
		config, err = clientcmd.BuildConfigFromFlags("", configPath[0])
	}

	if err != nil {
		return nil, err
	}

	client, err = dynamic.NewForConfig(config)

	return
}

func parseObjectTemplate(objectTemplate string, data interface{}) (obj map[string]interface{}, err error) {
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
