package v1

import (
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	networking "istio.io/api/networking/v1alpha3"
	corev1 "k8s.io/api/core/v1"
	"regexp"
	"strings"
	"time"
)

type WorkspacePhase string

// Workspace phases
const (
	WorkspaceStarted     WorkspacePhase = "Started"
	WorkspaceRunning     WorkspacePhase = "Running"
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
}

type Workspace struct {
	ID                       uint64
	Namespace                string
	UID                      string
	Name                     string `valid:"stringlength(3|30)~Name should be between 3 to 30 characters,dns,required"`
	Labels                   map[string]string
	Parameters               []Parameter
	ParametersBytes          []byte             `db:"parameters"` // to load from database
	Status                   WorkspaceStatus    `db:"status"`
	CreatedAt                time.Time          `db:"created_at"`
	ModifiedAt               *time.Time         `db:"modified_at"`
	WorkspaceTemplate        *WorkspaceTemplate `db:"workspace_template" valid:"-"`
	WorkspaceTemplateID      uint64             `db:"workspace_template_id"`
	WorkspaceTemplateVersion uint64             `db:"workspace_template_version"`
	Path                     string             `db:"path"` // the path to the workspace, a url that you can access via http
}

type WorkspaceSpec struct {
	Arguments             *Arguments                 `json:"arguments" protobuf:"bytes,1,opt,name=arguments"`
	Containers            []corev1.Container         `json:"containers" protobuf:"bytes,3,opt,name=containers"`
	Ports                 []corev1.ServicePort       `json:"ports" protobuf:"bytes,4,opt,name=ports"`
	Routes                []*networking.HTTPRoute    `json:"routes" protobuf:"bytes,5,opt,name=routes"`
	PostExecutionWorkflow *wfv1.WorkflowTemplateSpec `json:"postExecutionWorkflow" protobuf:"bytes,6,opt,name=postExecutionWorkflow"`
}

func (w *Workspace) GenerateUID() (string, error) {
	re, err := regexp.Compile(`[^a-zA-Z0-9-]{1,}`)
	if err != nil {
		return "", err
	}
	w.UID = strings.ToLower(re.ReplaceAllString(w.Name, `-`))

	return w.UID, nil
}

// returns all of the columns for WorkspaceStatus modified by alias, destination.
// see formatColumnSelect
func getWorkspaceStatusColumns(alias string, destination string, extraColumns ...string) []string {
	columns := []string{"phase", "started_at", "paused_at", "terminated_at"}
	return formatColumnSelect(columns, alias, destination, extraColumns...)
}
