package migration

import (
	"database/sql"
	v1 "github.com/onepanelio/core/pkg"
	uid2 "github.com/onepanelio/core/pkg/util/uid"
	"github.com/pressly/goose"
	"log"
)

const vscodeWorkspaceTemplate = `
# Docker containers that are part of the Workspace
containers:
  - name: vscode
    image: onepanel/vscode:1.0.0
    command: ["/bin/bash", "-c", "pip install onepanel-sdk && /usr/bin/entrypoint.sh --bind-addr 0.0.0.0:8080 --auth none ."]
    ports:
      - containerPort: 8080
        name: vscode
    volumeMounts:
      - name: data
        mountPath: /data
ports:
  - name: vscode
    port: 8080
    protocol: TCP
    targetPort: 8080
routes:
  - match:
      - uri:
          prefix: / #vscode runs at the default route
    route:
      - destination:
          port:
            number: 8080
# DAG Workflow to be executed once a Workspace action completes (optional)        
#postExecutionWorkflow:
#  entrypoint: main
#  templates:
#  - name: main
#    dag:
#       tasks:
#       - name: slack-notify
#         template: slack-notify
#  -  name: slack-notify
#     container:
#       image: technosophos/slack-notify
#       args:
#       - SLACK_USERNAME=onepanel SLACK_TITLE="Your workspace is ready" SLACK_ICON=https://www.gravatar.com/avatar/5c4478592fe00878f62f0027be59c1bd SLACK_MESSAGE="Your workspace is now running" ./slack-notify
#       command:
#       - sh
#       - -c
`

const vscodeWorkspaceTemplateName = "VisualStudioCode"

func initialize20200929144301() {
	if _, ok := initializedMigrations[20200929144301]; !ok {
		goose.AddMigration(Up20200929144301, Down20200929144301)
		initializedMigrations[20200929144301] = true
	}
}

// Up20200929144301 adds Visual Studio Code as a workspace template.
func Up20200929144301(tx *sql.Tx) error {
	client, err := getClient()
	if err != nil {
		return err
	}
	defer client.DB.Close()

	migrationsRan, err := getRanSQLMigrations(client)
	if err != nil {
		return err
	}

	if _, ok := migrationsRan[20200929144301]; ok {
		return nil
	}

	namespaces, err := client.ListOnepanelEnabledNamespaces()
	if err != nil {
		return err
	}

	workspaceTemplate := &v1.WorkspaceTemplate{
		Name:     vscodeWorkspaceTemplateName,
		Manifest: vscodeWorkspaceTemplate,
	}

	// Adding description
	workspaceTemplate.Description = "Interactive development environment for code, notebooks, and everything in between."

	for _, namespace := range namespaces {
		if _, err := client.CreateWorkspaceTemplate(namespace.Name, workspaceTemplate); err != nil {
			return err
		}
	}

	return nil
}

// Down20200929144301 removes Visual Studio Code from workspace templates.
func Down20200929144301(tx *sql.Tx) error {
	client, err := getClient()
	if err != nil {
		return err
	}

	namespaces, err := client.ListOnepanelEnabledNamespaces()
	if err != nil {
		return err
	}
	uid, err := uid2.GenerateUID(vscodeWorkspaceTemplateName, 30)
	if err != nil {
		return err
	}
	for _, namespace := range namespaces {
		if _, err := client.ArchiveWorkspaceTemplate(namespace.Name, uid); err != nil {
			log.Fatalf("error %v", err.Error())
		}
	}
	return nil
}
