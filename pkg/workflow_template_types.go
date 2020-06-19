package v1

import (
	"encoding/json"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/onepanelio/core/pkg/util/mapping"
	"github.com/onepanelio/core/util/sql"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

type WorkflowTemplate struct {
	ID                               uint64
	CreatedAt                        time.Time  `db:"created_at"`
	ModifiedAt                       *time.Time `db:"modified_at"`
	UID                              string
	Namespace                        string
	Name                             string
	Manifest                         string
	Version                          int64 // The latest version, unix timestamp
	Versions                         int64 `db:"versions"` // How many versions there are of this template total.
	IsLatest                         bool
	IsArchived                       bool `db:"is_archived"`
	IsSystem                         bool `db:"is_system"`
	ArgoWorkflowTemplate             *wfv1.WorkflowTemplate
	Labels                           map[string]string
	WorkflowExecutionStatisticReport *WorkflowExecutionStatisticReport
	CronWorkflowsStatisticsReport    *CronWorkflowStatisticReport
	// todo rename to have ID suffix
	WorkflowTemplateVersionId uint64  `db:"workflow_template_version_id"` // Reference to the associated workflow template version.
	Resource                  *string // utility in case we are specifying a workflow template for a specific resource
	ResourceUID               *string // see Resource field
}

func (wt *WorkflowTemplate) GetManifestBytes() []byte {
	return []byte(wt.Manifest)
}

func (wt *WorkflowTemplate) GetParametersKeyString() (map[string]string, error) {
	root := make(map[interface{}]interface{})

	if err := yaml.Unmarshal(wt.GetManifestBytes(), root); err != nil {
		return nil, err
	}

	arguments, ok := root["arguments"]
	if !ok {
		return nil, nil
	}

	argumentsMap, ok := arguments.(map[interface{}]interface{})
	if !ok {
		return nil, nil
	}

	parameters, ok := argumentsMap["parameters"]
	if !ok {
		return nil, nil
	}

	parametersAsArray, ok := parameters.([]interface{})
	if !ok {
		return nil, nil
	}

	if len(parametersAsArray) == 0 {
		delete(root, arguments)
	}

	result := make(map[string]string)
	for index, parameter := range parametersAsArray {
		parameterMap, ok := parameter.(map[interface{}]interface{})
		if !ok {
			continue
		}

		key := parameterMap["name"]
		keyAsString, ok := key.(string)
		if !ok {
			continue
		}

		parameterMap["order"] = index
		remainingParameters, err := yaml.Marshal(parameterMap)
		if err != nil {
			continue
		}

		result[keyAsString] = string(remainingParameters)
	}

	return result, nil
}

func (wt *WorkflowTemplate) UpdateManifestParameters(params []Parameter) error {
	manifestMap, err := mapping.NewFromYamlString(wt.Manifest)
	if err != nil {
		return err
	}

	arguments, err := manifestMap.GetChildMap("arguments")
	if err != nil {
		return err
	}

	arguments["parameters"] = params

	manifestBytes, err := manifestMap.ToYamlBytes()
	if err != nil {
		return err
	}

	wt.Manifest = string(manifestBytes)

	return nil
}

func (wt *WorkflowTemplate) GetWorkflowManifestBytes() ([]byte, error) {
	if wt.ArgoWorkflowTemplate == nil {
		return []byte{}, nil
	}

	wt.ArgoWorkflowTemplate.TypeMeta.Kind = "Workflow"
	wt.ArgoWorkflowTemplate.ObjectMeta = metav1.ObjectMeta{
		GenerateName: wt.ArgoWorkflowTemplate.ObjectMeta.GenerateName,
		Labels:       wt.ArgoWorkflowTemplate.ObjectMeta.Labels,
	}

	return json.Marshal(wt.ArgoWorkflowTemplate)
}

func (wt *WorkflowTemplate) FormatManifest() (string, error) {
	manifestMap, err := mapping.NewFromYamlString(wt.Manifest)
	if err != nil {
		log.WithFields(log.Fields{
			"Method": "FormatManifest",
			"Step":   "NewFromYamlString",
			"Error":  err.Error(),
		}).Error("FormatManifest Workflow Template failed.")

		return "", nil
	}

	manifestMap, err = manifestMap.GetChildMap("spec")
	if err != nil {
		log.WithFields(log.Fields{
			"Method": "FormatManifest",
			"Step":   "GetChildMap",
			"Error":  err.Error(),
		}).Error("GetChildMap Workflow Template failed.")

		return "", nil
	}
	manifestMap.PruneEmpty()

	wt.AddWorkflowTemplateParametersFromAnnotations(manifestMap)

	manifestBytes, err := manifestMap.ToYamlBytes()
	if err != nil {
		log.WithFields(log.Fields{
			"Method": "FormatManifest",
			"Step":   "ToYamlBytes",
			"Error":  err.Error(),
		}).Error("ToYamlBytes Workflow Template failed.")
	}

	return string(manifestBytes), nil
}

// Take the manifest from the workflow template, which is just the "spec" contents
// and wrap it so we have
// {
//    metadata: {},
//    spec: spec_data
// }
// the above wrapping is what is returned.
func (wt *WorkflowTemplate) WrapSpec() ([]byte, error) {
	data := wt.GetManifestBytes()

	spec := make(map[interface{}]interface{})

	if err := yaml.Unmarshal(data, spec); err != nil {
		return nil, err
	}

	contentMap := map[interface{}]interface{}{
		"metadata": make(map[interface{}]interface{}),
		"spec":     spec,
	}

	finalBytes, err := yaml.Marshal(contentMap)
	if err != nil {
		return nil, nil
	}

	return finalBytes, nil
}

func (wt *WorkflowTemplate) AddWorkflowTemplateParametersFromAnnotations(spec mapping.Mapping) {
	if wt.ArgoWorkflowTemplate == nil {
		return
	}

	annotations := wt.ArgoWorkflowTemplate.Annotations
	if spec == nil || len(annotations) == 0 {
		return
	}

	arguments, err := spec.GetChildMap("arguments")
	if err != nil {
		return
	}

	arguments["parameters"] = make([]interface{}, 0)
	parameters := make([]interface{}, len(annotations))

	for _, value := range annotations {
		data, err := mapping.NewFromYamlString(value)
		if err != nil {
			log.WithFields(log.Fields{
				"Method": "AddWorkflowTemplateParametersFromAnnotations",
				"Step":   "NewFromYamlString",
				"Error":  err.Error(),
			}).Error("Error with AddWorkflowTemplateParametersFromAnnotations")
			continue
		}

		order := 0
		orderValue, ok := data["order"]
		if ok {
			order = orderValue.(int)
			delete(data, "order")

			if order >= 0 && order < len(parameters) {
				parameters[order] = data
			}
		}
	}

	arguments["parameters"] = parameters
}

// getWorkflowTemplateColumns returns all of the columns for workflowTemplate modified by alias, destination.
// see formatColumnSelect
func getWorkflowTemplateColumns(aliasAndDestination ...string) []string {
	columns := []string{"id", "created_at", "uid", "name", "namespace", "modified_at", "is_archived"}
	return sql.FormatColumnSelect(columns, aliasAndDestination...)
}
