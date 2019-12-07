package model

import "github.com/google/uuid"

type WorkflowTemplate string

type Workflow struct {
	UUID     uuid.UUID
	Name     string
	Template WorkflowTemplate
}

func (wt *WorkflowTemplate) ToBytes() []byte {
	return []byte(*wt)
}
