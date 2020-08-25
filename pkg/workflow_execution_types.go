package v1

import (
	"encoding/json"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/onepanelio/core/pkg/util/types"
	uid2 "github.com/onepanelio/core/pkg/util/uid"
	"github.com/onepanelio/core/util/sql"
	"time"
)

// WorkflowExecution represents a workflow that is either in execution, or finished/failed.
type WorkflowExecution struct {
	ID               uint64
	CreatedAt        time.Time `db:"created_at"`
	UID              string
	Name             string
	Namespace        string
	GenerateName     string
	Parameters       []Parameter
	ParametersBytes  []byte `db:"parameters"` // to load from database
	Manifest         string
	Phase            wfv1.NodePhase
	StartedAt        *time.Time        `db:"started_at"`
	FinishedAt       *time.Time        `db:"finished_at"`
	WorkflowTemplate *WorkflowTemplate `db:"workflow_template"`
	Labels           types.JSONLabels
	ArgoWorkflow     *wfv1.Workflow
}

// WorkflowExecutionOptions are options you have for an executing workflow
type WorkflowExecutionOptions struct {
	Name           string
	GenerateName   string
	Entrypoint     string
	Parameters     []Parameter
	ServiceAccount string
	Labels         map[string]string
	ListOptions    *ListOptions
	PodGCStrategy  *PodGCStrategy
}

// WorkflowExecutionStatistic is a record keeping track of what happened to a workflow execution
type WorkflowExecutionStatistic struct {
	ID                 uint64
	WorkflowTemplateID uint64
	Name               string
	Namespace          string
	//Interface to support null values for timestamps, when scanning from db into structs
	CreatedAt  *time.Time `db:"created_at"`
	FinishedAt *time.Time `db:"finished_at"`
	FailedAt   *time.Time `db:"failed_at"`
}

// WorkflowExecutionStatus represents the status of a workflow execution. It's a convenience struct.
type WorkflowExecutionStatus struct {
	Phase      wfv1.NodePhase `json:"phase"`
	StartedAt  *time.Time     `db:"started_at" json:"startedAt"`
	FinishedAt *time.Time     `db:"finished_at" json:"finishedAt"`
}

// GenerateUID generates a uid from the input name and sets it on the workflow execution
func (we *WorkflowExecution) GenerateUID(name string) error {
	result, err := uid2.GenerateUID(name, 63)
	if err != nil {
		return err
	}

	we.UID = result

	return nil
}

// LoadParametersFromBytes loads Parameters from the WorkflowExecution's ParameterBytes field.
func (we *WorkflowExecution) LoadParametersFromBytes() ([]Parameter, error) {
	loadedParameters := make([]Parameter, 0)

	err := json.Unmarshal(we.ParametersBytes, &loadedParameters)
	if err != nil {
		return we.Parameters, err
	}

	// It might be nil because the value "null" is stored in db if there are no parameters.
	// for consistency, we return an empty array.
	if loadedParameters == nil {
		loadedParameters = make([]Parameter, 0)
	}

	we.Parameters = loadedParameters

	return we.Parameters, err
}

// GetParameterValue returns the value of the parameter with the given name, or nil if there is no such parameter
func (we *WorkflowExecution) GetParameterValue(name string) *string {
	for _, p := range we.Parameters {
		if p.Name == name {
			return p.Value
		}
	}

	return nil
}

// getWorkflowExecutionColumns returns all of the columns for workflowExecution modified by alias, destination.
// see formatColumnSelect
func getWorkflowExecutionColumns(aliasAndDestination ...string) []string {
	columns := []string{"id", "created_at", "uid", "name", "parameters", "phase", "started_at", "finished_at", "labels"}
	return sql.FormatColumnSelect(columns, aliasAndDestination...)
}
