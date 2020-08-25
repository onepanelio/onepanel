package v1

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

// TestCronWorkflow_GetParametersFromWorkflowSpec makes sure the GetParametersFromWorkflowSpec method works
func TestCronWorkflow_GetParametersFromWorkflowSpec(t *testing.T) {
	manifest := `concurrencyPolicy: Allow
failedJobsHistoryLimit: 1
schedule: '* * * * 2'
startingDeadlineSeconds: 0
successfulJobsHistoryLimit: 3
suspend: false
timezone: Etc/UTC
workflowSpec:
  arguments:
    parameters:
    - displayname: ""
      hint: ""
      name: source
      options: []
      required: false
      type: ""
      value: https://github.com/onepanelio/Mask_RCNN.git
    - displayname: ""
      hint: ""
      name: dataset-path
      options: []
      required: false
      type: ""
      value: datasets/test_05142020170720
    - displayname: ""
      hint: ""
      name: model-path
      options: []
      required: false
      type: ""
      value: models/rush/cvat6-20
    - displayname: ""
      hint: ""
      name: extras
      options: []
      required: false
      type: ""
      value: none
    - displayname: ""
      hint: ""
      name: task-name
      options: []
      required: false
      type: ""
      value: test
    - displayname: ""
      hint: ""
      name: num-classes
      options: []
      required: false
      type: ""
      value: "2"
    - displayname: ""
      hint: ""
      name: stage-1-epochs
      options: []
      required: false
      type: ""
      value: "1"
    - displayname: ""
      hint: ""
      name: stage-2-epochs
      options: []
      required: false
      type: ""
      value: "2"
    - displayname: ""
      hint: ""
      name: stage-3-epochs
      options: []
      required: false
      type: ""
      value: "3"
    - displayname: ""
      hint: ""
      name: tf-image
      options: []
      required: false
      type: ""
      value: tensorflow/tensorflow:1.13.1-py3
    - displayname: Node pool
      hint: Name of node pool or group
      name: sys-node-pool
      options:
      - name: 'CPU: 2, RAM: 8GB'
        value: Standard_D2s_v3
      - name: 'CPU: 4, RAM: 16GB'
        value: Standard_D4s_v3
      - name: 'GPU: 1xK80, CPU: 6, RAM: 56GB'
        value: Standard_NC6
      required: true
      type: select.select
      value: cake`

	cronWorkflow := CronWorkflow{
		Manifest: manifest,
	}

	parameters, err := cronWorkflow.GetParametersFromWorkflowSpec()
	assert.Nil(t, err)
	assert.NotNil(t, parameters)

	assert.Len(t, parameters, 11)
}
