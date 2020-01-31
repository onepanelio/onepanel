package kube

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ConfigMap struct {
	Name string
	Data map[string]string
}

func (c *Client) CreateConfigMap(namespace string, configMap *ConfigMap) (err error) {
	_, err = c.CoreV1().ConfigMaps(namespace).Create(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: configMap.Name,
		},
		Data: configMap.Data,
	})
	if err != nil {
		return
	}

	return
}

func (c *Client) GetConfigMap(namespace, name string) (configMap *ConfigMap, err error) {
	cm, err := c.CoreV1().ConfigMaps(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return
	}
	configMap = &ConfigMap{
		Name: name,
		Data: cm.Data,
	}

	return
}
