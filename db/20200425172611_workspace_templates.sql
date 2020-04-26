-- +goose Up
CREATE TABLE workspace_templates
(
    id              serial PRIMARY KEY,
    uid             varchar(36) UNIQUE NOT NULL CHECK(uid <> ''),
    name            text NOT NULL CHECK(name <> ''),
    namespace       varchar(36) NOT NULL,
    is_archived     boolean DEFAULT false,

    -- auditing info
    created_at      timestamp NOT NULL DEFAULT (NOW() at time zone 'utc'),
    modified_at     timestamp
);

CREATE UNIQUE INDEX workspace_templates_name_namespace_key ON workspace_templates (name, namespace) WHERE is_archived = false;

-- +goose Down
DROP TABLE workspace_templates;