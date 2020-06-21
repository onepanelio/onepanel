package v1

import (
	"fmt"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
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
	Labels                     map[string]string
	WorkflowTemplateID         uint64 `db:"workflow_template_id"`
}

// InjectRuntimeVariables will inject all runtime variables into the WorkflowTemplate's manifest.
func (wt *WorkspaceTemplate) InjectRuntimeVariables(config SystemConfig) error {
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
