package model

import "github.com/google/uuid"

type Workflow struct {
	ID               uint64
	UUID             uuid.UUID
	Name             string
	Parameters       []Parameter
	WorkflowTemplate WorkflowTemplate
}

type Parameter struct {
	Name  string
	Value *string
}

type WorkflowTemplate struct {
	ID       uint64
	UUID     uuid.UUID
	Name     string
	Manifest string
}

func (wt *WorkflowTemplate) ToBytes() []byte {
	return []byte(wt.Manifest)
}
