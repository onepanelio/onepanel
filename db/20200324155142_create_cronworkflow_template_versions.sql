-- +goose Up
-- SQL in this section is executed when the migration is applied.
CREATE TABLE cron_workflow_template_versions
(
    id                           serial PRIMARY KEY,
    cron_workflow_template_id    integer NOT NULL REFERENCES cron_workflow_templates ON DELETE CASCADE,
    version                      integer NOT NULL,
    manifest                     text NOT NULL,
    is_latest                    boolean DEFAULT false,

    -- auditing info
    created_at              timestamp NOT NULL DEFAULT (NOW() at time zone 'utc'),
    modified_at             timestamp
);

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
DROP TABLE cron_workflow_template_versions;