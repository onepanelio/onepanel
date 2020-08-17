package v1

import (
	"encoding/json"
	"github.com/onepanelio/core/pkg/util/mapping"
	"github.com/onepanelio/core/pkg/util/sql"
	"github.com/onepanelio/core/pkg/util/types"
	"gopkg.in/yaml.v2"
	"time"
)

// CronWorkflow represents a workflow that runs on a cron.
type CronWorkflow struct {
	ID                        uint64
	CreatedAt                 time.Time  `db:"created_at"`
	ModifiedAt                *time.Time `db:"modified_at"`
	UID                       string
	Name                      string
	GenerateName              string
	WorkflowExecution         *WorkflowExecution
	Labels                    types.JSONLabels
	Version                   int64
	WorkflowTemplateVersionID uint64 `db:"workflow_template_version_id"`
	Manifest                  string
	Namespace                 string `db:"namespace"`
}

// CronWorkflowManifest is a client representation of a CronWorkflowManifest
// It is usually provided as YAML by a client and this struct helps to marshal/unmarshal it
type CronWorkflowManifest struct {
	WorkflowExecutionSpec WorkflowExecutionSpec `json:"workflowSpec" yaml:"workflowSpec"`
}

// GetParametersFromWorkflowSpec parses the parameters from the CronWorkflow's manifest
func (cw *CronWorkflow) GetParametersFromWorkflowSpec() ([]Parameter, error) {
	manifestSpec := &CronWorkflowManifest{}

	if err := yaml.Unmarshal([]byte(cw.Manifest), manifestSpec); err != nil {
		return nil, err
	}

	parameters := manifestSpec.WorkflowExecutionSpec.Arguments.Parameters

	return parameters, nil
}

// GetParametersFromWorkflowSpecJSON parses the parameters from the CronWorkflow's manifest and returns them as a JSON string
func (cw *CronWorkflow) GetParametersFromWorkflowSpecJSON() ([]byte, error) {
	parameters, err := cw.GetParametersFromWorkflowSpec()
	if err != nil {
		return nil, err
	}

	parametersJSON, err := json.Marshal(parameters)
	if err != nil {
		return nil, err
	}

	return parametersJSON, nil
}

// AddToManifestSpec updates the CronWorkflow's manifest by setting the input manifest under the specified key
func (cw *CronWorkflow) AddToManifestSpec(key, manifest string) error {
	currentManifestMapping, err := mapping.NewFromYamlString(cw.Manifest)
	if err != nil {
		return err
	}

	additionalManifest, err := mapping.NewFromYamlString(manifest)
	if err != nil {
		return err
	}

	currentManifestMapping[key] = additionalManifest

	updatedManifest, err := currentManifestMapping.ToYamlBytes()
	if err != nil {
		return err
	}

	cw.Manifest = string(updatedManifest)

	return nil
}

// getCronWorkflowColumns returns all of the columns for cronWorkflow modified by alias, destination.
// see formatColumnSelect
func getCronWorkflowColumns(aliasAndDestination ...string) []string {
	columns := []string{"id", "created_at", "uid", "name", "workflow_template_version_id", "manifest", "namespace", "labels"}
	return sql.FormatColumnSelect(columns, aliasAndDestination...)
}

// CronWorkflowsToIDs returns an array of ids from the input CronWorkflow with no duplicates.
func CronWorkflowsToIDs(resources []*CronWorkflow) (ids []uint64) {
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
