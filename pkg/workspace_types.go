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
	WorkspaceStarted     WorkspacePhase = "Started"
	WorkspaceRunning     WorkspacePhase = "Running"
	WorkspacePausing     WorkspacePhase = "Pausing"
	WorkspacePaused      WorkspacePhase = "Paused"
	WorkspaceTerminating WorkspacePhase = "Terminating"
	WorkspaceTerminated  WorkspacePhase = "Terminated"
)

type WorkspaceStatus struct {
	Phase        WorkspacePhase
	StartedAt    *time.Time `db:"started_at"`
	PausedAt     *time.Time `db:"paused_at"`
	TerminatedAt *time.Time `db:"terminated_at"`
}

type Workspace struct {
	ID                uint64
	UID               string
	Name              string `valid:"stringlength(3|63)~Name should be between 3 to 63 characters,dns,required"`
	Labels            map[string]string
	Parameters        []Parameter
	Status            WorkspaceStatus
	CreatedAt         time.Time          `db:"created_at"`
	ModifiedAt        time.Time          `db:"modified_at"`
	WorkspaceTemplate *WorkspaceTemplate `db:"workspace_template" valid:"-"`
}

type WorkspaceSpec struct {
	Arguments             *Arguments                 `json:"arguments" protobuf:"bytes,1,opt,name=arguments"`
	Containers            []corev1.Container         `json:"containers" protobuf:"bytes,3,opt,name=containers"`
	Ports                 []corev1.ServicePort       `json:"ports" protobuf:"bytes,4,opt,name=ports"`
	Routes                []*networking.HTTPRoute    `json:"routes" protobuf:"bytes,5,opt,name=routes"`
	PostExecutionWorkflow *wfv1.WorkflowTemplateSpec `json:"postExecutionWorkflow" protobuf:"bytes,6,opt,name=postExecutionWorkflow"`
}
