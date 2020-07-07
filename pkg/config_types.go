package v1

import (
	corev1 "k8s.io/api/core/v1"
	"strings"
)

type ArtifactRepositoryS3Config struct {
	KeyFormat       string `yaml:"keyFormat"`
	Bucket          string
	Endpoint        string
	Insecure        bool
	Region          string
	AccessKeySecret corev1.SecretKeySelector
	SecretKeySecret corev1.SecretKeySelector
	AccessKey       string `yaml:"accessKey"`
	Secretkey       string `yaml:"secretKey"`
}
}

// FormatKey replaces placeholder values with their actual values and returns this string.
// {{workflow.namespace}} -> namespace
// {{workflow.name}} -> workflowName
// {{pod.name}} -> podName
func (a *ArtifactRepositoryS3Config) FormatKey(namespace, workflowName, podName string) string {
	keyFormat := a.KeyFormat

	keyFormat = strings.Replace(keyFormat, "{{workflow.namespace}}", namespace, -1)
	keyFormat = strings.Replace(keyFormat, "{{workflow.name}}", workflowName, -1)
	keyFormat = strings.Replace(keyFormat, "{{pod.name}}", podName, -1)

	return keyFormat
}

type ArtifactRepositoryConfig struct {
	S3 *ArtifactRepositoryS3Config
}

type NamespaceConfig struct {
	ArtifactRepository ArtifactRepositoryConfig
}
