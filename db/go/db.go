package migration

import (
	"errors"
	sq "github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
	v1 "github.com/onepanelio/core/pkg"
	"log"
	"strings"
)

// initializedMigrations is used to keep track of which migrations have been initialized.
// if they are initialized more than once, goose panics.
var initializedMigrations = make(map[int]bool)

// sqlRanMigrations keeps track of all the sql migrations that have been run.
// we need to know this because in an older version some go migrations ran alongside sql.
// So if they have already been run, we can't run them again.
var sqlRanMigrations = make(map[uint64]bool)

// migrationHasAlreadyBeenRun returns true if the migration has already been run in sql
// see sqlRanMigrations var
func migrationHasAlreadyBeenRun(version int) bool {
	_, ok := sqlRanMigrations[uint64(version)]
	return ok
}

// Initialize sets up the go migrations.
func Initialize() {
	client, err := getClient()
	if err != nil {
		log.Fatalf("unable to get client for go migrations: %v", err)
	}

	migrationsRan, err := getRanSQLMigrations(client)
	if err != nil {
		log.Fatalf("Unable to get already run sql migrations: %v", err)
	}
	sqlRanMigrations = migrationsRan

	initialize20200525160514()
	initialize20200528140124()
	initialize20200605090509()
	initialize20200605090535()
	initialize20200626113635()
	initialize20200704151301()
	initialize20200724220450()
	initialize20200727144157()
	initialize20200728190804()
	initialize20200812104328()
	initialize20200812113316()
	initialize20200814160856()
	initialize20200821162630()
	initialize20200824095513()
	initialize20200824101019()
	initialize20200824101905()
	initialize20200825154403()
	initialize20200826185926()

	if err := client.DB.Close(); err != nil {
		log.Printf("[error] closing db %v", err)
	}
}

func getClient() (*v1.Client, error) {
	kubeConfig := v1.NewConfig()
	client, err := v1.NewClient(kubeConfig, nil, nil)
	if err != nil {
		return nil, err
	}
	config, err := client.GetSystemConfig()
	if err != nil {
		return nil, err
	}

	dbDriverName, dbDataSourceName := config.DatabaseConnection()
	client.DB = v1.NewDB(sqlx.MustConnect(dbDriverName, dbDataSourceName))

	return client, nil
}

// getRanSQLMigrations returns a map where each key is a sql migration version ran.
func getRanSQLMigrations(client *v1.Client) (map[uint64]bool, error) {
	sb := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query := sb.Select("version_id").
		From("goose_db_version")

	versions := make([]uint64, 0)
	if err := client.Selectx(&versions, query); err != nil {
		return nil, err
	}

	result := make(map[uint64]bool)
	for _, version := range versions {
		result[version] = true
	}

	return result, nil
}

// ReplaceArtifactRepositoryType will look for {{.ArtifactRepositoryType}} in the migration and replace it based on config.
func ReplaceArtifactRepositoryType(client *v1.Client, namespace *v1.Namespace, workflowTemplate *v1.WorkflowTemplate, workspaceTemplate *v1.WorkspaceTemplate) error {
	artifactRepositoryType := "s3"
	nsConfig, err := client.GetNamespaceConfig(namespace.Name)
	if err != nil {
		return err
	}
	if nsConfig.ArtifactRepository.GCS != nil {
		artifactRepositoryType = "gcs"
	}

	if workflowTemplate != nil {
		workflowTemplate.Manifest = strings.NewReplacer(
			"{{.ArtifactRepositoryType}}", artifactRepositoryType).Replace(workflowTemplate.Manifest)
	}
	if workspaceTemplate != nil {
		workspaceTemplate.Manifest = strings.NewReplacer(
			"{{.ArtifactRepositoryType}}", artifactRepositoryType).Replace(workspaceTemplate.Manifest)
	}
	if workflowTemplate == nil && workspaceTemplate == nil {
		return errors.New("workflow and workspace template cannot be nil")
	}

	return nil
}
