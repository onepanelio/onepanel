package migration

import (
	"database/sql"
	uid2 "github.com/onepanelio/core/pkg/util/uid"
	"github.com/pressly/goose"
)

const jupyterWorkspaceTemplate4 = `
# Docker containers that are part of the Workspace
containers:
- name: jupyterlab
  image: onepanel/jupyterlab:1.0.1
  command: ["/bin/bash", "-c", "pip install onepanel-sdk && start.sh jupyter lab --LabApp.token='' --LabApp.allow_remote_access=True --LabApp.allow_origin=\"*\" --LabApp.disable_check_xsrf=True --LabApp.trust_xheaders=True --LabApp.base_url=/ --LabApp.tornado_settings='{\"headers\":{\"Content-Security-Policy\":\"frame-ancestors * \'self\'\"}}' --notebook-dir='/data' --allow-root"]
  env:
    - name: tornado
      value: "'{'headers':{'Content-Security-Policy':\"frame-ancestors\ *\ \'self'\"}}'"
  ports:
  - containerPort: 8888
    name: jupyterlab
  - containerPort: 6006
    name: tensorboard
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
          jupytertxt="/data/.jupexported.txt";
          if [ -f "$condayml" ]; then conda env update -f $condayml; fi;
          if [ -f "$jupytertxt" ]; then cat $jupytertxt | xargs -n 1 jupyter labextension install --no-build && jupyter lab build --minimize=False; fi;
    preStop:
      exec:
        command:
        - /bin/sh
        - -c
        - >
          conda env export > /data/.environment.yml -n base;
          jupyter labextension list 1>/dev/null 2> /data/.jup.txt;
          cat /data/.jup.txt | sed -n '2,$p' | awk 'sub(/v/,"@", $2){print $1$2}' > /data/.jupexported.txt;
ports:
- name: jupyterlab
  port: 80
  protocol: TCP
  targetPort: 8888
- name: tensorboard
  port: 6006
  protocol: TCP
  targetPort: 6006
routes:
- match:
  - uri:
      prefix: /tensorboard
  route:
  - destination:
      port:
        number: 6006
- match:
  - uri:
      prefix: / #jupyter runs at the default route
  route:
  - destination:
      port:
        number: 80
# DAG Workflow to be executed once a Workspace action completes (optional)
#postExecutionWorkflow:
#  entrypoint: main
#  templates:
#  - name: main
#    dag:
#       tasks:
#       - name: slack-notify
#         template: slack-notify
#  - name: slack-notify
#    container:
#      image: technosophos/slack-notify
#      args:
#      - SLACK_USERNAME=onepanel SLACK_TITLE="Your workspace is ready" SLACK_ICON=https://www.gravatar.com/avatar/5c4478592fe00878f62f0027be59c1bd SLACK_MESSAGE="Your workspace is now running" ./slack-notify
#      command:
#      - sh
#      - -c
`

func initialize20201008153033() {
	if _, ok := initializedMigrations[20201008153033]; !ok {
		goose.AddMigration(Up20201008153033, Down20201008153033)
		initializedMigrations[20201008153033] = true
	}
}

// Up20201008153033 updates the jupyterlab workspace to include container lifecycle hooks.
// These hooks will attempt to persist conda, pip, and jupyterlab extensions between pause and shut-down.
func Up20201008153033(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
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

	uid, err := uid2.GenerateUID(jupyterLabTemplateName, 30)
	if err != nil {
		return err
	}
	for _, namespace := range namespaces {
		if _, err := client.UpdateWorkspaceTemplateManifest(namespace.Name, uid, jupyterWorkspaceTemplate4); err != nil {
			return err
		}
	}

	return nil
}

// Down20201008153033 removes the lifecycle hooks from the template.
func Down20201008153033(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	// This code is executed when the migration is rolled back.
	client, err := getClient()
	if err != nil {
		return err
	}
	defer client.DB.Close()

	namespaces, err := client.ListOnepanelEnabledNamespaces()
	if err != nil {
		return err
	}

	uid, err := uid2.GenerateUID(jupyterLabTemplateName, 30)
	if err != nil {
		return err
	}

	for _, namespace := range namespaces {
		if _, err := client.UpdateWorkspaceTemplateManifest(namespace.Name, uid, jupyterWorkspaceTemplate3); err != nil {
			return err
		}
	}
	return nil
}
