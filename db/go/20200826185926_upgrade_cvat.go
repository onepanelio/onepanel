package migration

import (
	"database/sql"
	v1 "github.com/onepanelio/core/pkg"
	uid2 "github.com/onepanelio/core/pkg/util/uid"
	"github.com/pressly/goose"
)

const cvatWorkspaceTemplate8 = `# Workspace arguments
arguments:
  parameters:
  - name: sync-directory
    displayName: Directory to sync raw input and training output
    value: workflow-data
    hint: Location to sync raw input, models and checkpoints from default object storage. Note that this will be relative to the current namespace.
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
  image: onepanel/cvat:0.12.0_cvat.1.0.0
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
  - name: ONEPANEL_SYNC_DIRECTORY
    value: '{{workspace.parameters.sync-directory}}'
  - name: NVIDIA_VISIBLE_DEVICES
    value: all
  - name: NVIDIA_DRIVER_CAPABILITIES
    value: compute,utility
  - name: NVIDIA_REQUIRE_CUDA
    value: "cuda>=10.0 brand=tesla,driver>=384,driver<385 brand=tesla,driver>=410,driver<411"
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
  image: onepanel/cvat-ui:0.12.0_cvat.1.0.0
  ports:
  - containerPort: 80
    name: http
# You can add multiple FileSyncer sidecar containers if needed
- name: filesyncer
  image: onepanel/filesyncer:{{.ArtifactRepositoryType}}
  imagePullPolicy: Always
  args:
  - download
  - -server-prefix=/sys/filesyncer
  env:
  - name: FS_PATH
    value: /mnt/share
  - name: FS_PREFIX
    value: '{{workflow.namespace}}/{{workspace.parameters.sync-directory}}'
  volumeMounts:
  - name: share
    mountPath: /mnt/share
  - name: sys-namespace-config
    mountPath: /etc/onepanel
    readOnly: true
ports:
- name: cvat-ui
  port: 80
  protocol: TCP
  targetPort: 80
- name: cvat
  port: 8080
  protocol: TCP
  targetPort: 8080
- name: fs
  port: 8888
  protocol: TCP
  targetPort: 8888
routes:
- match:
  - uri:
      prefix: /sys/filesyncer
  route:
  - destination:
      port:
        number: 8888
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
#       - -c`

func initialize20200826185926() {
	if _, ok := initializedMigrations[20200826185926]; !ok {
		goose.AddMigration(Up20200826185926, Down20200826185926)
		initializedMigrations[20200826185926] = true
	}
}

// Up20200826185926 runs the migration to upgrade the cvat workspace template
func Up20200826185926(tx *sql.Tx) error {
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

	if _, ok := migrationsRan[20200826185926]; ok {
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

	for _, namespace := range namespaces {
		workspaceTemplate := &v1.WorkspaceTemplate{
			UID:         uid,
			Name:        cvatTemplateName,
			Manifest:    cvatWorkspaceTemplate8,
			Description: "Powerful and efficient Computer Vision Annotation Tool (CVAT)",
		}
		err = ReplaceArtifactRepositoryType(client, namespace, nil, workspaceTemplate)
		if err != nil {
			return err
		}
		if _, err := client.UpdateWorkspaceTemplate(namespace.Name, workspaceTemplate); err != nil {
			return err
		}
	}

	return nil
}

// Down20200826185926 does nothing
func Down20200826185926(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return nil
}
