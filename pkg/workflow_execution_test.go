package v1

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

// TestClient_CreateWorkflowExecution tests creating a workflow execution
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

// TestClient_GetWorkflowExecution tests getting a workflow execution that exists
func TestClient_GetWorkflowExecution(t *testing.T) {
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

	we, _ = c.CreateWorkflowExecution(namespace, we, wt)

	getWe, err := c.GetWorkflowExecution(namespace, we.UID)
	assert.Nil(t, err)

	assert.Equal(t, we.Name, getWe.Name)
	assert.Equal(t, we.UID, getWe.UID)
}

// TestClient_GetWorkflowExecution tests getting a workflow execution that doesn't exist
func TestClient_GetWorkflowExecution_NotExists(t *testing.T) {
	c := DefaultTestClient()
	clearDatabase(t)

	namespace := "onepanel"

	getWe, err := c.GetWorkflowExecution(namespace, "not-exist")
	assert.Nil(t, getWe)
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
