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
	config["artifactRepositoryS3AccessKeyValue"] = secret.Data["artifactRepositoryS3AccessKeyValue"]
	config["artifactRepositoryS3SecretKeyValue"] = secret.Data["artifactRepositoryS3SecretKeyValue"]

	return
}

func (r *ResourceManager) getS3Client(namespace string, config map[string]string) (s3Client *s3.Client, err error) {
	s3Client, err = s3.NewClient(s3.Config{
		Endpoint:  config["artifactRepositoryS3Endpoint"],
		Region:    config["artifactRepositoryS3Region"],
		AccessKey: config["artifactRepositoryS3AccessKeyValue"],
		SecretKey: config["artifactRepositoryS3SecretKeyValue"],
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
