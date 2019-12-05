package model

type WorkflowTemplate string

type Workflow struct {
	UUID     string
	Name     string
	Template WorkflowTemplate
}

func (wt *WorkflowTemplate) ToBytes() []byte {
	return []byte(*wt)
}
