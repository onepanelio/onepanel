-- +goose Up
ALTER TABLE workflow_executions ALTER COLUMN uid TYPE varchar(63);
ALTER TABLE workflow_executions ALTER COLUMN name TYPE varchar(63);

-- +goose Down
ALTER TABLE workflow_executions ALTER COLUMN uid TYPE varchar(30);
ALTER TABLE workflow_executions ALTER COLUMN name TYPE text;