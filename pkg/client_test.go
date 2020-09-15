package v1

import (
	"flag"
	"fmt"
	argoFake "github.com/argoproj/argo/pkg/client/clientset/versioned/fake"
	"github.com/jmoiron/sqlx"
	"github.com/pressly/goose"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"log"
	"os"
	"testing"
)

var (
	mockSystemSecret = &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "onepanel",
			Namespace: "onepanel",
		},
	}

	configArtifactRepository = `archiveLogs: true
s3:
  keyFormat: artifacts/{{workflow.namespace}}/{{workflow.name}}/{{pod.name}}
  bucket: test.onepanel.io
  endpoint: s3.amazonaws.com
  insecure: false
  region: us-west-2
  accessKeySecret:
    name: onepanel
    key: artifactRepositoryS3AccessKey
  secretKeySecret:
    name: onepanel
    key: artifactRepositoryS3SecretKey`

	mockSystemConfigMap = &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "onepanel",
			Namespace: "onepanel",
		},
		Data: map[string]string{
			"ONEPANEL_HOST":            "demo.onepanel.site",
			"ONEPANEL_DOMAIN":          "demo.onepanel.site",
			"artifactRepository":       configArtifactRepository,
			"applicationNodePoolLabel": "beta.kubernetes.io/instance-type",
			"applicationNodePoolOptions": `
- name: 'CPU: 2, RAM: 8GB'
  value: 'Standard_D2s_v3'
  default: true
- name: 'CPU: 4, RAM: 16GB'
  value: 'Standard_D4s_v3'
- name: 'CPU: 8, RAM: 32GB'
  value: 'Standard_D5s_v3'
`,
		},
	}

	database *sqlx.DB
)

var flagDatabaseService = flag.String("db", "localhost", "Name to connect to db, defaults to localhost")

func TestMain(m *testing.M) {
	// call flag.Parse() here if TestMain uses flags
	flag.Parse()

	databaseDataSourceName := fmt.Sprintf("host=%v user=%v password=%v dbname=%v sslmode=disable",
		*flagDatabaseService, "admin", "tester", "onepanel")

	dbDriverName := "postgres"
	database = sqlx.MustConnect(dbDriverName, databaseDataSourceName)

	// We don't run the go migrations as those setup data that we don't use in our testing
	if err := goose.Run("up", database.DB, "../db/sql"); err != nil {
		log.Fatalf("Failed to run database migrations: %v", err)
	}

	os.Exit(m.Run())
}

func NewTestClient(db *sqlx.DB, objects ...runtime.Object) (client *Client) {
	k8sFake := fake.NewSimpleClientset(objects...)
	argoFakeClient := argoFake.NewSimpleClientset()

	return &Client{
		Interface:        k8sFake,
		DB:               NewDB(db),
		argoprojV1alpha1: argoFakeClient.ArgoprojV1alpha1(),
	}
}

func DefaultTestClient() *Client {
	return NewTestClient(database, mockSystemConfigMap, mockSystemSecret)
}

func clearDatabase(t *testing.T) {
	// We do not delete from goose_db_version as we need it to mark the migrations as ran.
	query := `
		DELETE FROM workspaces;
		DELETE FROM workflow_executions;
		DELETE FROM cron_workflows;
		DELETE FROM workspace_templates;
		DELETE FROM workflow_templates;
		DELETE FROM workspace_template_versions;
		DELETE FROM workflow_template_versions;
	`

	_, err := database.Exec(query)
	if err != nil {
		t.Fatal(err)
	}
}
