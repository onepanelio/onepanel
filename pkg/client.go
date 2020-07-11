package v1

import (
	"cloud.google.com/go/storage"
	sq "github.com/Masterminds/squirrel"
	argoprojv1alpha1 "github.com/argoproj/argo/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	"github.com/onepanelio/core/pkg/util/s3"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Config = rest.Config

var sb = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

type Client struct {
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

func (c *Client) GetS3Client(namespace string, config *ArtifactRepositoryS3Config) (s3Client *s3.Client, err error) {
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

func (c *Client) GetGCSClient(namespace string, config *ArtifactRepositoryGCSConfig) (gcsClient *storage.Client, err error) {
	ctx := context.Background()
	creds, err := google.CredentialsFromJSON(ctx, []byte(config.ServiceAccountJSON), storage.ScopeReadWrite)
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace": namespace,
			"ConfigMap": config,
			"Error":     err.Error(),
		}).Error("GetGCSClient failed when initializing a new GCS client.")
		return
	}
	client, err := storage.NewClient(ctx, option.WithCredentials(creds))
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace": namespace,
			"ConfigMap": config,
			"Error":     err.Error(),
		}).Error("GetGCSClient failed when initializing a new GCS client.")
		return
	}
	return client, nil
}
