-- +goose Up
-- SQL in this section is executed when the migration is applied.
ALTER TABLE cron_workflows ADD COLUMN is_archived BOOL DEFAULT false NOT NULL;
ALTER TABLE workflow_executions ADD COLUMN is_archived BOOL DEFAULT false NOT NULL;
-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
ALTER TABLE cron_workflows DROP COLUMN is_archived;
ALTER TABLE workflow_executions DROP COLUMN is_archived;