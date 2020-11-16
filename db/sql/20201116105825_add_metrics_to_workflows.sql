-- +goose Up
-- SQL in this section is executed when the migration is applied.
ALTER TABLE workflow_executions ADD COLUMN metrics JSONB;
UPDATE workflow_executions SET metrics = '{}'::JSONB;
ALTER TABLE workflow_executions ALTER COLUMN metrics SET NOT NULL;

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
ALTER TABLE workflow_executions DROP COLUMN metrics;