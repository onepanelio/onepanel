package v1

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestClient_CreateWorkflowExecution(t *testing.T) {
	c := DefaultTestClient()
	clearDatabase(t)

	namespace := "onepanel"

	wt := &WorkflowTemplate{
		Name:     "test",
		Manifest: defaultWorkflowTemplate,
	}
	wt, _ = c.CreateWorkflowTemplate(namespace, wt)

	we := &WorkflowExecution{
		Name: "test",
	}

	we, err := c.CreateWorkflowExecution(namespace, we, wt)
	assert.Nil(t, err)
}

// TestClient_ArchiveWorkflowExecution_NotExist makes sure there is no error if the workflow
// execution does not exist
func TestClient_ArchiveWorkflowExecution_NotExist(t *testing.T) {
	c := DefaultTestClient()
	clearDatabase(t)

	err := c.ArchiveWorkflowExecution("onepanel-no-exist", "test-no-exist")
	assert.Nil(t, err)
}

// TestClient_ArchiveWorkflowExecution_Exist makes sure we archive an existing workflow execution correctly
func TestClient_ArchiveWorkflowExecution_Exist(t *testing.T) {
	c := DefaultTestClient()
	clearDatabase(t)

	namespace := "onepanel"
	weName := "test"

	wt := &WorkflowTemplate{
		Name:     "test",
		Manifest: defaultWorkflowTemplate,
	}
	wt, _ = c.CreateWorkflowTemplate(namespace, wt)

	we := &WorkflowExecution{
		Name: weName,
	}

	we, err := c.CreateWorkflowExecution(namespace, we, wt)

	err = c.ArchiveWorkflowExecution(namespace, weName)
	assert.Nil(t, err)
}
