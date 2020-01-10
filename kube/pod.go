package kube

import (
	"io"

	v1 "k8s.io/api/core/v1"
)

func (c *Client) GetPodLogs(namespace, podName, containerName string) (io.ReadCloser, error) {
	return c.CoreV1().Pods(namespace).GetLogs(podName, &v1.PodLogOptions{
		Container:  containerName,
		Follow:     true,
		Timestamps: true,
	}).Stream()
}
