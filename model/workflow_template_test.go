package model

import "testing"

func TestWorkflowTemplateToBytes(t *testing.T) {
	w := &WorkflowTemplate{
		Manifest: "test",
	}
	t.Log(w.GetManifest())
}
