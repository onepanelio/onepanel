package manager

import (
	"github.com/onepanelio/core/argo"
	"github.com/onepanelio/core/model"
	"github.com/spf13/viper"
)

func (r *ResourceManager) CreateWorkflow(namespace string, workflow *model.Workflow) (createdWorkflow *model.Workflow, err error) {
	r.workflowClient, err = argo.NewClient(namespace, viper.GetString("KUBECONFIG"))
	if err != nil {
		return
	}

	opts := &argo.Options{}
	for _, param := range workflow.Parameters {
		opts.Parameters = append(opts.Parameters, argo.Parameter{
			Name:  param.Name,
			Value: param.Value,
		})
	}

	createdWorkflows, err := r.workflowClient.Create(workflow.WorkflowTemplate.GetManifest(), opts)
	if err != nil {
		return
	}
	createdWorkflow = workflow
	createdWorkflow.Name = createdWorkflows[0].Name
	createdWorkflow.UID = string(createdWorkflows[0].ObjectMeta.UID)

	return
}
