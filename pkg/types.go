package v1

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"strings"
	"time"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	TypeWorkflowTemplate         string = "workflow_template"
	TypeWorkflowTemplateVersion  string = "workflow_template_version"
	TypeWorkflowExecution        string = "workflow_execution"
	TypeCronWorkflow             string = "cron_workflow"
	TypeWorkspaceTemplate        string = "workspace_template"
	TypeWorkspaceTemplateVersion string = "workspace_template_version"
	TypeWorkspace                string = "workspace"
)

func TypeToTableName(value string) string {
	switch value {
	case TypeWorkflowTemplate:
		return "workflow_templates"
	case TypeWorkflowTemplateVersion:
		return "workflow_template_versions"
	case TypeWorkflowExecution:
		return "workflow_executions"
	case TypeCronWorkflow:
		return "cron_workflows"
	case TypeWorkspaceTemplate:
		return "workspace_templates"
	case TypeWorkspaceTemplateVersion:
		return "workspace_template_versions"
	case TypeWorkspace:
		return "workspaces"
	}

	return ""
}

type Namespace struct {
	Name   string
	Labels map[string]string
}

type Secret struct {
	Name string
	Data map[string]string
}

type ConfigMap struct {
	Name string
	Data map[string]string
}

type LogEntry struct {
	Timestamp time.Time
	Content   string
}

// IsEmpty returns true if the content for the log entry is just an empty string
func (l *LogEntry) IsEmpty() bool {
	return l.Content == ""
}

// LogEntryFromLine creates a LogEntry given a line of text
// it tries to parse out a timestamp and content
func LogEntryFromLine(line *string) *LogEntry {
	if line == nil {
		return nil
	}

	if *line == "" {
		return &LogEntry{Content: ""}
	}

	parts := strings.Split(*line, " ")
	if len(parts) == 0 {
		return nil
	}

	timestamp, err := time.Parse(time.RFC3339, parts[0])
	if err != nil {
		return &LogEntry{Content: *line}
	}

	return &LogEntry{
		Timestamp: timestamp,
		Content:   strings.Join(parts[1:], " "),
	}
}

type Metric struct {
	Name   string
	Value  float64
	Format string `json:"omitempty"`
}

type WorkflowExecutionStatisticReport struct {
	WorkflowTemplateId uint64 `db:"workflow_template_id"`
	Total              int32
	LastExecuted       *time.Time `db:"last_executed"`
	Running            int32
	Completed          int32
	Failed             int32
	Terminated         int32
}

// WorkspaceStatisticReport contains stats on the phases the workspaces in the system are in
type WorkspaceStatisticReport struct {
	LastCreated       *time.Time `db:"last_created"`
	Launching         int32
	Running           int32
	Updating          int32
	Pausing           int32
	Paused            int32
	Terminating       int32
	Terminated        int32
	FailedToPause     int32 `db:"failed_to_pause" json:"failedToPause"`
	FailedToResume    int32 `db:"failed_to_resume" json:"failedToResume"`
	FailedToTerminate int32 `db:"failed_to_terminate" json:"failedToTerminate"`
	FailedToLaunch    int32 `db:"failed_to_launch" json:"failedToLaunch"`
	FailedToUpdate    int32 `db:"failed_to_update" json:"failedToUpdate"`
	Failed            int32
	Total             int32
}

type CronWorkflowStatisticReport struct {
	WorkflowTemplateId uint64 `db:"workflow_template_id"`
	Total              int32
}

type ListOptions = metav1.ListOptions

type PodGCStrategy = wfv1.PodGCStrategy

func WorkflowTemplatesToIds(workflowTemplates []*WorkflowTemplate) (ids []uint64) {
	mappedIds := make(map[uint64]bool)

	// This is to make sure we don't have duplicates
	for _, workflowTemplate := range workflowTemplates {
		mappedIds[workflowTemplate.ID] = true
	}

	for id := range mappedIds {
		ids = append(ids, id)
	}

	return
}

// WorkflowTemplatesToVersionIDs picks out the WorkflowTemplateVersionID from each template and returns
// it as an array. Duplicates are removed.
func WorkflowTemplatesToVersionIDs(workflowTemplates []*WorkflowTemplate) (ids []uint64) {
	mappedIds := make(map[uint64]bool)

	// This is to make sure we don't have duplicates
	for _, workflowTemplate := range workflowTemplates {
		mappedIds[workflowTemplate.WorkflowTemplateVersionID] = true
	}

	for id := range mappedIds {
		ids = append(ids, id)
	}

	return
}

// Metrics is a convenience type to work with multiple Metric(s)
type Metrics []*Metric

// Add adds the new metric to the metrics.
// If there is already metrics with the same name, and override is true
// the existing metrics will all be updated to the input value. Otherwise, they will be left unchanged.
func (m *Metrics) Add(input *Metric, override bool) {
	foundExisting := false

	for _, metric := range *m {
		if metric.Name == input.Name && override {
			foundExisting = true

			metric.Value = input.Value
			metric.Format = input.Format
		}
	}

	if !foundExisting {
		*m = append(*m, input)
	}
}

// Merge merges the metrics with other metrics
// If there is already metrics with the same name and override is true
// the existing metrics will all be updated to the input value. Otherwise they will be left unchanged.
func (m *Metrics) Merge(input Metrics, override bool) {
	for _, item := range input {
		m.Add(item, override)
	}
}

// Unmarshal unmarshal's the json in m to v, as in json.Unmarshal.
// This is to support Metrics working with JSONB column types in sql
func (m *Metrics) Unmarshal(v interface{}) error {
	if len(*m) == 0 {
		*m = make([]*Metric, 0)
	}

	v = m

	return nil
}

// Value returns j as a value.  This does a validating unmarshal into another
// RawMessage.  If j is invalid json, it returns an error.
// Note that nil values will return "[]" - empty JSON.
// This is to support Metrics working with JSONB column types in sql
func (m Metrics) Value() (driver.Value, error) {
	if m == nil {
		return json.Marshal(make([]*Metric, 0))
	}

	return json.Marshal(m)
}

// Scan stores the src in m.  No validation is done.
// This is to support Metrics working with JSONB column types in sql
func (m *Metrics) Scan(src interface{}) error {
	var source []byte
	switch t := src.(type) {
	case string:
		source = []byte(t)
	case []byte:
		if len(t) == 0 {
			source = []byte("[]")
		} else {
			source = t
		}
	case nil:
		*m = make([]*Metric, 0)
	default:
		return errors.New("incompatible type for Metrics")
	}

	return json.Unmarshal(source, m)
}
