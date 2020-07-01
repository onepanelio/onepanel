package v1

import (
	"database/sql"
	"fmt"
	"github.com/onepanelio/core/pkg/util"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"testing"
)

const defaultWorkflowTemplate = `entrypoint: main
arguments:
    parameters:
    - name: source
      value: https://github.com/onepanelio/pytorch-examples.git
    - name: command
      value: "python mnist/main.py --epochs=1"
volumeClaimTemplates:
  - metadata:
      name: data
    spec:
      accessModes: [ "ReadWriteOnce" ]
      resources:
        requests:
          storage: 2Gi
  - metadata:
      name: output
    spec:
      accessModes: [ "ReadWriteOnce" ]
      resources:
        requests:
          storage: 2Gi
templates:
  - name: main
    dag:
      tasks:
      - name: train-model
        template: pytorch
# Uncomment section below to send metrics to Slack
#      - name: notify-in-slack
#        dependencies: [train-model]
#        template: slack-notify-success
#        arguments:
#          parameters:
#          - name: status
#            value: "{{tasks.train-model.status}}"
#          artifacts:
#          - name: metrics
#            from: "{{tasks.train-model.outputs.artifacts.sys-metrics}}"
  - name: pytorch
    inputs:
      artifacts:
      - name: src
        path: /mnt/src
        git:
          repo: "{{workflow.parameters.source}}"
    outputs:
      artifacts:
      - name: model
        path: /mnt/output
        optional: true
        archive:
          none: {}
    container:
      image: pytorch/pytorch:latest
      command: [sh,-c]
      args: ["{{workflow.parameters.command}}"]
      workingDir: /mnt/src
      volumeMounts:
      - name: data
        mountPath: /mnt/data
      - name: output
        mountPath: /mnt/output
  - name: slack-notify-success
    container:
      image: technosophos/slack-notify
      command: [sh,-c]
      args: ['SLACK_USERNAME=Worker SLACK_TITLE="{{workflow.name}} {{inputs.parameters.status}}" SLACK_ICON=https://www.gravatar.com/avatar/5c4478592fe00878f62f0027be59c1bd SLACK_MESSAGE=$(cat /tmp/metrics.json)} ./slack-notify']
    inputs:
      parameters:
      - name: status
      artifacts:
      - name: metrics
        path: /tmp/metrics.json
        optional: true
`

// testClientGetWorkflowTemplateDBEmpty attempts to get a WorkflowTemplate when there isn't one.
// this should fail.
func testClientGetWorkflowTemplateDBEmpty(t *testing.T) {
	c := DefaultTestClient()
	clearDatabase(t)

	_, err := c.getWorkflowTemplateDB("test", "test")
	assert.Equal(t, sql.ErrNoRows, err)
}

// testClientGetWorkflowTemplateDBExists gets a WorkflowTemplate when there is one
// this should succeed
func testClientGetWorkflowTemplateDBExists(t *testing.T) {
	c := DefaultTestClient()
	clearDatabase(t)

	workflowTemplate := &WorkflowTemplate{
		Name:     "test",
		Manifest: defaultWorkflowTemplate,
	}
	_, err := c.CreateWorkflowTemplate("onepanel", workflowTemplate)
	assert.Nil(t, err)

	_, err = c.getWorkflowTemplateDB("onepanel", "test")
	assert.Nil(t, err)
}

// TestClient_getWorkflowTemplateDB tests getting a workflow template from the database
func TestClient_getWorkflowTemplateDB(t *testing.T) {
	testClientGetWorkflowTemplateDBEmpty(t)
	testClientGetWorkflowTemplateDBExists(t)
}

// testClientCreateWorkflowTemplateSuccess makes sure a correct workflow template is created correctly
func testClientCreateWorkflowTemplateSuccess(t *testing.T) {
	c := DefaultTestClient()
	clearDatabase(t)

	workflowTemplate := &WorkflowTemplate{
		Name:     "test",
		Manifest: defaultWorkflowTemplate,
	}

	wft, err := c.CreateWorkflowTemplate("onepanel", workflowTemplate)
	assert.Nil(t, err)
	assert.NotNil(t, wft.ArgoWorkflowTemplate)
}

