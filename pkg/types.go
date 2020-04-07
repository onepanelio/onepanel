package v1

import (
	"gopkg.in/yaml.v2"
	"strings"
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

type CronWorkflow struct {
	ID                         uint64
	CreatedAt                  time.Time `db:"created_at"`
	UID                        string
	Name                       string
	GenerateName               string
	Schedule                   string
	Timezone                   string
	Suspend                    bool
	ConcurrencyPolicy          string
	StartingDeadlineSeconds    *int64
	SuccessfulJobsHistoryLimit *int32
	FailedJobsHistoryLimit     *int32
	WorkflowExecution          *WorkflowExecution
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

type File struct {
	Path         string
	Name         string
	Size         int64
	Extension    string
	ContentType  string
	LastModified time.Time
	Directory    bool
}

func FilePathToName(path string) string {
	if strings.HasSuffix(path, "/") {
		path = path[:len(path)-1]
	}

	lastSlashIndex := strings.LastIndex(path, "/")
	if lastSlashIndex < 0 {
		return path
	}

	return path[lastSlashIndex+1:]
}

// Given a path, returns the parent path, asssuming a '/' delimitor
// Result does not have a trailing slash.
// -> a/b/c/d would return a/b/c
// -> a/b/c/d/ would return a/b/c
// If path is empty string, it is returned.
// If path is '/' (root) it is returned as is.
// If there is no '/', '/' is returned.
func FilePathToParentPath(path string) string {
	separator := "/"
	if path == "" || path == separator {
		return path
	}

	if strings.HasSuffix(path, "/") {
		path = path[0 : len(path)-1]
	}

	lastIndexOfForwardSlash := strings.LastIndex(path, separator)
	if lastIndexOfForwardSlash <= 0 {
		return separator
	}

	return path[0:lastIndexOfForwardSlash]
}

func FilePathToExtension(path string) string {
	dotIndex := strings.LastIndex(path, ".")

	if dotIndex == -1 {
		return ""
	}

	if dotIndex == (len(path) - 1) {
		return ""
	}

	return path[dotIndex+1:]
}

func WrapSpecInK8s(data []byte) ([]byte, error) {
	mapping := make(map[interface{}]interface{})
	if err := yaml.Unmarshal(data, mapping); err != nil {
		return nil, err
	}

	contentMap := map[interface{}]interface{}{
		"metadata": make(map[interface{}]interface{}),
		"spec":     mapping,
	}

	finalBytes, err := yaml.Marshal(contentMap)
	if err != nil {
		return nil, nil
	}

	return finalBytes, nil
}
