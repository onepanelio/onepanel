package v1

import (
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
