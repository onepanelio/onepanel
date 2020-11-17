package v1

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

// TestMetrics_Add tests the Add method of the Metrics type
func TestMetrics_Add(t *testing.T) {
	var initial Metrics = []*Metric{{
		Name:   "accuracy",
		Value:  0.98,
		Format: "",
	}}

	initial.Add(&Metric{
		Name:   "success",
		Value:  1.0,
		Format: "%",
	}, false)

	assert.Len(t, initial, 2)

	initial.Add(&Metric{
		Name:   "accuracy",
		Value:  0.99,
		Format: "%",
	}, false)

	assert.Len(t, initial, 2)

	initial.Add(&Metric{
		Name:   "accuracy",
		Value:  0.99,
		Format: "%",
	}, true)

	assert.Len(t, initial, 2)
	assert.True(t, initial[0].Value == 0.99)
}

// TestMetrics_Merge tests the Merge method of the Metrics Type
func TestMetrics_Merge(t *testing.T) {
	var initial Metrics = []*Metric{{
		Name:   "accuracy",
		Value:  0.98,
		Format: "",
	}, {
		Name:   "success",
		Value:  1.0,
		Format: "%",
	}}

	var toMerge Metrics = []*Metric{{
		Name:   "accuracy",
		Value:  0.00,
		Format: "",
	}, {
		Name:   "success",
		Value:  1.0,
		Format: "%",
	}, {
		Name:   "test",
		Value:  0.5,
		Format: "",
	}}

	initial.Merge(toMerge, true)

	assert.Len(t, initial, 3)
	assert.True(t, initial[0].Value == 0.00)
}
