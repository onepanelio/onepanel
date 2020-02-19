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
	Data map[string][]byte
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
	WorfklowPending   WorkflowPhase = "Pending"
	WorfklowRunning   WorkflowPhase = "Running"
	WorfklowSucceeded WorkflowPhase = "Succeeded"
	WorfklowSkipped   WorkflowPhase = "Skipped"
	WorfklowFailed    WorkflowPhase = "Failed"
	WorfklowError     WorkflowPhase = "Error"
)

type WorkflowPhase string

type Workflow struct {
	ID               uint64
	CreatedAt        time.Time `db:"created_at"`
	UID              string
	Name             string
	GenerateName     string
	Parameters       []WorkflowParameter
	Manifest         string
	Phase            WorkflowPhase
	StartedAt        time.Time
	FinishedAt       time.Time
	WorkflowTemplate *WorkflowTemplate
}

type WorkflowParameter struct {
	Name  string
	Value *string
}

type ListOptions = metav1.ListOptions

type PodGCStrategy = wfv1.PodGCStrategy

type WorkflowOptions struct {
	Name           string
	GenerateName   string
	Entrypoint     string
	Parameters     []WorkflowParameter
	ServiceAccount string
	Labels         *map[string]string
	ListOptions    *ListOptions
	PodGCStrategy  *PodGCStrategy
}
