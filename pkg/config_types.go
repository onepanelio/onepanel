package v1

import (
	"gopkg.in/yaml.v3"
	"encoding/base64"
	"fmt"
	"github.com/onepanelio/core/pkg/util/ptr"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"
	"strings"
)

// SystemConfig is configuration loaded from kubernetes config and secrets that includes information about the
// database, server, etc.
type SystemConfig map[string]string

// NodePoolOption extends ParameterOption to support resourceRequirements
type NodePoolOption struct {
	ParameterOption
	Resources corev1.ResourceRequirements
}

// NewSystemConfig creates a System config by getting the required data from a ConfigMap and Secret
func NewSystemConfig(configMap *ConfigMap, secret *Secret) (config SystemConfig, err error) {
	config = configMap.Data

	databaseUsername, err := base64.StdEncoding.DecodeString(secret.Data["databaseUsername"])
	if err != nil {
		return
	}
	config["databaseUsername"] = string(databaseUsername)

	databasePassword, err := base64.StdEncoding.DecodeString(secret.Data["databasePassword"])
	if err != nil {
		return
	}
	config["databasePassword"] = string(databasePassword)

	return
}

// GetValue returns the value in the underlying map if it exists, otherwise nil is returned
// If the value does not exist, it is also logged.
func (s SystemConfig) GetValue(name string) *string {
	value, ok := s[name]
	if !ok {
		log.WithFields(log.Fields{
			"Method": "SystemConfig.GetValue",
			"Name":   name,
			"Error":  "does not exist",
		})

		return nil
	}

	return &value
}

// Domain gets the ONEPANEL_DOMAIN value, or nil.
func (s SystemConfig) Domain() *string {
	return s.GetValue("ONEPANEL_DOMAIN")
}

// APIURL gets the ONEPANEL_API_URL, or nil.
func (s SystemConfig) APIURL() *string {
	return s.GetValue("ONEPANEL_API_URL")
}

// APIProtocol returns either http:// or https:// or nil.
// It is based on the ONEPANEL_API_URL config value and checks if it has https or http
func (s SystemConfig) APIProtocol() *string {
	url := s.APIURL()
	if url == nil {
		return nil
	}

	if strings.HasPrefix(*url, "https://") {
		return ptr.String("https://")
	}

	return ptr.String("http://")
}

// FQDN gets the ONEPANEL_FQDN value or nil.
func (s SystemConfig) FQDN() *string {
	return s.GetValue("ONEPANEL_FQDN")
}

// NodePoolLabel gets the applicationNodePoolLabel from the config or returns nil.
func (s SystemConfig) NodePoolLabel() (label *string) {
	return s.GetValue("applicationNodePoolLabel")
}

// NodePoolOptions loads and parses the applicationNodePoolOptions from the config.
// If there is no data, an error is returned.
func (s SystemConfig) NodePoolOptions() (options []*NodePoolOption, err error) {
	data := s.GetValue("applicationNodePoolOptions")
	if data == nil {
		return nil, fmt.Errorf("no nodePoolOptions in config")
	}

	if err = yaml.Unmarshal([]byte(*data), &options); err != nil {
		return
	}

	return
}

// NodePoolOptionByValue returns the nodePoolOption based on a given value
func (s SystemConfig) NodePoolOptionByValue(value string) (option *NodePoolOption, err error) {
	options, err := s.NodePoolOptions()
	if err != nil {
		return
	}
	for _, opt := range options {
		if opt.Value == value {
			option = opt
			return
		}
	}
	return
}

// DatabaseDriverName gets the databaseDriverName value, or nil.
func (s SystemConfig) DatabaseDriverName() *string {
	return s.GetValue("databaseDriverName")
}

// DatabaseConnection returns system config information to connect to a database
func (s SystemConfig) DatabaseConnection() (driverName, dataSourceName string) {
	dataSourceName = fmt.Sprintf("host=%v user=%v password=%v dbname=%v sslmode=disable",
		s["databaseHost"], s["databaseUsername"], s["databasePassword"], s["databaseName"])

	driverName = *s.DatabaseDriverName()

	return
}

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
	KeyFormat               string `yaml:"keyFormat"`
	Bucket                  string
	Endpoint                string
	Insecure                bool
	ServiceAccountKey       string                   `yaml:"serviceAccountKey"`
	ServiceAccountKeySecret ArtifactRepositorySecret `yaml:"serviceAccountKeySecret"`
	ServiceAccountJSON      string                   `yaml:"omitempty"`
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
				Key:  a.AccessKeySecret.Key,
			},
			SecretKeySecret: ArtifactRepositorySecret{
				Name: a.SecretKeySecret.Name,
				Key:  a.SecretKeySecret.Key,
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
