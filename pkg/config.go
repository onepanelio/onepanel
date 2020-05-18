package v1

import (
	"encoding/base64"
	"github.com/ghodss/yaml"
	"github.com/onepanelio/core/pkg/util"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func (c *Client) GetSystemConfig() (config map[string]string, err error) {
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

	err = yaml.Unmarshal([]byte(configMap.Data["config"]), &config)
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
