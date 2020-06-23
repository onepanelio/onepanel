package v1

import (
	"encoding/base64"
	"fmt"
	"github.com/onepanelio/core/pkg/util"
	"github.com/onepanelio/core/pkg/util/ptr"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"regexp"
	"sigs.k8s.io/yaml"
	"strings"
)

// SystemConfig is configuration loaded from kubernetes config and secrets that includes information about the
// database, server, etc.
type SystemConfig map[string]string

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
func (s SystemConfig) NodePoolOptions() (options []*ParameterOption, err error) {
	data := s.GetValue("applicationNodePoolOptions")
	if data == nil {
		return nil, fmt.Errorf("no nodePoolOptions in config")
	}

	if err = yaml.Unmarshal([]byte(*data), &options); err != nil {
		return
	}

	return
}

// DatabaseDriverName gets the databaseDriverName value, or nil.
func (s SystemConfig) DatabaseDriverName() *string {
	return s.GetValue("databaseDriverName")
}

// IsSubdomainValid checks if the subdomain is valid.
// Nil is returned if it is, otherwise an error
// message describing the violation is returned.
func (s SystemConfig) IsSubdomainValid(namespace, subdomain string) error {
	totalLength := len(subdomain) + len(namespace) + 2
	maxSubdomainLength := 63 - 2 - len(namespace)

	if totalLength > 63 {
		return fmt.Errorf("must be less than %v characters", maxSubdomainLength)
	}

	re, _ := regexp.Compile(`^[A-Za-z][A-Za-z0-9_-]*$`)
	if re.Match([]byte(subdomain)) {
		return nil
	}

	return fmt.Errorf("must start with a letter")
}

func (c *Client) getConfigMap(namespace, name string) (configMap *ConfigMap, err error) {
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

// ClearSystemConfigCache wipes out the cached system configuration so that the next call to
// GetSystemConfig will pull it from the resources
func (c *Client) ClearSystemConfigCache() {
	c.systemConfig = nil
}

// GetSystemConfig loads various system configurations and bundles them into a map.
// The configuration is cached once it is loaded, and that cached value is used from here on out.
func (c *Client) GetSystemConfig() (config SystemConfig, err error) {
	if c.systemConfig != nil {
		return c.systemConfig, nil
	}

	namespace := "onepanel"
	configMap, err := c.getConfigMap(namespace, "onepanel")
	if err != nil {
		return
	}
	config = configMap.Data

	secret, err := c.GetSecret(namespace, "onepanel")
	if err != nil {
		return
	}
	databaseUsername, _ := base64.StdEncoding.DecodeString(secret.Data["databaseUsername"])
	config["databaseUsername"] = string(databaseUsername)
	databasePassword, _ := base64.StdEncoding.DecodeString(secret.Data["databasePassword"])
	config["databasePassword"] = string(databasePassword)

	c.systemConfig = config

	return
}

func (c *Client) GetNamespaceConfig(namespace string) (config *NamespaceConfig, err error) {
	configMap, err := c.getConfigMap(namespace, "onepanel")
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace": namespace,
			"Error":     err.Error(),
		}).Error("getNamespaceConfig failed getting config map.")
		return
	}
	config = &NamespaceConfig{
		ArtifactRepository: ArtifactRepositoryConfig{},
	}

	err = yaml.Unmarshal([]byte(configMap.Data["artifactRepository"]), &config.ArtifactRepository)
	if err != nil || config.ArtifactRepository.S3 == nil {
		return nil, util.NewUserError(codes.NotFound, "Artifact repository config not found.")
	}

	secret, err := c.GetSecret(namespace, "onepanel")
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace": namespace,
			"Error":     err.Error(),
		}).Error("getNamespaceConfig failed getting secret.")
		return
	}

	// TODO: replace with switch statement to support additional object storage
	if config.ArtifactRepository.S3 == nil {
		return nil, util.NewUserError(codes.NotFound, "Artifact repository config not found.")
	}
	accessKey, _ := base64.StdEncoding.DecodeString(secret.Data[config.ArtifactRepository.S3.AccessKeySecret.Key])
	config.ArtifactRepository.S3.AccessKey = string(accessKey)
	secretKey, _ := base64.StdEncoding.DecodeString(secret.Data[config.ArtifactRepository.S3.SecretKeySecret.Key])
	config.ArtifactRepository.S3.Secretkey = string(secretKey)

	return
}
