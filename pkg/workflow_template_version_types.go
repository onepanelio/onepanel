package v1

import (
	"github.com/onepanelio/core/util/sql"
	"time"
)

// WorkflowTemplateVersion represents a different version of a WorkflowTemplate
// each version can have a different manifest and labels.
// This is used to version control the template
type WorkflowTemplateVersion struct {
	ID               uint64
	UID              string
	Version          int64
	IsLatest         bool `db:"is_latest"`
	Manifest         string
	CreatedAt        time.Time         `db:"created_at"`
	WorkflowTemplate *WorkflowTemplate `db:"workflow_template"`
	Labels           map[string]string
}

// WorkflowTemplateVersionsToIDs returns an array of ids from the input WorkflowTemplateVersion with no duplicates.
func WorkflowTemplateVersionsToIDs(resources []*WorkflowTemplateVersion) (ids []uint64) {
	mappedIds := make(map[uint64]bool)

	// This is to make sure we don't have duplicates
	for _, resource := range resources {
		mappedIds[resource.ID] = true
	}

	for id := range mappedIds {
		ids = append(ids, id)
	}

	return
}

// getWorkflowTemplateVersionColumns returns all of the columns for workflow template versions modified by alias, destination.
// see formatColumnSelect
func getWorkflowTemplateVersionColumns(aliasAndDestination ...string) []string {
	columns := []string{"id", "created_at", "version", "is_latest", "manifest"}
	return sql.FormatColumnSelect(columns, aliasAndDestination...)
}
