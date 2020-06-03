package migration

import (
	"database/sql"
	v1 "github.com/onepanelio/core/pkg"
	uid2 "github.com/onepanelio/core/pkg/util/uid"
	"github.com/pressly/goose"
	"log"
)

const cvatWorkspaceTemplate = `# Docker containers that are part of the Workspace
containers:
- name: cvat-db
  image: postgres:10-alpine
  env:
  - name: POSTGRES_USER
    value: root
  - name: POSTGRES_DB
    value: cvat
  - name: POSTGRES_HOST_AUTH_METHOD
    value: trust
  - name: PGDATA
    value: /var/lib/psql/data
  ports:
  - containerPort: 5432
    name: tcp
  volumeMounts:
  - name: db
    mountPath: /var/lib/psql
- name: cvat-redis
  image: redis:4.0-alpine
  ports:
  - containerPort: 6379
    name: tcp
- name: cvat
  image: onepanel/cvat:v0.6.24
  command: ["/bin/bash", "-c"]
  args: ["/usr/bin/supervisord && /usr/bin/python3 ~/manage.py shell --command='import os;from django.contrib.auth.models import User;u = User(username=os.getenv('DJANGO_SUPERUSER_USERNAME','admin'));u.set_password(os.getenv('DJANGO_SUPERUSER_PASSWORD','admin'));u.is_superuser = True;u.is_staff = True;u.save();'"]
  env:
  - name: DJANGO_MODWSGI_EXTRA_ARGS
    value: ""
  - name: ALLOWED_HOSTS
    value: '*'
  - name: CVAT_REDIS_HOST
    value: localhost
  - name: CVAT_POSTGRES_HOST
    value: localhost
  - name: CVAT_SHARE_URL
    value: /home/django/data
  ports:
  - containerPort: 8080
    name: http
  volumeMounts:
  - name: data
    mountPath: /home/django/data
  - name: keys
    mountPath: /home/django/keys
  - name: logs
    mountPath: /home/django/logs
  - name: models
    mountPath: /home/django/models
- name: cvat-ui
  image: onepanel/cvat-ui:v0.6.24
  ports:
  - containerPort: 80
    name: http
ports:
- name: cvat-ui
  port: 80
  protocol: TCP
  targetPort: 80
- name: cvat
  port: 8080
  protocol: TCP
  targetPort: 8080
routes:
- match:
  - uri:
      regex: /api/.*|/git/.*|/tensorflow/.*|/auto_annotation/.*|/analytics/.*|/static/.*|/admin/.*|/documentation/.*|/dextr/.*|/reid/.*
  - queryParams:
      id:
        regex: \d+.*
  route:
  - destination:
      port:
        number: 8080
- match:
  - uri:
      prefix: /
  route:
  - destination:
      port:
        number: 80
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
       - -c
`

const cvatTemplateName = "CVAT"

func init() {
	goose.AddMigration(Up20200528140124, Down20200528140124)
}

func Up20200528140124(tx *sql.Tx) error {
	// This code is executed when the migration is applied.

	client, err := getClient()
	if err != nil {
		return err
	}

	namespaces, err := client.ListOnepanelEnabledNamespaces()
	if err != nil {
		return err
	}

	workspaceTemplate := &v1.WorkspaceTemplate{
		Name:     cvatTemplateName,
		Manifest: cvatWorkspaceTemplate,
	}

	for _, namespace := range namespaces {
		if _, err := client.CreateWorkspaceTemplate(namespace.Name, workspaceTemplate); err != nil {
			log.Fatalf("error %v", err.Error())
		}
	}

	return nil
}

func Down20200528140124(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.

	client, err := getClient()
	if err != nil {
		return err
	}

	namespaces, err := client.ListOnepanelEnabledNamespaces()
	if err != nil {
		return err
	}

	uid, err := uid2.GenerateUID(cvatTemplateName, 30)
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
