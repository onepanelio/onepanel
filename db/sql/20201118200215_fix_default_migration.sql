-- +goose Up
-- SQL in this section is executed when the migration is applied.
UPDATE workflow_executions SET metrics = '[]'::JSONB
WHERE metrics = '{}'::JSONB;

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
UPDATE workflow_executions SET metrics = '{}'::JSONB
WHERE metrics = '[]'::JSONB;