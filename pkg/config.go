package v1

import (
	"encoding/base64"
	"github.com/onepanelio/core/pkg/util"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

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
	c.cache = make(map[string]interface{})
}

// GetSystemConfig loads various system configurations and bundles them into a map.
// The configuration is cached once it is loaded, and that cached value is used from here on out.
func (c *Client) GetSystemConfig() (config SystemConfig, err error) {
	if c.systemConfig != nil {
		return c.systemConfig, nil
	}

	namespace := "onepanel"
	name := "onepanel"

	configMap, err := c.getConfigMap(namespace, name)
	if err != nil {
		return
	}

	secret, err := c.GetSecret(namespace, name)
	if err != nil {
		return
	}

	config, err = NewSystemConfig(configMap, secret)

	c.systemConfig = config

	return
}

// GetDefaultConfig returns the default configuration of the system
func (c *Client) GetDefaultConfig() (config *ConfigMap, err error) {
	config, err = c.getConfigMap("onepanel", "onepanel")

	return
}

// GetNamespaceConfig returns the NamespaceConfig given a namespace
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
		ArtifactRepository: ArtifactRepositoryProvider{},
	}

	err = yaml.Unmarshal([]byte(configMap.Data["artifactRepository"]), &config.ArtifactRepository)
	if err != nil || (config.ArtifactRepository.S3 == nil && config.ArtifactRepository.GCS == nil) {
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

	if config.ArtifactRepository.S3 == nil {
		return nil, util.NewUserError(codes.NotFound, "Artifact repository config not found.")
	}

	accessKey, _ := base64.StdEncoding.DecodeString(secret.Data[config.ArtifactRepository.S3.AccessKeySecret.Key])
	config.ArtifactRepository.S3.AccessKey = string(accessKey)
	secretKey, _ := base64.StdEncoding.DecodeString(secret.Data[config.ArtifactRepository.S3.SecretKeySecret.Key])
	config.ArtifactRepository.S3.Secretkey = string(secretKey)

	return
}
