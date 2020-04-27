-- +goose Up
CREATE TABLE cron_workflows
(
    id          serial PRIMARY KEY,
    uid         varchar(36) UNIQUE NOT NULL CHECK(uid <> ''),
    name varchar(255),
    workflow_template_version_id INT REFERENCES workflow_template_versions,
    schedule varchar(255),
    timezone varchar(255),
    suspend boolean,
    concurrency_policy varchar(255),
    starting_deadline_seconds INT,
    successful_jobs_history_limit INT,
    failed_jobs_history_limit INT,
    workflow_spec TEXT,

    -- auditing info
    created_at  timestamp NOT NULL DEFAULT (NOW() at time zone 'utc'),
    modified_at timestamp
);

-- +goose Down
DROP TABLE cron_workflows;
