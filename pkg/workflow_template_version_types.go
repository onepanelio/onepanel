package v1

import (
	"encoding/json"
	"time"

	"github.com/onepanelio/core/pkg/util/sql"
	"github.com/onepanelio/core/pkg/util/types"
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
	Labels           types.JSONLabels
	Parameters       []Parameter
	ParametersBytes  []byte `db:"parameters"` // to load from database
	Description      string `db:"description"`
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

// LoadParametersFromBytes loads Parameters from the WorkflowTemplateVersion's ParameterBytes field.
func (wtv *WorkflowTemplateVersion) LoadParametersFromBytes() ([]Parameter, error) {
	loadedParameters := make([]Parameter, 0)

	err := json.Unmarshal(wtv.ParametersBytes, &loadedParameters)
	if err != nil {
		return wtv.Parameters, err
	}

	// It might be nil because the value "null" is stored in db if there are no parameters.
	// for consistency, we return an empty array.
	if loadedParameters == nil {
		loadedParameters = make([]Parameter, 0)
	}

	wtv.Parameters = loadedParameters

	return wtv.Parameters, err
}

// getWorkflowTemplateVersionColumns returns all of the columns for workflow template versions modified by alias, destination.
// see formatColumnSelect
func getWorkflowTemplateVersionColumns(aliasAndDestination ...string) []string {
	columns := []string{"id", "created_at", "version", "is_latest", "manifest", "parameters", "labels", "description"}
	return sql.FormatColumnSelect(columns, aliasAndDestination...)
}
