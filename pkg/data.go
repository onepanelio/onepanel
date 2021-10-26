package v1

import (
	"github.com/onepanelio/core/pkg/util/data"
	"github.com/onepanelio/core/pkg/util/extensions"
)

// createWorkspaceTemplateFromGenericFile will create the workspace template given by {{templateName}} with the contents
// given by {{filename}} for the input {{namespace}}
func (c *Client) createWorkspaceTemplateFromGenericManifest(namespace string, manifestFile *data.ManifestFile) (err error) {
	manifest, err := manifestFile.SpecString()
	if err != nil {
		return err
	}
	templateName := manifestFile.Metadata.Name
	description := manifestFile.Metadata.Description

	artifactRepositoryType, err := c.GetArtifactRepositoryType(namespace)
	if err != nil {
		return err
	}

	replaceMap := map[string]string{
		"{{.ArtifactRepositoryType}}": artifactRepositoryType,
	}
	manifest = extensions.ReplaceMapValues(manifest, replaceMap)

	workspaceTemplate, err := CreateWorkspaceTemplate(templateName)
	if err != nil {
		return err
	}
	workspaceTemplate.Manifest = manifest

	if description != nil {
		workspaceTemplate.Description = *description
	}

	_, err = c.CreateWorkspaceTemplate(namespace, workspaceTemplate)

	return
}

// updateWorkspaceTemplateManifest will update the workspace template given by {{templateName}} with the contents
// given by {{filename}}
func (c *Client) updateWorkspaceTemplateManifest(namespace string, manifestFile *data.ManifestFile) (err error) {
	manifest, err := manifestFile.SpecString()
	if err != nil {
		return err
	}
	templateName := manifestFile.Metadata.Name

	artifactRepositoryType, err := c.GetArtifactRepositoryType(namespace)
	if err != nil {
		return err
	}

	replaceMap := map[string]string{
		"{{.ArtifactRepositoryType}}": artifactRepositoryType,
	}
	manifest = extensions.ReplaceMapValues(manifest, replaceMap)

	workspaceTemplate, err := CreateWorkspaceTemplate(templateName)
	if err != nil {
		return err
	}
	workspaceTemplate.Manifest = manifest

	_, err = c.UpdateWorkspaceTemplateManifest(namespace, workspaceTemplate.UID, workspaceTemplate.Manifest)

	return
}

// createWorkflowTemplate will create the workflow template given by {{templateName}} with the contents
// given by {{filename}}
func (c *Client) createWorkflowTemplateFromGenericManifest(namespace string, manifestFile *data.ManifestFile) (err error) {
	manifest, err := manifestFile.SpecString()
	if err != nil {
		return err
	}
	templateName := manifestFile.Metadata.Name
	labels := manifestFile.Metadata.Labels

	artifactRepositoryType, err := c.GetArtifactRepositoryType(namespace)
	if err != nil {
		return err
	}

	replaceMap := map[string]string{
		"{{.ArtifactRepositoryType}}": artifactRepositoryType,
	}
	manifest = extensions.ReplaceMapValues(manifest, replaceMap)

	workflowTemplate, err := CreateWorkflowTemplate(templateName)
	if err != nil {
		return
	}
	workflowTemplate.Manifest = manifest
	workflowTemplate.Labels = labels

	_, err = c.CreateWorkflowTemplate(namespace, workflowTemplate)

	return
}

// updateWorkflowTemplateManifest will update the workflow template given by {{templateName}} with the contents
// given by {{filename}}
func (c *Client) updateWorkflowTemplateManifest(namespace string, manifestFile *data.ManifestFile) (err error) {
	manifest, err := manifestFile.SpecString()
	if err != nil {
		return err
	}
	templateName := manifestFile.Metadata.Name
	labels := manifestFile.Metadata.Labels

	artifactRepositoryType, err := c.GetArtifactRepositoryType(namespace)
	if err != nil {
		return err
	}

	replaceMap := map[string]string{
		"{{.ArtifactRepositoryType}}": artifactRepositoryType,
	}
	manifest = extensions.ReplaceMapValues(manifest, replaceMap)

	workflowTemplate, err := CreateWorkflowTemplate(templateName)
	if err != nil {
		return
	}
	workflowTemplate.Manifest = manifest
	workflowTemplate.Labels = labels

	_, err = c.CreateWorkflowTemplateVersion(namespace, workflowTemplate)

	return
}
