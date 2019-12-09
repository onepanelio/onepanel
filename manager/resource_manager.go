package manager

import (
	"github.com/onepanelio/core/argo"
	"github.com/onepanelio/core/repository"
)

type ResourceManager struct {
	workflowClient     *argo.Client
	workflowRepository *repository.WorkflowRepository
}

func NewResourceManager(db *repository.DB) *ResourceManager {
	return &ResourceManager{
		workflowRepository: repository.NewWorkflowRepository(db),
	}
}
