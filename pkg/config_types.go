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

type ArtifactRepositoryGCSConfig struct {
	KeyFormat         string `yaml:"keyFormat"`
	Bucket            string
	Endpoint          string
	Insecure          bool
	ServiceAccountKey string `yaml:"serviceAccountKey"`
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

// ArtifactRepositoryGCSProvider is meant to be used
// by the CLI. CLI will marshal this struct into the correct
// YAML structure for k8s configmap / secret.
type ArtifactRepositoryGCSProvider struct {
	KeyFormat               string `yaml:"keyFormat"`
	Bucket                  string
	Endpoint                string
	Insecure                bool
	ServiceAccountKey       string                   `yaml:"serviceAccountKey,omitempty"`
	ServiceAccountKeySecret ArtifactRepositorySecret `yaml:"serviceAccountKeySecret"`
}

type ArtifactRepositoryProviderConfig struct {
	S3  ArtifactRepositoryS3Provider  `yaml:"s3,omitempty"`
	GCS ArtifactRepositoryGCSProvider `yaml:"gcs,omitempty"`
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

func (g *ArtifactRepositoryGCSConfig) MarshalToYaml() (error, string) {
	builder := &strings.Builder{}
	encoder := yaml.NewEncoder(builder)
	encoder.SetIndent(6)
	defer encoder.Close()
	err := encoder.Encode(&ArtifactRepositoryProviderConfig{
		GCS: ArtifactRepositoryGCSProvider{
			KeyFormat: g.KeyFormat,
			Bucket:    g.Bucket,
			Endpoint:  g.Endpoint,
			Insecure:  g.Insecure,
			ServiceAccountKeySecret: ArtifactRepositorySecret{
				Key:  "serviceAccountKey",
				Name: "onepanel",
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

// FormatKey replaces placeholder values with their actual values and returns this string.
// {{workflow.namespace}} -> namespace
// {{workflow.name}} -> workflowName
// {{pod.name}} -> podName
func (g *ArtifactRepositoryGCSConfig) FormatKey(namespace, workflowName, podName string) string {
	keyFormat := g.KeyFormat

	keyFormat = strings.Replace(keyFormat, "{{workflow.namespace}}", namespace, -1)
	keyFormat = strings.Replace(keyFormat, "{{workflow.name}}", workflowName, -1)
	keyFormat = strings.Replace(keyFormat, "{{pod.name}}", podName, -1)

	return keyFormat
}

type ArtifactRepositoryConfig struct {
	S3  *ArtifactRepositoryS3Config
	GCS *ArtifactRepositoryGCSConfig
}

type NamespaceConfig struct {
	ArtifactRepository ArtifactRepositoryConfig
}
