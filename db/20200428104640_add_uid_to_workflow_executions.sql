-- +goose Up
ALTER TABLE workflow_executions ADD COLUMN uid varchar(30) UNIQUE CHECK(uid <> '');
UPDATE workflow_executions SET uid = uuid_generate_v4();
ALTER TABLE workflow_executions ALTER COLUMN uid SET NOT NULL;

-- +goose Down
ALTER TABLE workflow_executions DROP COLUMN uid;
