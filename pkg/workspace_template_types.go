package v1

import (
	"time"
)

type WorkspaceTemplate struct {
	ID                 uint64
	UID                string
	CreatedAt          time.Time  `db:"created_at"`
	ModifiedAt         *time.Time `db:"modified_at"`
	IsArchived         string     `db:"is_archived"`
	Name               string     `valid:"stringlength(3|30)~Name should be between 3 to 30 characters,required"`
	Namespace          string
	Version            int64
	Manifest           string
	IsLatest           bool
	WorkflowTemplate   *WorkflowTemplate `db:"workflow_template"`
	Labels             map[string]string
	WorkflowTemplateID uint64 `db:"workflow_template_id"`
}
