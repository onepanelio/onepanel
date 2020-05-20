package v1

import (
	"time"
)

type WorkspaceTemplate struct {
	ID                         uint64
	WorkspaceTemplateVersionID uint64 `db:"workspace_template_version_id"`
	UID                        string
	CreatedAt                  time.Time  `db:"created_at"`
	ModifiedAt                 *time.Time `db:"modified_at"`
	IsArchived                 bool       `db:"is_archived"`
	Name                       string     `valid:"stringlength(3|30)~Name should be between 3 to 30 characters,required"`
	Namespace                  string
	Version                    int64
	Manifest                   string
	IsLatest                   bool
	WorkflowTemplate           *WorkflowTemplate `db:"workflow_template"`
	Labels                     map[string]string
	WorkflowTemplateID         uint64 `db:"workflow_template_id"`
}

func WorkspaceTemplatesToVersionIds(resources []*WorkspaceTemplate) (ids []uint64) {
	mappedIds := make(map[uint64]bool)

	// This is to make sure we don't have duplicates
	for _, resource := range resources {
		mappedIds[resource.WorkspaceTemplateVersionID] = true
	}

	for id := range mappedIds {
		ids = append(ids, id)
	}

	return
}

// returns all of the columns for workspace template modified by alias, destination.
// see formatColumnSelect
func getWorkspaceTemplateColumns(alias string, destination string, extraColumns ...string) []string {
	columns := []string{"id", "uid", "created_at", "modified_at", "name", "namespace", "is_archived", "workflow_template_id"}
	return formatColumnSelect(columns, alias, destination, extraColumns...)
}
