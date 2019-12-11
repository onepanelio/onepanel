package manager

import (
	"github.com/onepanelio/core/argo"
	"github.com/onepanelio/core/repository"
)

type ResourceManager struct {
	argClient          *argo.Client
	workflowRepository *repository.WorkflowRepository
}

func NewResourceManager(db *repository.DB, argoClient *argo.Client) *ResourceManager {
	return &ResourceManager{
		argClient:          argoClient,
		workflowRepository: repository.NewWorkflowRepository(db),
	}
}
