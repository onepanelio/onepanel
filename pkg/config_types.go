package v1

import (
	"gopkg.in/yaml.v3"
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

// ArtifactRepositoryS3Provider is meant to be used
// by the CLI. CLI will marshal this struct into the correct
// YAML structure for k8s configmap / secret.
type ArtifactRepositoryS3Provider struct {
	KeyFormat       string `yaml:"keyFormat"`
	Bucket          string
	Endpoint        string
	Insecure        bool
	Region          string
	AccessKeySecret ArtifactRepositorySecret `yaml:"accessKeySecret"`
	SecretKeySecret ArtifactRepositorySecret `yaml:"secretKeySecret"`
}

type ArtifactRepositoryProviderConfig struct {
	S3 ArtifactRepositoryS3Provider `yaml:"s3"`
}

type ArtifactRepositorySecret struct {
	Key  string `yaml:"key"`
	Name string `yaml:"name"`
}

func (a *ArtifactRepositoryS3Config) MarshalToYaml() (error, string) {
	builder := &strings.Builder{}
	encoder := yaml.NewEncoder(builder)
	encoder.SetIndent(6)
	defer encoder.Close()
	err := encoder.Encode(&ArtifactRepositoryProviderConfig{
		S3: ArtifactRepositoryS3Provider{
			KeyFormat: a.KeyFormat,
			Bucket:    a.Bucket,
			Endpoint:  a.Endpoint,
			Insecure:  a.Insecure,
			Region:    a.Region,
			AccessKeySecret: ArtifactRepositorySecret{
				Name: a.AccessKeySecret.Name,
				Key:  a.AccessKey,
			},
			SecretKeySecret: ArtifactRepositorySecret{
				Name: a.SecretKeySecret.Name,
				Key:  a.Secretkey,
			},
		},
	})

	if err != nil {
		return err, ""
	}

	return nil, builder.String()
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
