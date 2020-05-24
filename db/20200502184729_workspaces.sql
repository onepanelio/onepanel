-- +goose Up
CREATE TABLE workspaces
(
    id                          serial PRIMARY KEY,
    uid                         varchar(30) NOT NULL CHECK(uid <> ''),
    name                        varchar(30) NOT NULL CHECK(name <> ''),
    namespace                   varchar(30) NOT NULL,
    phase                       varchar(50) NOT NULL,
    parameters                  jsonb NOT NULL,

    workspace_template_id       integer NOT NULL REFERENCES workspace_templates ON DELETE CASCADE,
    workspace_template_version  integer NOT NULL,

    started_at                  timestamp,
    paused_at                   timestamp,
    terminated_at               timestamp,

    -- auditing info
    created_at                  timestamp NOT NULL DEFAULT (NOW() at time zone 'utc'),
    modified_at                 timestamp
);

CREATE UNIQUE INDEX workspaces_name_namespace_key ON workspaces (name, namespace) WHERE phase <> 'Terminated';
CREATE UNIQUE INDEX workspaces_uid_namespace_key ON workspaces (uid, namespace) WHERE phase <> 'Terminated';

-- +goose Down
DROP TABLE workspaces;
