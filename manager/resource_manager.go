package manager

import (
	"github.com/onepanelio/core/kube"
	"github.com/onepanelio/core/repository"
	"github.com/onepanelio/core/s3"
)

type ResourceManager struct {
	kubeClient         *kube.Client
	workflowRepository *repository.WorkflowRepository
}

const (
	artifactRepositoryEndpointKey       = "artifactRepositoryEndpoint"
	artifactRepositoryBucketKey         = "artifactRepositoryBucket"
	artifactRepositoryRegionKey         = "artifactRepositoryRegion"
	artifactRepositoryAccessKeyValueKey = "artifactRepositoryAccessKeyValue"
	artifactRepositorySecretKeyValueKey = "artifactRepositorySecretKeyValue"
)

func (r *ResourceManager) getNamespaceConfig(namespace string) (config map[string]string, err error) {
	configMap, err := r.kubeClient.GetConfigMap(namespace, "onepanel")
	if err != nil {
		return
	}
	config = configMap.Data

	secret, err := r.kubeClient.GetSecret(namespace, "onepanel")
	if err != nil {
		return
	}
	config[artifactRepositoryAccessKeyValueKey] = secret.Data[artifactRepositoryAccessKeyValueKey]
	config[artifactRepositorySecretKeyValueKey] = secret.Data[artifactRepositorySecretKeyValueKey]

	return
}

func (r *ResourceManager) getS3Client(namespace string, config map[string]string) (s3Client *s3.Client, err error) {
	s3Client, err = s3.NewClient(s3.Config{
		Endpoint:  config[artifactRepositoryEndpointKey],
		Region:    config[artifactRepositoryRegionKey],
		AccessKey: config[artifactRepositoryAccessKeyValueKey],
		SecretKey: config[artifactRepositorySecretKeyValueKey],
	})
	if err != nil {
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
