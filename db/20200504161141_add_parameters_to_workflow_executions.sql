-- +goose Up
ALTER TABLE workflow_executions ADD COLUMN parameters JSONB;
UPDATE workflow_executions SET parameters = '{}'::JSONB;
ALTER TABLE workflow_executions ALTER COLUMN parameters SET NOT NULL;

-- +goose Down
ALTER TABLE workflow_executions DROP COLUMN parameters;
