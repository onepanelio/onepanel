package v1

import (
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	networking "istio.io/api/networking/v1alpha3"
	corev1 "k8s.io/api/core/v1"
	"time"
)

type WorkspacePhase string

// Workspace phases
const (
	WorkspaceLaunching   WorkspacePhase = "Launching"
	WorkspaceRunning     WorkspacePhase = "Running"
	WorkspaceUpdating    WorkspacePhase = "Updating"
	WorkspacePausing     WorkspacePhase = "Pausing"
	WorkspacePaused      WorkspacePhase = "Paused"
	WorkspaceTerminating WorkspacePhase = "Terminating"
	WorkspaceTerminated  WorkspacePhase = "Terminated"
)

type WorkspaceStatus struct {
	Phase        WorkspacePhase `db:"phase"`
	StartedAt    *time.Time     `db:"started_at"`
	PausedAt     *time.Time     `db:"paused_at"`
	TerminatedAt *time.Time     `db:"terminated_at"`
	UpdatedAt    *time.Time     `db:"updated_at"`
}

type Workspace struct {
	ID                       uint64
	Namespace                string
	UID                      string `valid:"stringlength(3|30)~UID should be between 3 to 30 characters,dns,required"`
	Name                     string `valid:"stringlength(3|30)~Name should be between 3 to 30 characters,required"`
	Labels                   map[string]string
	Parameters               []Parameter
	ParametersBytes          []byte                   `db:"parameters"` // to load from database
	Status                   WorkspaceStatus          `db:"status"`
	CreatedAt                time.Time                `db:"created_at"`
	ModifiedAt               *time.Time               `db:"modified_at"`
	WorkspaceTemplate        *WorkspaceTemplate       `db:"workspace_template" valid:"-"`
	WorkspaceTemplateID      uint64                   `db:"workspace_template_id"`
	WorkspaceTemplateVersion uint64                   `db:"workspace_template_version"`
	URL                      string                   `db:"url"`                       // the path to the workspace, a url that you can access via http
	WorkflowTemplateVersion  *WorkflowTemplateVersion `db:"workflow_template_version"` // helper to store data from workflow template version
}

type WorkspaceSpec struct {
	Arguments             *Arguments                     `json:"arguments" protobuf:"bytes,1,opt,name=arguments"`
	Containers            []corev1.Container             `json:"containers" protobuf:"bytes,3,opt,name=containers"`
	Ports                 []corev1.ServicePort           `json:"ports" protobuf:"bytes,4,opt,name=ports"`
	Routes                []*networking.HTTPRoute        `json:"routes" protobuf:"bytes,5,opt,name=routes"`
	VolumeClaims          []corev1.PersistentVolumeClaim `json:"volumeClaims" protobuf:"bytes,6,opt,name=volumeClaims"`
	PostExecutionWorkflow *wfv1.WorkflowTemplateSpec     `json:"postExecutionWorkflow" protobuf:"bytes,7,opt,name=postExecutionWorkflow"`
}

// returns all of the columns for workspace modified by alias, destination.
// see formatColumnSelect
func getWorkspaceColumns(alias string, destination string, extraColumns ...string) []string {
	columns := []string{"id", "created_at", "modified_at", "uid", "name", "namespace", "parameters", "workspace_template_id", "workspace_template_version", "url"}
	return formatColumnSelect(columns, alias, destination, extraColumns...)
}

// returns all of the columns for WorkspaceStatus modified by alias, destination.
// see formatColumnSelect
func getWorkspaceStatusColumns(alias string, destination string, extraColumns ...string) []string {
	columns := []string{"phase", "started_at", "paused_at", "terminated_at"}
	return formatColumnSelect(columns, alias, destination, extraColumns...)
}
