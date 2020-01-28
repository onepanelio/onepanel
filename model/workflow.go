package model

import "time"

type Workflow struct {
	ID               uint64
	CreatedAt        time.Time `db:"created_at"`
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
