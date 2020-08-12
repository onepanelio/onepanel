package migration

import (
	"database/sql"
	v1 "github.com/onepanelio/core/pkg"
	uid2 "github.com/onepanelio/core/pkg/util/uid"
	"github.com/pressly/goose"
	"strings"
)

const cvatWorkspaceTemplate5 = `# Workspace arguments
arguments:
  parameters:
  - name: storage-prefix
    displayName: Directory in default object storage
    value: workflow-data
    hint: Location of data and models in default object storage, will continuously sync to '/mnt/share'. This will be automatically prefixed with the namespace name.
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
  image: onepanel/cvat:v0.7.12-defaultuser-dynamic-workflow
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
  - name: sys-namespace-config
    mountPath: /etc/onepanel
    readOnly: true
- name: cvat-ui
  image: onepanel/cvat-ui:v0.7.12-defaultuser-dynamic-workflow
  ports:
  - containerPort: 80
    name: http
# You can add multiple FileSyncer sidecar containers if needed
- name: filesyncer
  image: onepanel/filesyncer:s3
  args:
  - download
  env:
  - name: FS_PATH
    value: /mnt/share
  - name: FS_PREFIX
    value: '{{workflow.namespace}}/{{workspace.parameters.storage-prefix}}'
  volumeMounts:
  - name: share
    mountPath: /mnt/share
  - name: sys-namespace-config
    mountPath: /etc/onepanel
    readOnly: true
# Uncomment following lines to enable analytics
# - name: cvat-elasticsearch
#   image: onepanel/cvat-elasticsearch:v0.0.1
#   volumeMounts:
#   - name: events
#     mountPath: /usr/share/elasticsearch/data
#   ports:
#   - containerPort: 9200
#     name: http
# - name: cvat-kibana
#   image: onepanel/cvat-kibana:v0.0.1
#   ports:
#   - containerPort: 5601
#     name: http
#   env:
#   - name: ELASTICSEARCH_URL
#     value: http://localhost:9200
# - name: cvat-kibana-setup
#   image:  onepanel/cvat:v0.7.10-elastic
#   command: ['bash']
#   args: ['-c','/bin/bash wait-for-it.sh localhost:9200 -t 0 --; /bin/bash wait-for-it.sh localhost:5601 -t 0 -- ; /usr/bin/python3 /tmp/components/analytics/kibana/setup.py -Hlocalhost /tmp/components/analytics/kibana/export.json ; sleep infinity']
#   env:
#   - name: DJANGO_LOG_SERVER_HOST
#     value: localhost
#   - name: DJANGO_LOG_SERVER_PORT
#     value: 5000
#   - name: DJANGO_LOG_VIEWER_HOST
#     value: localhost
#   - name: DJANGO_LOG_VIEWER_PORT
#     value: 5601
# - name: cvat-logstash
#   image: onepanel/cvat-logstash:v0.0.2
#   ports:
#  - containerPort: 5000
#    name: tcp

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
      regex: /api/.*|/git/.*|/tensorflow/.*|/onepanelio/.*|/tracking/.*|/auto_annotation/.*|/analytics/.*|/static/.*|/admin/.*|/documentation/.*|/dextr/.*|/reid/.*
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

func initialize20200812113316() {
	if _, ok := initializedMigrations[20200812113316]; !ok {
		goose.AddMigration(Up20200812113316, Down20200812113316)
		initializedMigrations[20200812113316] = true
	}
}

func Up20200812113316(tx *sql.Tx) error {
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

	if _, ok := migrationsRan[20200812113316]; ok {
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
		Manifest: cvatWorkspaceTemplate5,
	}

	for _, namespace := range namespaces {
		artifactRepositoryType := "s3"
		nsConfig, err := client.GetNamespaceConfig(namespace.Name)
		if err != nil {
			return err
		}
		if nsConfig.ArtifactRepository.GCS != nil {
			artifactRepositoryType = "gcs"
		}
		workspaceTemplate.Manifest = strings.NewReplacer(
			"{{.ArtifactRepositoryType}}", artifactRepositoryType).Replace(workspaceTemplate.Manifest)
		if _, err := client.UpdateWorkspaceTemplate(namespace.Name, workspaceTemplate); err != nil {
			return err
		}
	}

	return nil
}

func Down20200812113316(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return nil
}
