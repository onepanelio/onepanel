package migration

import (
	"database/sql"
	uid2 "github.com/onepanelio/core/pkg/util/uid"
	"github.com/pressly/goose"
)

const vscodeWorkspaceTemplate2 = `
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
    lifecycle:
      postStart:
        exec:
          command: 
           - /bin/sh
           - -c
           - >
            condayml="/data/.environment.yml";
            vscodetxt="/data/.vscode-extensions.txt";
            if [ -f "$condayml" ]; then conda env update -f $condayml; fi;
            if [ -f "$vscodetxt" ]; then cat $vscodetxt | xargs -n 1 code-server --install-extension; fi;
      preStop:
        exec:
          command: 
          - /bin/sh
          - -c
          - >
            conda env export > /data/.environment.yml -n base;
            code-server --list-extensions | tail -n +2 > /data/.vscode-extensions.txt;
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

func initialize20201008153041() {
	if _, ok := initializedMigrations[20201008153041]; !ok {
		goose.AddMigration(Up20201008153041, Down20201008153041)
		initializedMigrations[20201008153041] = true
	}
}

// Up20201008153041 migration will add lifecycle hooks to VSCode template.
// These hooks will attempt to export the conda, pip, and vscode packages that are installed,
// to a text file.
// On workspace resume / start, the code then tries to install these packages.
func Up20201008153041(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	client, err := getClient()
	if err != nil {
		return err
	}
	defer client.DB.Close()

	migrationsRan, err := getRanSQLMigrations(client)
	if err != nil {
		return err
	}

	if _, ok := migrationsRan[20201008153041]; ok {
		return nil
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
		if _, err := client.UpdateWorkspaceTemplateManifest(namespace.Name, uid, vscodeWorkspaceTemplate2); err != nil {
			return err
		}
	}

	return nil
}

// Down20201008153041 removes the lifecycle hooks from VSCode workspace template.
func Down20201008153041(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	client, err := getClient()
	if err != nil {
		return err
	}
	defer client.DB.Close()

	migrationsRan, err := getRanSQLMigrations(client)
	if err != nil {
		return err
	}

	if _, ok := migrationsRan[20201008153041]; ok {
		return nil
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
		if _, err := client.UpdateWorkspaceTemplateManifest(namespace.Name, uid, vscodeWorkspaceTemplate); err != nil {
			return err
		}
	}
	return nil
}
