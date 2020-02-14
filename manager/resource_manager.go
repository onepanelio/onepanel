package manager

import (
	"strconv"

	"github.com/onepanelio/core/util/logging"
	log "github.com/sirupsen/logrus"

	"github.com/onepanelio/core/kube"
	"github.com/onepanelio/core/repository"
	"github.com/onepanelio/core/s3"
	"github.com/onepanelio/core/util/env"
)

var (
	labelKeyPrefix = env.GetEnv("KUBE_LABEL_KEY_PREFIX", "onepanel.io/")
)

type ResourceManager struct {
	kubeClient         *kube.Client
	workflowRepository *repository.WorkflowRepository
}

const (
	artifactRepositoryEndpointKey       = "artifactRepositoryEndpoint"
	artifactRepositoryBucketKey         = "artifactRepositoryBucket"
	artifactRepositoryRegionKey         = "artifactRepositoryRegion"
	artifactRepositoryInSecureKey       = "artifactRepositoryInsecure"
	artifactRepositoryAccessKeyValueKey = "artifactRepositoryAccessKey"
	artifactRepositorySecretKeyValueKey = "artifactRepositorySecretKey"
)

func (r *ResourceManager) getNamespaceConfig(namespace string) (config map[string]string, err error) {
	configMap, err := r.kubeClient.GetConfigMap(namespace, "onepanel")
	if err != nil {
		logging.Logger.Log.WithFields(log.Fields{
			"Namespace": namespace,
			"Error":     err.Error(),
		}).Error("getNamespaceConfig failed getting config map.")
		return
	}
	config = configMap.Data

	secret, err := r.kubeClient.GetSecret(namespace, "onepanel")
	if err != nil {
		logging.Logger.Log.WithFields(log.Fields{
			"Namespace": namespace,
			"Error":     err.Error(),
		}).Error("getNamespaceConfig failed getting secret.")
		return
	}
	config[artifactRepositoryAccessKeyValueKey] = secret.Data[artifactRepositoryAccessKeyValueKey]
	config[artifactRepositorySecretKeyValueKey] = secret.Data[artifactRepositorySecretKeyValueKey]

	return
}

func (r *ResourceManager) getS3Client(namespace string, config map[string]string) (s3Client *s3.Client, err error) {
	insecure, err := strconv.ParseBool(config[artifactRepositoryInSecureKey])
	if err != nil {
		logging.Logger.Log.WithFields(log.Fields{
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
		logging.Logger.Log.WithFields(log.Fields{
			"Namespace": namespace,
			"ConfigMap": config,
			"Error":     err.Error(),
		}).Error("getS3Client failed when initializing a new S3 client.")
		return
	}

	return
}

func NewResourceManager(db *repository.DB, kubeClient *kube.Client) *ResourceManager {
	return &ResourceManager{
		kubeClient:         kubeClient,
		workflowRepository: repository.NewWorkflowRepository(db),
	}
}

func (r *ResourceManager) GetClient(token string) (*kube.Client, error) {
	return kube.GetClient(token)
}
