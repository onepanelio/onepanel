package model

import "testing"

func TestWorkflowTemplateToBytes(t *testing.T) {
	var w WorkflowTemplate = "test"
	t.Log(w.ToBytes())
}
