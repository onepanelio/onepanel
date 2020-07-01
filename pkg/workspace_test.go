package v1

import (
	"github.com/onepanelio/core/pkg/util"
	"github.com/onepanelio/core/pkg/util/ptr"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"testing"
)

// testClientPrivateCreateWorkspaceNoWorkflowTemplate makes sure we get an error when there is no workflow template for the workspace
func testClientPrivateCreateWorkspaceNoWorkflowTemplate(t *testing.T) {
	c := DefaultTestClient()
	clearDatabase(t)

	namespace := "onepanel"

	workspace := &Workspace{
		Name: "test",
		WorkspaceTemplate: &WorkspaceTemplate{
			WorkflowTemplate: &WorkflowTemplate{
				UID:     "not-exist",
				Version: 1,
			},
		},
	}
	workspace.GenerateUID("test")

	_, err := c.createWorkspace(namespace, []byte(""), workspace)

	userErr, ok := err.(*util.UserError)
	assert.True(t, ok)
	assert.Equal(t, userErr.Code, codes.NotFound)
}

// testClientPrivateCreateWorkspaceSuccess tests creating a workspace successfully
func testClientPrivateCreateWorkspaceSuccess(t *testing.T) {
	c := DefaultTestClient()
	clearDatabase(t)

	namespace := "onepanel"

	workspaceTemplate := &WorkspaceTemplate{
		Name:     "test",
		Manifest: jupyterLabWorkspaceManifest,
	}

	workspaceTemplate, _ = c.CreateWorkspaceTemplate(namespace, workspaceTemplate)

	workspace := &Workspace{
		Name:              "test2",
		WorkspaceTemplate: workspaceTemplate,
		Parameters: []Parameter{
			{
				Name:  "workflow-execution-name",
				Value: ptr.String("test2"),
			},
		},
	}
	workspace.GenerateUID("test")

	_, err := c.createWorkspace(namespace, []byte("{}"), workspace)

	assert.Nil(t, err)
}

func TestClient_createWorkspace(t *testing.T) {
	testClientPrivateCreateWorkspaceNoWorkflowTemplate(t)
	testClientPrivateCreateWorkspaceSuccess(t)
}

func TestClient_CreateWorkspace(t *testing.T) {

}
