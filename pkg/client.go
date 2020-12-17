package v1

import (
	"fmt"
	sq "github.com/Masterminds/squirrel"
	argoprojv1alpha1 "github.com/argoproj/argo/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	"github.com/jmoiron/sqlx"
	"github.com/onepanelio/core/pkg/util/env"
	"github.com/onepanelio/core/pkg/util/gcs"
	"github.com/onepanelio/core/pkg/util/router"
	"github.com/onepanelio/core/pkg/util/s3"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"strconv"
	"time"
)

type Config = rest.Config

var sb = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

type Client struct {
	Token string
	kubernetes.Interface
	argoprojV1alpha1 argoprojv1alpha1.ArgoprojV1alpha1Interface
	*DB
	systemConfig SystemConfig
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

// GetDefaultClient loads a default k8s client
func GetDefaultClient() (*Client, error) {
	kubeConfig := NewConfig()
	client, err := NewClient(kubeConfig, nil, nil)
	if err != nil {
		return nil, err
	}
	config, err := client.GetSystemConfig()
	if err != nil {
		return nil, err
	}

	dbDriverName, dbDataSourceName := config.DatabaseConnection()
	client.DB = NewDB(sqlx.MustConnect(dbDriverName, dbDataSourceName))

	return client, nil
}

// GetDefaultClientWithDB loads a default k8s client with an existing DB
func GetDefaultClientWithDB(db *DB) (*Client, error) {
	kubeConfig := NewConfig()
	client, err := NewClient(kubeConfig, nil, nil)
	if err != nil {
		return nil, err
	}

	client.DB = db

	return client, nil
}

// NewClient creates a client to interact with the Onepanel system.
// It includes access to the database, kubernetes, argo, and configuration.
func NewClient(config *Config, db *DB, systemConfig SystemConfig) (client *Client, err error) {
	if config.BearerToken != "" {
		config.BearerTokenFile = ""
		config.Username = ""
		config.Password = ""
		config.CertData = nil
		config.CertFile = ""
	}

	config.Timeout = getKubernetesTimeout()

	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return
	}

	argoClient, err := argoprojv1alpha1.NewForConfig(config)
	if err != nil {
		return
	}

	return &Client{
		Interface:        kubeClient,
		argoprojV1alpha1: argoClient,
		DB:               db,
		systemConfig:     systemConfig,
	}, nil
}

// GetS3Client initializes a client to Amazon Cloud Storage.
func (c *Client) GetS3Client(namespace string, config *ArtifactRepositoryS3Provider) (s3Client *s3.Client, err error) {
	s3Client, err = s3.NewClient(s3.Config{
		Endpoint:  config.Endpoint,
		Region:    config.Region,
		AccessKey: config.AccessKey,
		SecretKey: config.Secretkey,
		InSecure:  config.Insecure,
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

// GetGCSClient initializes a client to Google Cloud Storage.
func (c *Client) GetGCSClient(namespace string, config *ArtifactRepositoryGCSProvider) (gcsClient *gcs.Client, err error) {
	return gcs.NewClient(namespace, config.ServiceAccountJSON)
}

// GetWebRouter creates a new web router using the system configuration
func (c *Client) GetWebRouter() (router.Web, error) {
	sysConfig, err := c.GetSystemConfig()
	if err != nil {
		return nil, err
	}

	fqdn := sysConfig.FQDN()
	if fqdn == nil {
		return nil, fmt.Errorf("unable to get fqdn")
	}

	protocol := sysConfig.APIProtocol()
	if protocol == nil {
		return nil, fmt.Errorf("unable to get protcol")
	}

	webRouter, err := router.NewWebRouter(*protocol, *fqdn)

	return webRouter, err
}

// getKubernetesTimeout returns the timeout for kubernetes requests.
// It uses the KUBERNETES_TIMEOUT environment variable and defaults to 60 seconds if not found or an error occurs
// parsing the set timeout.
func getKubernetesTimeout() time.Duration {
	timeoutSeconds := env.Get("KUBERNETES_TIMEOUT", "180")

	timeout, err := strconv.Atoi(timeoutSeconds)
	if err != nil {
		log.Warn("Unable to parse KUBERNETES_TIMEOUT environment variable. Defaulting to 60 seconds")
		return 180 * time.Second
	}

	return time.Duration(timeout) * time.Second
}
