package manager

import (
	"github.com/onepanelio/core/argo"
	"github.com/spf13/viper"
)

func (r *ResourceManager) CreateWorkflow(workflowTemplate string) (err error) {
	r.workflowClient, err = argo.NewClient("rushtehrani", viper.GetString("KUBECONFIG"))

	return
}
