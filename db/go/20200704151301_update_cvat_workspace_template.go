package migration

import (
	"database/sql"
	v1 "github.com/onepanelio/core/pkg"
	uid2 "github.com/onepanelio/core/pkg/util/uid"
	"github.com/pressly/goose"
)

const cvatWorkspaceTemplate3 = `# Docker containers that are part of the Workspace
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
  image: onepanel/cvat:v0.7.10-stable
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
  - name: share
    mountPath: /home/django/share
- name: cvat-ui
  image: onepanel/cvat-ui:v0.7.10-stable
  ports:
  - containerPort: 80
    name: http
# Uncomment following lines to enable S3 FileSyncer
# Refer to https://docs.onepanel.ai/docs/getting-started/use-cases/computervision/annotation/cvat/cvat_quick_guide#setting-up-environment-variables
#- name: filesyncer
#  image: onepanel/filesyncer:v0.0.4
#  command: ['python3', 'main.py']
#  volumeMounts:
#  - name: share
#    mountPath: /mnt/share
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
  timeout: 600s
- match:
  - uri:
      prefix: /
  route:
  - destination:
      port:
        number: 80
  timeout: 600s
# DAG Workflow to be executed once a Workspace action completes (optional)
# Uncomment the lines below if you want to send Slack notifications
#postExecutionWorkflow:
#  entrypoint: main
#  templates:
#  - name: main
#    dag:
#       tasks:
#       - name: slack-notify
#         template: slack-notify
#  - name: slack-notify
#     container:
#       image: technosophos/slack-notify
#       args:
#       - SLACK_USERNAME=onepanel SLACK_TITLE="Your workspace is ready" SLACK_ICON=https://www.gravatar.com/avatar/5c4478592fe00878f62f0027be59c1bd SLACK_MESSAGE="Your workspace is now running" ./slack-notify
#       command:
#       - sh
#       - -c
`

func initialize20200704151301() {
	if _, ok := initializedMigrations[20200704151301]; !ok {
		goose.AddMigration(Up20200704151301, Down20200704151301)
		initializedMigrations[20200704151301] = true
	}
}

// Up20200704151301 updates the CVAT template to a new version.
func Up20200704151301(tx *sql.Tx) error {
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

	if _, ok := migrationsRan[20200704151301]; ok {
		return nil
	}

	namespaces, err := client.ListOnepanelEnabledNamespaces()
	if err != nil {
		return err
	}

	uid, err := uid2.GenerateUID(cvatTemplateName, 30)
	if err != nil {
		return err
	}
	workspaceTemplate := &v1.WorkspaceTemplate{
		UID:      uid,
		Name:     cvatTemplateName,
		Manifest: cvatWorkspaceTemplate3,
	}

	for _, namespace := range namespaces {
		if _, err := client.UpdateWorkspaceTemplate(namespace.Name, workspaceTemplate); err != nil {
			return err
		}
	}

	return nil
}

// Down20200704151301 removes the CVAT template update
func Down20200704151301(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return nil
}
