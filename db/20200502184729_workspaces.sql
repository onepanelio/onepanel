-- +goose Up
CREATE TABLE workspaces
(
    id                          serial PRIMARY KEY,
    uid                         varchar(36) UNIQUE NOT NULL CHECK(uid <> ''),
    name                        text NOT NULL CHECK(name <> ''),
    namespace                   varchar(36) NOT NULL,
    phase                       varchar(50) NOT NULL,
    parameters                  jsonb NOT NULL,

    workspace_template_id       integer NOT NULL REFERENCES workspace_templates ON DELETE CASCADE,
    workspace_template_version  integer NOT NULL,

    started_at                  timestamp,
    running_at                  timestamp,
    paused_at                   timestamp,
    terminated_at               timestamp,

    -- auditing info
    created_at                  timestamp NOT NULL DEFAULT (NOW() at time zone 'utc'),
    modified_at                 timestamp
);

-- +goose Down
DROP TABLE workspaces;