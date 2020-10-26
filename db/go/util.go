package migration

import (
	v1 "github.com/onepanelio/core/pkg"
	uid2 "github.com/onepanelio/core/pkg/util/uid"
)

// updateWorkspaceTemplateManifest will update the workspace template given by {{templateName}} with the contents
// given by {{filename}}
// It will do so for all namespaces.
func updateWorkspaceTemplateManifest(filename, templateName string) error {
	client, err := getClient()
	if err != nil {
		return err
	}
	defer client.DB.Close()

	namespaces, err := client.ListOnepanelEnabledNamespaces()
	if err != nil {
		return err
	}

	newManifest, err := readDataFile(filename)
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
