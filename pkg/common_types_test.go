package v1

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

// TestParseParametersFromManifest makes sure that we have correct parsing of parameters from a manifest
func TestParseParametersFromManifest(t *testing.T) {
	manifest := `arguments:
  parameters:
  - name: source
    value: https://github.com/onepanelio/Mask_RCNN.git
  - name: dataset-path
    value: datasets/test_05142020170720
    visibility: public
  - name: model-path
    value: models/rush/cvat6-20
  - name: extras
    value: none
  - name: task-name
    value: test
  - name: num-classes
    value: 2
  - name: tf-image
    value: tensorflow/tensorflow:1.13.1-py3
  - displayName: Node pool
    hint: Name of node pool or group
    type: select.select
    name: sys-node-pool
    required: true
    options:
    - name: 'CPU: 2, RAM: 8GB'
      value: Standard_D2s_v3
    - name: 'CPU: 4, RAM: 16GB'
      value: Standard_D4s_v3
    - name: 'GPU: 1xK80, CPU: 6, RAM: 56GB'
      value: Standard_NC6
`

	parameters, err := ParseParametersFromManifest([]byte(manifest))
	assert.Nil(t, err)
	assert.NotNil(t, parameters)
	assert.Len(t, parameters, 8)

	keyedParameters := MapParametersByName(parameters)

	// Make sure visibility is set
	assert.Equal(t, *keyedParameters["dataset-path"].Visibility, "public")

	// Make sure visibility is not set if omitted
	assert.Nil(t, keyedParameters["tf-image"].Visibility)

	// Make sure numbers, slashes, dashes, and letters are parsed correctly
	assert.Equal(t, *keyedParameters["tf-image"].Value, "tensorflow/tensorflow:1.13.1-py3")

	// Make sure integers are parsed as strings and not ignored or omitted
	assert.Equal(t, *keyedParameters["num-classes"].Value, "2")

	// Make sure missing values have a nil value to show they are not there
	assert.Nil(t, keyedParameters["sys-node-pool"].Value, nil)

	// Make sure options are parsed
	assert.Len(t, keyedParameters["sys-node-pool"].Options, 3)

	// Make sure string values are correctly parsed
	assert.Equal(t, *keyedParameters["extras"].Value, "none")
}
