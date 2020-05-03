-- +goose Up
CREATE TABLE workspaces
(
    id                      serial PRIMARY KEY,
    uid                     varchar(36) UNIQUE NOT NULL CHECK(uid <> ''),
    name                    text NOT NULL CHECK(name <> ''),
    namespace               varchar(36) NOT NULL,

    workspace_template_id   integer NOT NULL REFERENCES workspace_templates ON DELETE CASCADE,
    workflow_execution_id   integer NOT NULL REFERENCES workflow_executions ON DELETE CASCADE,

    -- auditing info
    created_at              timestamp NOT NULL DEFAULT (NOW() at time zone 'utc'),
    modified_at             timestamp
);

-- +goose Down
DROP TABLE workspaces;