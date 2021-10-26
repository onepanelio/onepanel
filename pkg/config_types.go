package v1

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/onepanelio/core/pkg/util/ptr"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	k8yaml "sigs.k8s.io/yaml"
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

	hmac, err := base64.StdEncoding.DecodeString(secret.Data["hmac"])
	if err != nil {
		return
	}
	config["hmac"] = string(hmac)

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

	if err = k8yaml.Unmarshal([]byte(*data), &options); err != nil {
		return
	}

	return
}

// NodePoolOptionsAsParameters returns the NodePool options as []*ParameterOption
func (s SystemConfig) NodePoolOptionsAsParameters() (result []*ParameterOption, err error) {
	nodePoolOptions, err := s.NodePoolOptions()
	if err != nil {
		return nil, err
	}

	result = make([]*ParameterOption, 0)
	for _, option := range nodePoolOptions {
		result = append(result, &ParameterOption{
			Name:  option.Name,
			Value: option.Value,
		})
	}

	return
}

// NodePoolOptionsMap returns a map where each key is a node pool value and the value is a NodePoolOption
func (s SystemConfig) NodePoolOptionsMap() (result map[string]*NodePoolOption, err error) {
	data := s.GetValue("applicationNodePoolOptions")
	if data == nil {
		return nil, fmt.Errorf("no nodePoolOptions in config")
	}

	options := make([]*NodePoolOption, 0)
	if err = k8yaml.Unmarshal([]byte(*data), &options); err != nil {
		return
	}

	result = make(map[string]*NodePoolOption)
	for i := range options {
		val := options[i]

		result[val.Value] = val
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

// Provider gets the ONEPANEL_PROVIDER value, or nil.
func (s SystemConfig) Provider() *string {
	return s.GetValue("ONEPANEL_PROVIDER")
}

// DatabaseConnection returns system config information to connect to a database
func (s SystemConfig) DatabaseConnection() (driverName, dataSourceName string) {
	dataSourceName = fmt.Sprintf("host=%v user=%v password=%v dbname=%v sslmode=disable",
		s["databaseHost"], s["databaseUsername"], s["databasePassword"], s["databaseName"])

	driverName = *s.DatabaseDriverName()

	return
}

// UpdateNodePoolOptions will update the sys-node-pool parameter's options with runtime values
// The original slice is unmodified, the returned slice has the updated values
// If sys-node-pool is not present, nothing happens.
func (s SystemConfig) UpdateNodePoolOptions(parameters []Parameter) ([]Parameter, error) {
	result := make([]Parameter, 0)

	var nodePoolParameter *Parameter

	// Copy the original parameters, skipping sys-node-pool
	for i := range parameters {
		parameter := parameters[i]
		if parameter.Name == "sys-node-pool" {
			nodePoolParameter = &parameter
			continue
		}

		result = append(result, parameter)
	}

	if nodePoolParameter == nil {
		return result, nil
	}

	nodePoolOptions, err := s.NodePoolOptions()
	if err != nil {
		return result, err
	}

	options := make([]*ParameterOption, 0)
	for _, option := range nodePoolOptions {
		newOption := &ParameterOption{
			Name:  option.Name,
			Value: option.Value,
		}

		options = append(options, newOption)
	}

	nodePoolParameter.Options = options

	result = append(result, *nodePoolParameter)

	return result, nil
}

// HMACKey gets the HMAC value, or nil.
func (s SystemConfig) HMACKey() []byte {
	hmac := s.GetValue("hmac")
	if hmac == nil {
		return []byte{}
	}

	return []byte(*hmac)
}

// ArtifactRepositoryS3Provider is meant to be used
// by the CLI. CLI will marshal this struct into the correct
// YAML structure for k8s configmap / secret.
type ArtifactRepositoryS3Provider struct {
	Source          string
	KeyFormat       string `yaml:"keyFormat"`
	Bucket          string
	Endpoint        string
	PublicEndpoint  string `yaml:"publicEndpoint"`
	PublicInsecure  bool   `yaml:"publicInsecure"`
	Insecure        bool
	Region          string
	AccessKeySecret ArtifactRepositorySecret `yaml:"accessKeySecret"`
	SecretKeySecret ArtifactRepositorySecret `yaml:"secretKeySecret"`
	AccessKey       string                   `yaml:"accessKey,omitempty"`
	Secretkey       string                   `yaml:"secretKey,omitempty"`
}

// ArtifactRepositoryGCSProvider is meant to be used
// by the CLI. CLI will marshal this struct into the correct
// YAML structure for k8s configmap / secret.
type ArtifactRepositoryGCSProvider struct {
	Source                  string
	KeyFormat               string `yaml:"keyFormat"`
	Bucket                  string
	Endpoint                string
	Insecure                bool
	ServiceAccountKey       string                   `yaml:"serviceAccountKey,omitempty"`
	ServiceAccountKeySecret ArtifactRepositorySecret `yaml:"serviceAccountKeySecret"`
	ServiceAccountJSON      string                   `yaml:"serviceAccountJSON,omitempty"`
}

// ArtifactRepositoryProvider is used to setup access into AWS Cloud Storage
// or Google Cloud storage.
// - The relevant sub-struct (S3, GCS) is unmarshalled into from the cluster configmap.
// Right now, either the S3 or GCS struct will be filled in. Multiple cloud
// providers are not supported at the same time in params.yaml (manifests deployment).
type ArtifactRepositoryProvider struct {
	S3  *ArtifactRepositoryS3Provider  `yaml:"s3,omitempty"`
	GCS *ArtifactRepositoryGCSProvider `yaml:"gcs,omitempty"`
}

// ArtifactRepositorySecret holds information about a kubernetes Secret.
// - The "key" is the specific key inside the Secret.
// - The "name" is the name of the Secret.
// Usually, this is used to figure out what secret to look into for a specific value.
type ArtifactRepositorySecret struct {
	Key  string `yaml:"key"`
	Name string `yaml:"name"`
}

// MarshalToYaml is used by the CLI to generate configmaps during deployment
// or build operations.
func (a *ArtifactRepositoryS3Provider) MarshalToYaml() (string, error) {
	builder := &strings.Builder{}
	encoder := yaml.NewEncoder(builder)
	encoder.SetIndent(6)
	defer encoder.Close()
	err := encoder.Encode(&ArtifactRepositoryProvider{
		S3: &ArtifactRepositoryS3Provider{
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
		return "", err
	}

	return builder.String(), nil
}

// MarshalToYaml is used by the CLI to generate configmaps during deployment
// or build operations.
func (g *ArtifactRepositoryGCSProvider) MarshalToYaml() (string, error) {
	builder := &strings.Builder{}
	encoder := yaml.NewEncoder(builder)
	encoder.SetIndent(6)
	defer encoder.Close()
	err := encoder.Encode(&ArtifactRepositoryProvider{
		GCS: &ArtifactRepositoryGCSProvider{
			KeyFormat: g.KeyFormat,
			Bucket:    g.Bucket,
			Endpoint:  g.Endpoint,
			Insecure:  g.Insecure,
			ServiceAccountKeySecret: ArtifactRepositorySecret{
				Key:  "artifactRepositoryGCSServiceAccountKey",
				Name: "onepanel",
			},
		},
	})

	if err != nil {
		return "", err
	}

	return builder.String(), nil
}

// FormatKey replaces placeholder values with their actual values and returns this string.
// {{workflow.namespace}} -> namespace
// {{workflow.name}} -> workflowName
// {{pod.name}} -> podName
func (a *ArtifactRepositoryS3Provider) FormatKey(namespace, workflowName, podName string) string {
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
func (g *ArtifactRepositoryGCSProvider) FormatKey(namespace, workflowName, podName string) string {
	keyFormat := g.KeyFormat

	keyFormat = strings.Replace(keyFormat, "{{workflow.namespace}}", namespace, -1)
	keyFormat = strings.Replace(keyFormat, "{{workflow.name}}", workflowName, -1)
	keyFormat = strings.Replace(keyFormat, "{{pod.name}}", podName, -1)

	return keyFormat
}

// NamespaceConfig represents configuration for the namespace
type NamespaceConfig struct {
	ArtifactRepository ArtifactRepositoryProvider
}
