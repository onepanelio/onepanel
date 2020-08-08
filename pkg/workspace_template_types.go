package v1

import (
	"fmt"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/onepanelio/core/pkg/util/types"
	uid2 "github.com/onepanelio/core/pkg/util/uid"
	"github.com/onepanelio/core/util/sql"
	"sigs.k8s.io/yaml"
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
	Labels                     types.JSONLabels
	WorkflowTemplateID         uint64 `db:"workflow_template_id"`
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
	columns := []string{"id", "uid", "created_at", "modified_at", "name", "namespace", "is_archived", "workflow_template_id", "labels"}
	return sql.FormatColumnSelect(columns, aliasAndDestination...)
}
