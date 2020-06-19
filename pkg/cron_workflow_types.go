package v1

import (
	"encoding/json"
	"github.com/onepanelio/core/pkg/util/mapping"
	"github.com/onepanelio/core/util/sql"
	"gopkg.in/yaml.v2"
	"time"
)

type CronWorkflow struct {
	ID                        uint64
	CreatedAt                 time.Time  `db:"created_at"`
	ModifiedAt                *time.Time `db:"modified_at"`
	UID                       string
	Name                      string
	GenerateName              string
	WorkflowExecution         *WorkflowExecution
	Labels                    map[string]string
	Version                   int64
	WorkflowTemplateVersionId uint64 `db:"workflow_template_version_id"`
	Manifest                  string
	Namespace                 string `db:"namespace"`
}

func (cw *CronWorkflow) GetParametersFromWorkflowSpec() ([]Parameter, error) {
	var parameters []Parameter

	mappedData := make(map[string]interface{})

	if err := yaml.Unmarshal([]byte(cw.Manifest), mappedData); err != nil {
		return nil, err
	}

	workflowSpec, ok := mappedData["workflowSpec"]
	if !ok {
		return parameters, nil
	}

	workflowSpecMap := workflowSpec.(map[interface{}]interface{})
	arguments, ok := workflowSpecMap["arguments"]
	if !ok {
		return parameters, nil
	}

	argumentsMap := arguments.(map[interface{}]interface{})
	parametersRaw, ok := argumentsMap["parameters"]
	if !ok {
		return parameters, nil
	}

	parametersArray, ok := parametersRaw.([]interface{})
	for _, parameter := range parametersArray {
		paramMap, ok := parameter.(map[interface{}]interface{})
		if !ok {
			continue
		}

		workflowParameter := ParameterFromMap(paramMap)

		parameters = append(parameters, *workflowParameter)
	}

	return parameters, nil
}

func (cw *CronWorkflow) GetParametersFromWorkflowSpecJson() ([]byte, error) {
	parameters, err := cw.GetParametersFromWorkflowSpec()
	if err != nil {
		return nil, err
	}

	parametersJson, err := json.Marshal(parameters)
	if err != nil {
		return nil, err
	}

	return parametersJson, nil
}

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
	columns := []string{"cw.id", "cw.created_at", "cw.uid", "cw.name", "cw.workflow_template_version_id", "cw.manifest", "cw.namespace"}
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
