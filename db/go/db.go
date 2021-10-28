package migration

import (
	"errors"
	"fmt"
	sq "github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
	v1 "github.com/onepanelio/core/pkg"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
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
	initialize20200922103448()
	initialize20200929144301()
	initialize20200929153931()
	initialize20201001070806()
	initialize20201016170415()
	initialize20201028145442()
	initialize20201028145443()
	initialize20201031165106()
	initialize20201102104048()
	initialize20201113094916()
	initialize20201115133046()
	initialize20201115134934()
	initialize20201115145814()
	initialize20201130130433()
	initialize20201208155115()
	initialize20201208155805()
	initialize20201209124226()
	initialize20201211161117()
	initialize20201214133458()
	initialize20201221194344()
	initialize20201221195937()
	initialize20201223062947()
	initialize20201223202929()
	initialize20201225172926()
	initialize20201229205644()
	initialize20210107094725()
	initialize20210118175809()
	initialize20210129134326()
	initialize20210129142057()
	initialize20210129152427()
	initialize20210224180017()
	initialize20210323175655()
	initialize20210329171739()
	initialize20210329194731()
	initialize20210414165510()
	initialize20210719190719()
	initialize20211028205201()

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
	artifactRepositoryType, err := client.GetArtifactRepositoryType(namespace.Name)
	if err != nil {
		return err
	}

	replaceMap := map[string]string{
		"{{.ArtifactRepositoryType}}": artifactRepositoryType,
	}

	if workflowTemplate == nil && workspaceTemplate == nil {
		return errors.New("workflow and workspace template cannot both be nil")
	}
	if workflowTemplate != nil {
		workflowTemplate.Manifest = ReplaceMapValues(workflowTemplate.Manifest, replaceMap)
	}
	if workspaceTemplate != nil {
		workspaceTemplate.Manifest = ReplaceMapValues(workspaceTemplate.Manifest, replaceMap)
	}

	return nil
}

// ReplaceMapValues will replace strings that are keys in the input map with their values
// the result is returned
func ReplaceMapValues(value string, replaceMap map[string]string) string {
	replacePairs := make([]string, 0)

	for key, value := range replaceMap {
		replacePairs = append(replacePairs, key)
		replacePairs = append(replacePairs, value)
	}

	return strings.NewReplacer(replacePairs...).
		Replace(value)
}

// ReplaceRuntimeVariablesInManifest will replace any values that are runtime variables
// with the values currently present in the configuration for a given namespace.
// the result is returned
func ReplaceRuntimeVariablesInManifest(client *v1.Client, namespace string, manifest string) (string, error) {
	artifactRepositoryType, err := client.GetArtifactRepositoryType(namespace)
	if err != nil {
		return manifest, err
	}

	sysConfig, err := client.GetSystemConfig()
	if err != nil {
		return manifest, err
	}

	nodePoolOptions, err := sysConfig.NodePoolOptions()
	if err != nil {
		return manifest, err
	}

	if len(nodePoolOptions) == 0 {
		return manifest, fmt.Errorf("no node pool options in the configuration")
	}

	replaceMap := map[string]string{
		"{{.ArtifactRepositoryType}}": artifactRepositoryType,
		"{{.NodePoolLabel}}":          *sysConfig.NodePoolLabel(),
		"{{.DefaultNodePoolOption}}":  nodePoolOptions[0].Value,
	}

	return ReplaceMapValues(manifest, replaceMap), nil
}

// readDataFile returns the contents of a file in the db/data/{path} directory
// path can indicate subdirectories like cvat/20201016170415.yaml
func readDataFile(path string) (string, error) {
	curDir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	finalPath := []string{curDir, "db", "yaml"}

	for _, pathPart := range strings.Split(path, string(os.PathSeparator)) {
		finalPath = append(finalPath, pathPart)
	}

	data, err := ioutil.ReadFile(filepath.Join(finalPath...))
	if err != nil {
		return "", err
	}

	return string(data), nil
}
