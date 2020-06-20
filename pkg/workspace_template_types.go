package v1

import (
	"fmt"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/onepanelio/core/util/sql"
	"gopkg.in/yaml.v2"
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

// RuntimeVars contains data that is needed for workspaces to function at runtime.
// This includes data that is configuration dependent, which may change.
// For example, the sys-host might change because it depends on the ONEPANEL_DOMAIN config variable
type RuntimeVars struct {
	AdditionalParameters []*Parameter
	VirtualService       *wfv1.Template
	WorkspaceSpec        *WorkspaceSpec
	StatefulSetManifest  string
}

// RuntimeVars returns a set of RuntimeVars associated with the WorkspaceTemplate.
// These require a config loaded from the system
func (wt *WorkspaceTemplate) RuntimeVars(config SystemConfig) (runtimeVars *RuntimeVars, err error) {
	if wt.WorkflowTemplate == nil {
		err = fmt.Errorf("workflow Template is nil for workspace template")
		return
	}

	runtimeVars = &RuntimeVars{}
	workspaceSpec, err := parseWorkspaceSpec(wt.Manifest)
	if err != nil {
		return
	}

	runtimeVars.WorkspaceSpec = workspaceSpec
	parsedManifest := make(map[string]interface{})
	if err := yaml.Unmarshal([]byte(wt.WorkflowTemplate.Manifest), parsedManifest); err != nil {
		return nil, err
	}

	runtimeParameters, err := generateRuntimeParamters(config)
	if err != nil {
		return nil, err
	}
	for i := range runtimeParameters {
		parameter := &runtimeParameters[i]
		runtimeVars.AdditionalParameters = append(runtimeVars.AdditionalParameters, parameter)
	}

	vs, err := createVirtualServiceManifest(workspaceSpec, true)
	if err != nil {
		return nil, err
	}

	runtimeVars.VirtualService = &wfv1.Template{
		Name: "virtual-service-resource",
		Resource: &wfv1.ResourceTemplate{
			Action:   "{{workflow.parameters.sys-resource-action}}",
			Manifest: vs,
		},
	}

	statefulSet, err := createStatefulSetManifest(workspaceSpec, config, true)
	if err != nil {
		return nil, err
	}
	runtimeVars.StatefulSetManifest = statefulSet

	return
}

// InjectRuntimeVariables will inject all runtime variables into the WorkflowTemplate's manifest.
func (wt *WorkspaceTemplate) InjectRuntimeVariables(config SystemConfig) error {
	if wt.WorkflowTemplate == nil {
		return fmt.Errorf("workflow Template is nil for workspace template")
	}

	runtimeVars, err := wt.RuntimeVars(config)
	if err != nil {
		return err
	}

	parsedManifest := make(map[string]interface{})
	if err := yaml.Unmarshal([]byte(wt.WorkflowTemplate.Manifest), parsedManifest); err != nil {
		return err
	}

	arguments, ok := parsedManifest["arguments"]
	if !ok {
		return fmt.Errorf("argumnets not found in workflow template manifest")
	}

	argumentsMap := arguments.(map[interface{}]interface{})
	parameters := argumentsMap["parameters"]
	parametersArray := parameters.([]interface{})

	for _, param := range runtimeVars.AdditionalParameters {
		parametersArray = append(parametersArray, param)
	}
	argumentsMap["parameters"] = parametersArray

	templates := parsedManifest["templates"].([]interface{})
	finalTemplates := make([]interface{}, 0)
	for _, t := range templates {
		template := t.(map[interface{}]interface{})
		name, ok := template["name"]
		if !ok {
			continue
		}

		if name == runtimeVars.VirtualService.Name {
			continue
		}

		if name == "stateful-set-resource" {
			resource := template["resource"]
			resourceMap := resource.(map[interface{}]interface{})
			resourceMap["manifest"] = runtimeVars.StatefulSetManifest
		}

		finalTemplates = append(finalTemplates, template)
	}
	finalTemplates = append(finalTemplates, runtimeVars.VirtualService)
	parsedManifest["templates"] = finalTemplates

	resultManifest, err := yaml.Marshal(parsedManifest)
	if err != nil {
		return err
	}

	wt.WorkflowTemplate.Manifest = string(resultManifest)

	return nil
}

func WorkspaceTemplatesToVersionIDs(resources []*WorkspaceTemplate) (ids []uint64) {
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

// getWorkspaceTemplateColumns returns all of the columns for workspace template modified by alias, destination.
// see formatColumnSelect
func getWorkspaceTemplateColumns(aliasAndDestination ...string) []string {
	columns := []string{"id", "uid", "created_at", "modified_at", "name", "namespace", "is_archived", "workflow_template_id"}
	return sql.FormatColumnSelect(columns, aliasAndDestination...)
}
