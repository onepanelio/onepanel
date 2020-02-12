package model

import "time"

type WorkflowPhase string

const (
	WorfklowPending   WorkflowPhase = "Pending"
	WorfklowRunning   WorkflowPhase = "Running"
	WorfklowSucceeded WorkflowPhase = "Succeeded"
	WorfklowSkipped   WorkflowPhase = "Skipped"
	WorfklowFailed    WorkflowPhase = "Failed"
	WorfklowError     WorkflowPhase = "Error"
)

type Workflow struct {
	ID               uint64
	CreatedAt        time.Time `db:"created_at"`
	UID              string
	Name             string
	GenerateName     string
	Parameters       []Parameter
	Manifest         string
	Phase            WorkflowPhase
	StartedAt        time.Time
	FinishedAt       time.Time
	WorkflowTemplate *WorkflowTemplate
}

type Parameter struct {
	Name  string
	Value *string
}
