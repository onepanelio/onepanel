package migration

import (
	"database/sql"
	v1 "github.com/onepanelio/core/pkg"
	uid2 "github.com/onepanelio/core/pkg/util/uid"
	"github.com/pressly/goose"
)

func initialize20201016170415() {
	if _, ok := initializedMigrations[20201016170415]; !ok {
		goose.AddMigration(Up20201016170415, Down20201016170415)
		initializedMigrations[20201016170415] = true
	}
}

// Up20201016170415 updates cvat to a new version
func Up20201016170415(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	client, err := getClient()
	if err != nil {
		return err
	}
	defer client.DB.Close()

	namespaces, err := client.ListOnepanelEnabledNamespaces()
	if err != nil {
		return err
	}

	newManifest, err := readDataFile("cvat9.yaml")
	if err != nil {
		return err
	}

	uid, err := uid2.GenerateUID(cvatTemplateName, 30)
	if err != nil {
		return err
	}

	for _, namespace := range namespaces {
		workspaceTemplate := &v1.WorkspaceTemplate{
			UID:      uid,
			Name:     cvatTemplateName,
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

// Down20201016170415 does nothing
func Down20201016170415(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return nil
}
