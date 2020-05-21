-- +goose Up
-- SQL in this section is executed when the migration is applied.
ALTER TABLE cron_workflows ADD COLUMN is_archived BOOL;
ALTER TABLE workflow_executions ADD COLUMN is_archived BOOL;
-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
ALTER TABLE cron_workflows ADD COLUMN is_archived BOOL;
ALTER TABLE workflow_executions ADD COLUMN is_archived BOOL;