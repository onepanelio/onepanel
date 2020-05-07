-- +goose Up
ALTER TABLE cron_workflows ADD COLUMN manifest TEXT;
UPDATE cron_workflows SET manifest = '';
ALTER TABLE cron_workflows ALTER manifest SET NOT NULL;

ALTER TABLE cron_workflows
    DROP COLUMN schedule,
    DROP COLUMN timezone,
    DROP COLUMN suspend,
    DROP COLUMN concurrency_policy,
    DROP COLUMN starting_deadline_seconds,
    DROP COLUMN successful_jobs_history_limit,
    DROP COLUMN failed_jobs_history_limit,
    DROP COLUMN workflow_spec
;

-- +goose Down
ALTER TABLE cron_workflows
    ADD COLUMN schedule varchar(255),
    ADD COLUMN timezone varchar(255),
    ADD COLUMN suspend boolean,
    ADD COLUMN concurrency_policy varchar(255),
    ADD COLUMN starting_deadline_seconds INT,
    ADD COLUMN successful_jobs_history_limit INT,
    ADD COLUMN failed_jobs_history_limit INT,
    ADD COLUMN workflow_spec TEXT
;

ALTER TABLE cron_workflows DROP COLUMN manifest;
