package manager

import (
	"github.com/onepanelio/core/argo"
	"github.com/onepanelio/core/model"
)

func (r *ResourceManager) CreateWorkflow(namespace string, workflow *model.Workflow) (createdWorkflow *model.Workflow, err error) {
	opts := &argo.Options{
		Namespace: namespace,
	}
	for _, param := range workflow.Parameters {
		opts.Parameters = append(opts.Parameters, argo.Parameter{
			Name:  param.Name,
			Value: param.Value,
		})
	}

	createdWorkflows, err := r.argClient.Create(workflow.WorkflowTemplate.GetManifestBytes(), opts)
	if err != nil {
		return
	}
	createdWorkflow = workflow
	createdWorkflow.Name = createdWorkflows[0].Name
	createdWorkflow.UID = string(createdWorkflows[0].ObjectMeta.UID)

	return
}

func (r *ResourceManager) CreateWorkflowTemplate(namespace string, workflowTemplate *model.WorkflowTemplate) (createdWorkflowTemplate *model.WorkflowTemplate, err error) {
	return r.workflowRepository.CreateWorkflowTemplate(workflowTemplate)
}
