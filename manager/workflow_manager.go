package manager

import (
	"github.com/onepanelio/core/argo"
	"github.com/onepanelio/core/model"
	"github.com/onepanelio/core/util"
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
	createdWorkflowTemplate, err = r.workflowRepository.CreateWorkflowTemplate(workflowTemplate)
	if err != nil {
		return nil, util.UserErrorWrap(err, "Workflow template")
	}

	return
}

func (r *ResourceManager) GetWorkflowTemplate(namespace, uid string) (workflowTemplate *model.WorkflowTemplate, err error) {
	workflowTemplate, err = r.workflowRepository.GetWorkflowTemplate(uid)
	if err != nil {
		return nil, util.NewUserError(404, "Workflow template not found.")
	}

	return
}
