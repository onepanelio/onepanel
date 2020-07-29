package v1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestLabelFromString tests the LabelFromString function
func TestLabelFromString(t *testing.T) {
	// Blank value gives us no label
	label, err := LabelFromString("")
	assert.NotNil(t, err)
	assert.Nil(t, label)

	// Missing value, should give error
	label, err = LabelFromString("key=a")
	assert.NotNil(t, err)

	// Missing value, but still have comma, should give error
	label, err = LabelFromString("key=a,")
	assert.NotNil(t, err)

	// Missing key, should give error
	label, err = LabelFromString("value=a")
	assert.NotNil(t, err)

	// Missing key, still have comma, should give error
	label, err = LabelFromString("value=a,")
	assert.NotNil(t, err)

	// Correct, should not give an error
	label, err = LabelFromString("key=a,value=b")
	assert.Nil(t, err)
	assert.Equal(t, label.Key, "a")
	assert.Equal(t, label.Value, "b")
}

// TestLabelsFromString tests the LabelsFromString function
func TestLabelsFromString(t *testing.T) {
	// Empty should give no error and no labels
	labels, err := LabelsFromString("")
	assert.Nil(t, err)
	assert.Len(t, labels, 0)

	// Bad data, should give no labels
	labels, err = LabelsFromString("&&&&")
	assert.Nil(t, err)
	assert.Len(t, labels, 0)

	// Test just one label
	labels, err = LabelsFromString("key=a,value=b")
	assert.Nil(t, err)
	assert.Len(t, labels, 1)

	// Test many labels
	labels, err = LabelsFromString("key=a,value=b&key=c,value=d&key=e,value=f")
	assert.Nil(t, err)
	assert.Len(t, labels, 3)
}
