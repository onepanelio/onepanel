package v1

import (
	"github.com/onepanelio/core/pkg/util"
	"github.com/onepanelio/core/pkg/util/validate"
	"google.golang.org/grpc/codes"
)

func (c *Client) CreateWorkspace(namespace string, workspace *Workspace) (err error) {
	for _, p := range workspace.Parameters {
		if p.Name == "sys-name" {
			if p.Value == nil || !validate.IsDNSHost(*p.Value) {
				return util.NewUserError(codes.InvalidArgument, "Workspace name is not valid.")
			}
			break
		}
	}

	_, err = c.CreateWorkflowExecution(namespace, &WorkflowExecution{
		Parameters:       workspace.Parameters,
		WorkflowTemplate: nil,
	})

	return
}
