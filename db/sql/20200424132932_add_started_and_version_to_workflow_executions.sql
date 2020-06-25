-- +goose Up
ALTER TABLE workflow_executions
    ADD COLUMN started_at TIMESTAMP,
    ADD COLUMN workflow_template_version_id INT REFERENCES workflow_template_versions,
    ADD COLUMN phase VARCHAR(50),
    ADD COLUMN cron_workflow_id INT REFERENCES cron_workflows,
    DROP COLUMN failed_at,
    DROP COLUMN workflow_template_id
;

UPDATE workflow_executions
    SET started_at = created_at,
        phase = 'Succeeded'
;

-- +goose Down
ALTER TABLE workflow_executions
    DROP COLUMN started_at,
    DROP COLUMN workflow_template_version_id,
    DROP COLUMN phase,
    DROP COLUMN cron_workflow_id,
    ADD COLUMN failed_at TIMESTAMP,
    ADD COLUMN workflow_template_id INT
;
