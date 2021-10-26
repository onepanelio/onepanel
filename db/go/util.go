package migration

import (
	"fmt"
	v1 "github.com/onepanelio/core/pkg"
	"github.com/onepanelio/core/pkg/util/data"
	uid2 "github.com/onepanelio/core/pkg/util/uid"
	"path/filepath"
)

// createWorkspaceTemplate will create the workspace template given by {{templateName}} with the contents
// given by {{filename}}
// It will do so for all namespaces.
func createWorkspaceTemplate(filename, templateName, description string) error {
	client, err := getClient()
	if err != nil {
		return err
	}
	defer client.DB.Close()

	namespaces, err := client.ListOnepanelEnabledNamespaces()
	if err != nil {
		return err
	}

	filename = filepath.Join("db", "yaml", filename)
	manifestFile, err := data.ManifestFileFromFile(filename)
	if err != nil {
		return err
	}

	newManifest, err := manifestFile.SpecString()
	if err != nil {
		return err
	}

	uid, err := uid2.GenerateUID(templateName, 30)
	if err != nil {
		return err
	}

	for _, namespace := range namespaces {
		workspaceTemplate := &v1.WorkspaceTemplate{
			UID:         uid,
			Name:        templateName,
			Manifest:    newManifest,
			Description: description,
		}

		err = ReplaceArtifactRepositoryType(client, namespace, nil, workspaceTemplate)
		if err != nil {
			return err
		}

		if _, err := client.CreateWorkspaceTemplate(namespace.Name, workspaceTemplate); err != nil {
			return err
		}
	}

	return nil
}

func archiveWorkspaceTemplate(templateName string) error {
	client, err := getClient()
	if err != nil {
		return err
	}
	defer client.DB.Close()

	namespaces, err := client.ListOnepanelEnabledNamespaces()
	if err != nil {
		return err
	}

	uid, err := uid2.GenerateUID(templateName, 30)
	if err != nil {
		return err
	}

	for _, namespace := range namespaces {
		hasRunning, err := client.WorkspaceTemplateHasRunningWorkspaces(namespace.Name, uid)
		if err != nil {
			return fmt.Errorf("Unable to get check running workspaces")
		}
		if hasRunning {
			return fmt.Errorf("unable to archive workspace template. There are running workspaces that use it")
		}

		_, err = client.ArchiveWorkspaceTemplate(namespace.Name, uid)
		if err != nil {
			return err
		}
	}

	return nil
}

// updateWorkspaceTemplateManifest will update the workspace template given by {{templateName}} with the contents
// given by {{filename}}
// It will do so for all namespaces.
func updateWorkspaceTemplateManifest(filename, templateName string) error {
	client, err := getClient()
	if err != nil {
		return err
	}
	defer client.DB.Close()

	filename = filepath.Join("db", "yaml", filename)

	namespaces, err := client.ListOnepanelEnabledNamespaces()
	if err != nil {
		return err
	}

	manifest, err := data.ManifestFileFromFile(filename)
	if err != nil {
		return err
	}

	newManifest, err := manifest.SpecString()
	if err != nil {
		return err
	}

	uid, err := uid2.GenerateUID(templateName, 30)
	if err != nil {
		return err
	}

	for _, namespace := range namespaces {
		workspaceTemplate := &v1.WorkspaceTemplate{
			UID:      uid,
			Name:     templateName,
			Manifest: newManifest,
		}
		err = ReplaceArtifactRepositoryType(client, namespace, nil, workspaceTemplate)
		if err != nil {
			return err
		}
		if _, err := client.UpdateWorkspaceTemplateManifest(namespace.Name, uid, workspaceTemplate.Manifest); err != nil {
			return err
		}
	}

	return nil
}

// createWorkflowTemplate will create the workflow template given by {{templateName}} with the contents
// given by {{filename}}
// It will do so for all namespaces.
func createWorkflowTemplate(filename, templateName string, labels map[string]string) error {
	client, err := getClient()
	if err != nil {
		return err
	}
	defer client.DB.Close()

	namespaces, err := client.ListOnepanelEnabledNamespaces()
	if err != nil {
		return err
	}

	filename = filepath.Join("db", "yaml", filename)

	manifestFile, err := data.ManifestFileFromFile(filename)
	if err != nil {
		return err
	}

	manifest, err := manifestFile.SpecString()
	if err != nil {
		return err
	}

	uid, err := uid2.GenerateUID(templateName, 30)
	if err != nil {
		return err
	}

	for _, namespace := range namespaces {
		workflowTemplate := &v1.WorkflowTemplate{
			UID:      uid,
			Name:     templateName,
			Manifest: manifest,
			Labels:   labels,
		}

		workflowTemplate.Manifest, err = ReplaceRuntimeVariablesInManifest(client, namespace.Name, workflowTemplate.Manifest)
		if err != nil {
			return err
		}
		if _, err := client.CreateWorkflowTemplate(namespace.Name, workflowTemplate); err != nil {
			return err
		}
	}

	return nil
}

// updateWorkflowTemplateManifest will update the workflow template given by {{templateName}} with the contents
// given by {{filename}}
// It will do so for all namespaces.
func updateWorkflowTemplateManifest(filename, templateName string, labels map[string]string) error {
	client, err := getClient()
	if err != nil {
		return err
	}
	defer client.DB.Close()

	namespaces, err := client.ListOnepanelEnabledNamespaces()
	if err != nil {
		return err
	}

	filename = filepath.Join("db", "yaml", filename)

	manifestFile, err := data.ManifestFileFromFile(filename)
	if err != nil {
		return err
	}

	newManifest, err := manifestFile.SpecString()
	if err != nil {
		return err
	}

	uid, err := uid2.GenerateUID(templateName, 30)
	if err != nil {
		return err
	}

	for _, namespace := range namespaces {
		workflowTemplate := &v1.WorkflowTemplate{
			UID:      uid,
			Name:     templateName,
			Manifest: newManifest,
			Labels:   labels,
		}

		workflowTemplate.Manifest, err = ReplaceRuntimeVariablesInManifest(client, namespace.Name, workflowTemplate.Manifest)
		if err != nil {
			return err
		}

		if _, err := client.CreateWorkflowTemplateVersion(namespace.Name, workflowTemplate); err != nil {
			return err
		}
	}

	return nil
}

// archiveWorkflowTemplate removes a Workflow Template by a given templateName
func archiveWorkflowTemplate(templateName string) error {
	client, err := getClient()
	if err != nil {
		return err
	}
	defer client.DB.Close()

	namespaces, err := client.ListOnepanelEnabledNamespaces()
	if err != nil {
		return err
	}

	uid, err := uid2.GenerateUID(templateName, 30)
	if err != nil {
		return err
	}

	for _, namespace := range namespaces {
		if _, err := client.ArchiveWorkflowTemplate(namespace.Name, uid); err != nil {
			return err
		}
	}

	return nil
}
