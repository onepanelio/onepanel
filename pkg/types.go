package v1

import (
	"time"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/google/uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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

type WorkflowTemplate struct {
	ID         uint64
	CreatedAt  time.Time `db:"created_at"`
	UID        string
	Name       string
	Manifest   string
	Version    int32
	IsLatest   bool `db:"is_latest"`
	IsArchived bool `db:"is_archived"`
}

func (wt *WorkflowTemplate) GetManifestBytes() []byte {
	return []byte(wt.Manifest)
}

func (wt *WorkflowTemplate) GenerateUID() (string, error) {
	uid, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}
	wt.UID = uid.String()

	return wt.UID, nil
}

const (
	WorfklowPending   WorkflowExecutionPhase = "Pending"
	WorfklowRunning   WorkflowExecutionPhase = "Running"
	WorfklowSucceeded WorkflowExecutionPhase = "Succeeded"
	WorfklowSkipped   WorkflowExecutionPhase = "Skipped"
	WorfklowFailed    WorkflowExecutionPhase = "Failed"
	WorfklowError     WorkflowExecutionPhase = "Error"
)

type WorkflowExecutionPhase string

type WorkflowExecution struct {
	ID               uint64
	CreatedAt        time.Time `db:"created_at"`
	UID              string
	Name             string
	GenerateName     string
	Parameters       []WorkflowExecutionParameter
	Manifest         string
	Phase            WorkflowExecutionPhase
	StartedAt        time.Time
	FinishedAt       time.Time
	WorkflowTemplate *WorkflowTemplate
}

type WorkflowExecutionParameter struct {
	Name  string
	Value *string
}

type ListOptions = metav1.ListOptions

type PodGCStrategy = wfv1.PodGCStrategy

type WorkflowExecutionOptions struct {
	Name           string
	GenerateName   string
	Entrypoint     string
	Parameters     []WorkflowExecutionParameter
	ServiceAccount string
	Labels         *map[string]string
	ListOptions    *ListOptions
	PodGCStrategy  *PodGCStrategy
}