// testClientCreateWorkflowTemplateTimestamp makes sure we can create mulitple
// workflow templtate versions one after another with practically no time delay.
// This handles an edge case where versions were set using second time precision and could fail in migrations
// as they were created one after another.
func testClientCreateWorkflowTemplateTimestamp(t *testing.T) {
	c := DefaultTestClient()
	clearDatabase(t)

	namespace := "onepanel"

	workflowTemplate := &WorkflowTemplate{
		Name:     "test",
		Manifest: defaultWorkflowTemplate,
	}

	// This method creates a workflow template version underneath
	wft, err := c.CreateWorkflowTemplate("onepanel", workflowTemplate)
	assert.Nil(t, err)
	assert.NotNil(t, wft.ArgoWorkflowTemplate)

	// This method creates a brand new version
	wft, err = c.CreateWorkflowTemplateVersion(namespace, workflowTemplate)
	assert.Nil(t, err)
	assert.NotNil(t, wft.ArgoWorkflowTemplate)
}

// testClientCreateWorkflowTemplateInsertSameName attempts to insert a WorkflowTemplate with the same name
// this should fail
func testClientCreateWorkflowTemplateInsertSameName(t *testing.T) {
	c := DefaultTestClient()
	clearDatabase(t)

	workflowTemplate := &WorkflowTemplate{
		Name:     "test",
		Manifest: defaultWorkflowTemplate,
	}
	_, err := c.CreateWorkflowTemplate("onepanel", workflowTemplate)
	assert.Nil(t, err)

	_, err = c.CreateWorkflowTemplate("onepanel", workflowTemplate)
	assert.NotNil(t, err)

	assert.IsType(t, &util.UserError{}, err)
	userErr := err.(*util.UserError)

	assert.Equal(t, userErr.Code, codes.AlreadyExists)
}

// TestClient_CreateWorkflowTemplate tests creating a workflow template
func TestClient_CreateWorkflowTemplate(t *testing.T) {
	testClientCreateWorkflowTemplateInsertSameName(t)
	testClientCreateWorkflowTemplateSuccess(t)
	testClientCreateWorkflowTemplateTimestamp(t)
}

// testClientPrivateGetWorkflowTemplateSuccess gets a workflow template with no error conditions encountered
func testClientPrivateGetWorkflowTemplateSuccess(t *testing.T) {
	c := DefaultTestClient()
	clearDatabase(t)

	namespace := "onepanel"
	workflowTemplate := &WorkflowTemplate{
		Name:     "test",
		Manifest: defaultWorkflowTemplate,
	}
	created, _ := c.CreateWorkflowTemplate(namespace, workflowTemplate)

	wt, err := c.getWorkflowTemplate(namespace, created.UID, 0)
	assert.NotNil(t, wt)
	assert.Nil(t, err)
}

// testClientGetWorkflowTemplateSuccessVersion gets a workflow template for a specific version with no error conditions encountered
func testClientPrivateGetWorkflowTemplateSuccessVersion(t *testing.T) {
	c := DefaultTestClient()
	clearDatabase(t)

	namespace := "onepanel"
	workflowTemplate := &WorkflowTemplate{
		Name:     "test",
		Manifest: defaultWorkflowTemplate,
	}
	created, _ := c.CreateWorkflowTemplate(namespace, workflowTemplate)
	c.CreateWorkflowTemplateVersion(namespace, workflowTemplate)

	wt, err := c.getWorkflowTemplate(namespace, created.UID, created.Version)
	assert.NotNil(t, wt)
	assert.Nil(t, err)

	assert.Equal(t, created.Version, wt.Version)
	assert.Equal(t, created.Manifest, wt.Manifest)
}

// testClientGetWorkflowTemplateNotFound attempts to get a not-found workflow template
func testClientPrivateGetWorkflowTemplateNotFound(t *testing.T) {
	c := DefaultTestClient()
	clearDatabase(t)

	wt, err := c.getWorkflowTemplate("onepanel", "uid-not-found", 0)
	assert.Nil(t, wt)
	assert.Nil(t, err)
}

