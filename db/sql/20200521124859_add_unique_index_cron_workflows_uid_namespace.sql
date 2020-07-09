-- +goose Up
-- SQL in this section is executed when the migration is applied.
CREATE UNIQUE INDEX cron_workflow_namespace_uid ON cron_workflows (uid, namespace);
-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
DROP INDEX cron_workflow_namespace_uid;