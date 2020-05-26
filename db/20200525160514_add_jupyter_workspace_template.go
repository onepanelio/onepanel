package migration

import (
	"database/sql"
	"fmt"
	"github.com/jmoiron/sqlx"
	v1 "github.com/onepanelio/core/pkg"
	"github.com/pressly/goose"
	"log"
)

const jupyterWorkspaceTemplate = `# Docker containers that are part of the Workspace
containers:
- name: jupyterlab-tensorflow
  image: jupyter/tensorflow-notebook
  command: [start.sh, jupyter]
  env:
    - name: tornado
      value: "{ 'headers': { 'Content-Security-Policy': \"frame-ancestors * 'self'\" }  }"
  args:
    - lab
    - --LabApp.token=''
    - --LabApp.allow_remote_access=True
    - --LabApp.allow_origin="*"
    - --LabApp.disable_check_xsrf=True
    - --LabApp.trust_xheaders=True
    - --LabApp.tornado_settings=$(tornado)
    - --notebook-dir='/data'
  ports:
  - containerPort: 8888
    name: jupyterlab
  # Volumes to be mounted in this container
  # Onepanel will automatically create these volumes and mount them to the container
  volumeMounts:
  - name: data
    mountPath: /data
# Ports that need to be exposed
ports:
- name: jupyterlab
  port: 80
  protocol: TCP
  targetPort: 8888
# Routes that will map to ports
routes:
- match:
  - uri:
      prefix: /
  route:
  - destination:
      port:
        number: 80
# DAG Workflow to be executed once a Workspace action completes
postExecutionWorkflow:
  entrypoint: main
  templates:
  - name: main
    dag:
       tasks:
       - name: slack-notify
         template: slack-notify
  -  name: slack-notify
     container:
       image: technosophos/slack-notify
       args:
       - SLACK_USERNAME=onepanel SLACK_TITLE="Your workspace is ready" SLACK_ICON=https://www.gravatar.com/avatar/5c4478592fe00878f62f0027be59c1bd SLACK_MESSAGE="Your workspace is now running" ./slack-notify
       command:
       - sh
       - -c`

const jupyterLabTemplateName = "jupyterlab"

func init() {
	goose.AddMigration(Up20200525160514, Down20200525160514)
}

func getClient() (*v1.Client, error) {
	kubeConfig := v1.NewConfig()
	client, err := v1.NewClient(kubeConfig, nil)
	if err != nil {
		return nil, err
	}
	config, err := client.GetSystemConfig()
	if err != nil {
		return nil, err
	}

	databaseDataSourceName := fmt.Sprintf("host=%v user=%v password=%v dbname=%v sslmode=disable",
		"localhost", "admin", "admin", "onepanel-core")
	client.DB = sqlx.MustConnect(config["databaseDriverName"], databaseDataSourceName)

	return client, nil
}

func Up20200525160514(tx *sql.Tx) error {
	client, err := getClient()
	if err != nil {
		return err
	}

	namespaces, err := client.ListOnepanelEnabledNamespaces()
	if err != nil {
		return err
	}

	workspaceTemplate := &v1.WorkspaceTemplate{
		Name:     jupyterLabTemplateName,
		Manifest: jupyterWorkspaceTemplate,
	}

	for _, namespace := range namespaces {
		if _, err := client.CreateWorkspaceTemplate(namespace.Name, workspaceTemplate); err != nil {
			log.Printf("error %v", err.Error())
		}
	}

	return nil
}

func Down20200525160514(tx *sql.Tx) error {
	client, err := getClient()
	if err != nil {
		return err
	}

	namespaces, err := client.ListOnepanelEnabledNamespaces()
	if err != nil {
		return err
	}

	for _, namespace := range namespaces {
		if err := client.DeleteWorkspace(namespace.Name, jupyterLabTemplateName); err != nil {
			log.Printf("error %v", err.Error())
		}
	}

	return nil
}
