package v1

import (
	"fmt"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/onepanelio/core/pkg/util/ptr"
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

// InjectRuntimeVariables will inject all runtime variables into the WorkflowTemplate's manifest.
func (wt *WorkspaceTemplate) InjectRuntimeVariables(config map[string]string) error {
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

// RuntimeVars contains data that is needed for workspaces to function at runtime.
// This includes data that is configuration dependent, which may change.
// For example, the sys-host might change because it depends on the ONEPANEL_DOMAIN config variable
type RuntimeVars struct {
	AdditionalParameters []*Parameter
	VirtualService       *wfv1.Template
	WorkspaceSpec        *WorkspaceSpec
	StatefulSetManifest  string
}

// TODO document this and then maybe add separate functions to inject it
func (wt *WorkspaceTemplate) RuntimeVars(config map[string]string) (runtimeVars *RuntimeVars, err error) {
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

	arguments, ok := parsedManifest["arguments"]
	if !ok {
		err = fmt.Errorf("argumnets not found in workflow template manifest")
		return
	}

	argumentsMap := arguments.(map[interface{}]interface{})
	localParameters := argumentsMap["parameters"]
	parametersArray := localParameters.([]interface{})

	// sys-host
	parametersArray = append(parametersArray, Parameter{
		Name:  "sys-host",
		Value: ptr.String(config["ONEPANEL_DOMAIN"]),
		Type:  "input.hidden",
	})

	// Node pool parameter and options
	var options []*ParameterOption
	if err := yaml.Unmarshal([]byte(config["applicationNodePoolOptions"]), &options); err != nil {
		return nil, err
	}

	runtimeVars.AdditionalParameters = append(runtimeVars.AdditionalParameters, &Parameter{
		Name:        "sys-node-pool",
		Value:       ptr.String(options[0].Value),
		Type:        "select.select",
		Options:     options,
		DisplayName: ptr.String("Node pool"),
		Hint:        ptr.String("Name of node pool or group"),
		Required:    true,
	})

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
