package v1

import (
	"encoding/base64"
	"strconv"

	argoprojv1alpha1 "github.com/argoproj/argo/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	"github.com/jmoiron/sqlx"
	"github.com/onepanelio/core/pkg/util/s3"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	artifactRepositoryEndpointKey       = "artifactRepositoryEndpoint"
	artifactRepositoryBucketKey         = "artifactRepositoryBucket"
	artifactRepositoryRegionKey         = "artifactRepositoryRegion"
	artifactRepositoryInSecureKey       = "artifactRepositoryInsecure"
	artifactRepositoryAccessKeyValueKey = "artifactRepositoryAccessKey"
	artifactRepositorySecretKeyValueKey = "artifactRepositorySecretKey"
)

type Config = rest.Config

type DB = sqlx.DB

type Client struct {
	kubernetes.Interface
	argoprojV1alpha1 argoprojv1alpha1.ArgoprojV1alpha1Interface
	*DB
}

func (c *Client) ArgoprojV1alpha1() argoprojv1alpha1.ArgoprojV1alpha1Interface {
	return c.argoprojV1alpha1
}

func NewConfig() (config *Config) {
	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{}).ClientConfig()
	if err != nil {
		panic(err)
	}

	return
}

func NewClient(config *Config, db *sqlx.DB) (client *Client, err error) {
	if config.BearerToken != "" {
		config.BearerTokenFile = ""
		config.Username = ""
		config.Password = ""
		config.CertData = nil
		config.CertFile = ""
	}

	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return
	}

	argoClient, err := argoprojv1alpha1.NewForConfig(config)
	if err != nil {
		return
	}

	return &Client{Interface: kubeClient, argoprojV1alpha1: argoClient, DB: db}, nil
}

func (c *Client) GetSystemConfig() (config map[string]string, err error) {
	namespace := "onepanel"
	configMap, err := c.GetConfigMap(namespace, "onepanel")
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
	databasePassword, _ := base64.StdEncoding.DecodeString(secret.Data["databaseUsername"])
	config["databasePassword"] = string(databasePassword)

	return
}

func (c *Client) GetNamespaceConfig(namespace string) (config map[string]string, err error) {
	configMap, err := c.GetConfigMap(namespace, "onepanel")
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace": namespace,
			"Error":     err.Error(),
		}).Error("getNamespaceConfig failed getting config map.")
		return
	}
	config = configMap.Data

	secret, err := c.GetSecret(namespace, "onepanel")
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace": namespace,
			"Error":     err.Error(),
		}).Error("getNamespaceConfig failed getting secret.")
		return
	}
	accessKey, _ := base64.StdEncoding.DecodeString(secret.Data[artifactRepositoryAccessKeyValueKey])
	config[artifactRepositoryAccessKeyValueKey] = string(accessKey)
	secretKey, _ := base64.StdEncoding.DecodeString(secret.Data[artifactRepositorySecretKeyValueKey])
	config[artifactRepositorySecretKeyValueKey] = string(secretKey)

	return
}

func (c *Client) GetS3Client(namespace string, config map[string]string) (s3Client *s3.Client, err error) {
	insecure, err := strconv.ParseBool(config[artifactRepositoryInSecureKey])
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace": namespace,
			"ConfigMap": config,
			"Error":     err.Error(),
		}).Error("getS3Client failed when parsing bool.")
		return
	}
	s3Client, err = s3.NewClient(s3.Config{
		Endpoint:  config[artifactRepositoryEndpointKey],
		Region:    config[artifactRepositoryRegionKey],
		AccessKey: config[artifactRepositoryAccessKeyValueKey],
		SecretKey: config[artifactRepositorySecretKeyValueKey],
		InSecure:  insecure,
	})
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace": namespace,
			"ConfigMap": config,
			"Error":     err.Error(),
		}).Error("getS3Client failed when initializing a new S3 client.")
		return
	}

	return
}
