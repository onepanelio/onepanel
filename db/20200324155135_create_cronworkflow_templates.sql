-- +goose Up
-- SQL in this section is executed when the migration is applied.
CREATE TABLE cron_workflow_templates
(
    id          serial PRIMARY KEY,
    uid         varchar(36) UNIQUE NOT NULL CHECK(uid <> ''),
    name        text NOT NULL CHECK(name <> ''),
    namespace   varchar(36) NOT NULL,
    is_archived boolean DEFAULT false,

    -- auditing info
    created_at  timestamp NOT NULL DEFAULT (NOW() at time zone 'utc'),
    modified_at timestamp,

    UNIQUE (name, namespace)
);
-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
DROP TABLE cron_workflow_templates;