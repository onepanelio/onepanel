package manager

import (
	"github.com/onepanelio/core/argo"
	"github.com/onepanelio/core/kube"
	"github.com/onepanelio/core/repository"
)

type ResourceManager struct {
	argClient          *argo.Client
	kubeClient         *kube.Client
	workflowRepository *repository.WorkflowRepository
}

func NewResourceManager(db *repository.DB, argoClient *argo.Client, kubeClient *kube.Client) *ResourceManager {
	return &ResourceManager{
		argClient:          argoClient,
		kubeClient:         kubeClient,
		workflowRepository: repository.NewWorkflowRepository(db),
	}
}