// Test_getWorkflowTemplate tests getting a workflow template
func Test_getWorkflowTemplate(t *testing.T) {
	testClientPrivateGetWorkflowTemplateSuccess(t)
	testClientPrivateGetWorkflowTemplateNotFound(t)
	testClientPrivateGetWorkflowTemplateSuccessVersion(t)
}

func TestClient_getWorkflowTemplateVersionDB(t *testing.T) {
	c := DefaultTestClient()
	clearDatabase(t)

	namespace := "onepanel"
	name := "test"
	workflowTemplate := &WorkflowTemplate{
		Name:     name,
		Manifest: defaultWorkflowTemplate,
	}

	original, _ := c.CreateWorkflowTemplate(namespace, workflowTemplate)

	versionAsString := fmt.Sprintf("%v", original.Version)
	originalRes, err := c.getWorkflowTemplateVersionDB(namespace, name, versionAsString)

	assert.Nil(t, err)
	assert.Equal(t, original.Version, originalRes.Version)
}

// testClientCreateWorkflowTemplateVersionNew makes sure you can successfully create a new workflow template version
func testClientCreateWorkflowTemplateVersionNew(t *testing.T) {
	c := DefaultTestClient()
	clearDatabase(t)

	namespace := "onepanel"
	workflowTemplate := &WorkflowTemplate{
		Name:     "test",
		Manifest: defaultWorkflowTemplate,
	}

	c.CreateWorkflowTemplate(namespace, workflowTemplate)
	_, err := c.CreateWorkflowTemplateVersion(namespace, workflowTemplate)

	assert.Nil(t, err)
}

// testClientCreateWorkflowTemplateVersionMarkOldNotLatest makes sure older versions are no longer marked as latest
func testClientCreateWorkflowTemplateVersionMarkOldNotLatest(t *testing.T) {
	c := DefaultTestClient()
	clearDatabase(t)

	namespace := "onepanel"
	name := "test"
	workflowTemplate := &WorkflowTemplate{
		Name:     name,
		Manifest: defaultWorkflowTemplate,
	}

	original, _ := c.CreateWorkflowTemplate(namespace, workflowTemplate)
	originalVersionAsString := fmt.Sprintf("%v", original.Version)
	c.CreateWorkflowTemplateVersion(namespace, workflowTemplate)

	updated, _ := c.getWorkflowTemplateVersionDB(namespace, name, originalVersionAsString)

	assert.False(t, updated.IsLatest)
}

// Test_getWorkflowTemplate_SuccessVersion tests cases for creating a workflow template version
func TestClient_CreateWorkflowTemplateVersion(t *testing.T) {
	testClientCreateWorkflowTemplateVersionNew(t)
	testClientCreateWorkflowTemplateVersionMarkOldNotLatest(t)
}

// testGetWorkflowTemplateSuccess gets a workflow template with no error conditions encountered
func testClientGetWorkflowTemplateSuccess(t *testing.T) {
	c := DefaultTestClient()
	clearDatabase(t)

	namespace := "onepanel"
	workflowTemplate := &WorkflowTemplate{
		Name:     "test",
		Manifest: defaultWorkflowTemplate,
	}
	created, _ := c.CreateWorkflowTemplate(namespace, workflowTemplate)

	wt, err := c.GetWorkflowTemplate(namespace, created.UID, 0)
	assert.NotNil(t, wt)
	assert.Nil(t, err)
}

// testGetWorkflowTemplateNotFound attempts to get a not-found workflow template
func testClientGetWorkflowTemplateNotFound(t *testing.T) {
	c := DefaultTestClient()
	clearDatabase(t)

	wt, err := c.GetWorkflowTemplate("onepanel", "uid-not-found", 0)
	assert.Nil(t, wt)

	userErr, ok := err.(*util.UserError)
	assert.True(t, ok)

	assert.Equal(t, codes.NotFound, userErr.Code)
}

func TestClient_GetWorkflowTemplate(t *testing.T) {
	testClientGetWorkflowTemplateSuccess(t)
	testClientGetWorkflowTemplateNotFound(t)
}
