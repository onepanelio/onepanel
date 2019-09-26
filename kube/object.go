package kube

import (
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func CreateObject(objectTemplate string, data interface{}) (err error) {
	client, err := newDynamicClient("/Users/rush/.kube/config")
	if err != nil {
		return
	}

	obj, err := parseObjectTemplate(objectTemplate, data)
	if err != nil {
		return
	}

	res := client.Resource(schema.GroupVersionResource{
		Group:    "networking.istio.io",
		Version:  "v1alpha3",
		Resource: "virtualservices",
	}).Namespace("rushtehrani")

	_, err = res.Create(&unstructured.Unstructured{
		Object: obj,
	}, metav1.CreateOptions{})
	if err != nil && errors.IsAlreadyExists(err) {
		_, err = res.Update(&unstructured.Unstructured{
			Object: obj,
		}, metav1.UpdateOptions{})
	}

	return
}
