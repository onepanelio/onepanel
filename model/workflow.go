package model

type Workflow struct {
	ID               uint64
	UID              string
	Name             string
	GeneratedName    string
	Parameters       []Parameter
	Status           string
	WorkflowTemplate *WorkflowTemplate
}

type Parameter struct {
	Name  string
	Value *string
}
