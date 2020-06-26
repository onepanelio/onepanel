package v1

import (
	"database/sql"
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

// Test_GetWorkflowTemplateDB_Empty attempts to get a WorkflowTemplate when there isn't one.
// this should fail.
func Test_GetWorkflowTemplateDB_Empty(t *testing.T) {
	c := DefaultTestClient()
	clearDatabase(t)

	_, err := c.GetWorkflowTemplateDB("test", "test")
	assert.Equal(t, sql.ErrNoRows, err)
}

// Test_GetWorkflowTemplateDB_Exists gets a WorkflowTemplate when there is one
// this should succeed
func Test_GetWorkflowTemplateDB_Exists(t *testing.T) {
	c := DefaultTestClient()
	clearDatabase(t)

	workflowTemplate := &WorkflowTemplate{
		Name:     "test",
		Manifest: defaultWorkflowTemplate,
	}
	_, err := c.CreateWorkflowTemplate("onepanel", workflowTemplate)
	assert.Nil(t, err)

	_, err = c.GetWorkflowTemplateDB("onepanel", "test")
	assert.Nil(t, err)
}

// Test_GetWorkflowTemplateDB_InsertSameName attempts to insert a WorkflowTemplate with the same name
// this should fail
func Test_GetWorkflowTemplateDB_InsertSameName(t *testing.T) {
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

// TestClient_CreateWorkflowTemplate_Success makes sure a correct workflow template is created correctly
func TestClient_CreateWorkflowTemplate_Success(t *testing.T) {
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
