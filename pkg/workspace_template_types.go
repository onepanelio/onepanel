package v1

import (
	"fmt"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/onepanelio/core/pkg/util/extensions"
	"github.com/onepanelio/core/pkg/util/sql"
	"github.com/onepanelio/core/pkg/util/types"
	uid2 "github.com/onepanelio/core/pkg/util/uid"
	yaml3 "gopkg.in/yaml.v3"
	"sigs.k8s.io/yaml"
	"strings"
	"time"
)

// WorkspaceService represents services available to external access in a Workspace
type WorkspaceService struct {
	Name string
	Path string
}

// WorkspaceTemplate represents the data associated with a WorkspaceTemplate
// this is a mix of DB and "in-memory" fields
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
	Labels                     types.JSONLabels
	WorkflowTemplateID         uint64 `db:"workflow_template_id"`
	Description                string
}

// GenerateUID generates a uid from the input name and sets it on the workflow template
func (wt *WorkspaceTemplate) GenerateUID(name string) error {
	result, err := uid2.GenerateUID(name, 30)
	if err != nil {
		return err
	}

	wt.UID = result

	return nil
}

// CreateWorkspaceTemplate creates a new workspace template with the given name.
// All fields that can be generated in memory without external requests are filled out, such as the UID.
func CreateWorkspaceTemplate(name string) (*WorkspaceTemplate, error) {
	nameUID, err := uid2.GenerateUID(name, 30)
	if err != nil {
		return nil, err
	}

	workspaceTemplate := &WorkspaceTemplate{
		Name: name,
		UID:  nameUID,
	}

	return workspaceTemplate, nil
}

// InjectRuntimeParameters will inject all runtime variables into the WorkflowTemplate's manifest.
func (wt *WorkspaceTemplate) InjectRuntimeParameters(config SystemConfig) error {
	if wt.WorkflowTemplate == nil {
		return fmt.Errorf("workflow Template is nil for workspace template")
	}

	manifest := struct {
		Arguments Arguments `json:"arguments"`
		wfv1.WorkflowSpec
	}{}
	if err := yaml.Unmarshal([]byte(wt.WorkflowTemplate.Manifest), &manifest); err != nil {
		return err
	}

	runtimeParameters, err := generateRuntimeParameters(config)
	if err != nil {
		return err
	}

	runtimeParametersMap := make(map[string]*string)
	for _, p := range runtimeParameters {
		runtimeParametersMap[p.Name] = p.Value
	}

	for i, p := range manifest.Arguments.Parameters {
		value := runtimeParametersMap[p.Name]
		if value != nil {
			manifest.Arguments.Parameters[i].Value = value
		}
	}

	resultManifest, err := yaml.Marshal(manifest)
	if err != nil {
		return err
	}
	wt.WorkflowTemplate.Manifest = string(resultManifest)

	return nil
}

// GetServices returns an array of WorkspaceServices
func (wt *WorkspaceTemplate) GetServices() ([]*WorkspaceService, error) {
	result := make([]*WorkspaceService, 0)

	root := &yaml3.Node{}
	if err := yaml3.Unmarshal([]byte(wt.Manifest), root); err != nil {
		return nil, err
	}

	containers, err := extensions.GetNode(root, extensions.CreateYamlIndex("containers"))
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return result, nil
		}

		return nil, err
	}

	for _, container := range containers.Content {
		hasKeyValue, err := extensions.HasKeyValue(container, "name", "sys-filesyncer")
		if err != nil {
			return nil, err
		}

		if hasKeyValue {
			argsValue, err := extensions.GetKeyValue(container, "args")
			if err != nil {
				continue
			}

			path := ""
			for _, arg := range argsValue.Content {
				if strings.Contains(arg.Value, "server-prefix") {
					parts := strings.Split(arg.Value, "=")
					if len(parts) > 1 {
						path = parts[1]
					}
				}
			}

			fmt.Printf("%v", argsValue)

			result = append(result, &WorkspaceService{
				Name: "sys-filesyncer",
				Path: path,
			})
		}
	}

	return result, nil
}

// WorkspaceTemplatesToVersionIDs plucks the WorkspaceTemplateVersionID from each template and returns it in an array
// No duplicates are included.
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
	columns := []string{"id", "uid", "created_at", "modified_at", "name", "description", "namespace", "is_archived", "workflow_template_id", "labels"}
	return sql.FormatColumnSelect(columns, aliasAndDestination...)
}

// getWorkspaceTemplateColumnsWithoutLabels returns all of the columns for workspace template, excluding labels, modified by alias, destination.
// see formatColumnSelect
func getWorkspaceTemplateColumnsWithoutLabels(aliasAndDestination ...string) []string {
	columns := []string{"id", "uid", "created_at", "modified_at", "name", "description", "namespace", "is_archived", "workflow_template_id"}
	return sql.FormatColumnSelect(columns, aliasAndDestination...)
}

// getWorkspaceTemplateColumnsMap returns a map where the keys are the columns of the workspace_templates table
// the value is the raw column name as it is in the database
func getWorkspaceTemplateColumnsMap(camelCase bool) map[string]string {
	result := map[string]string{
		"id":          "id",
		"labels":      "labels",
		"name":        "name",
		"uid":         "uid",
		"namespace":   "namespace",
		"description": "description",
	}

	if camelCase {
		result["createdAt"] = "created_at"
		result["modifiedAt"] = "modified_at"
		result["isArchived"] = "is_archived"
	} else {
		result["created_at"] = "created_at"
		result["modified_at"] = "modified_at"
		result["is_archived"] = "is_archived"
	}

	return result
}
