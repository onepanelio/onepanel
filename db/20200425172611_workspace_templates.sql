-- +goose Up
CREATE TABLE workspace_templates
(
    id                      serial PRIMARY KEY,
    uid                     varchar(30) NOT NULL CHECK(uid <> ''),
    name                    varchar(30) NOT NULL CHECK(name <> ''),
    namespace               varchar(30) NOT NULL,
    is_archived             boolean DEFAULT false,

    workflow_template_id    integer NOT NULL REFERENCES workflow_templates ON DELETE CASCADE,

    -- auditing info
    created_at              timestamp NOT NULL DEFAULT (NOW() at time zone 'utc'),
    modified_at             timestamp
);

CREATE UNIQUE INDEX workspace_templates_name_namespace_key ON workspace_templates (name, namespace) WHERE is_archived = false;
CREATE UNIQUE INDEX workspace_templates_uid_namespace_key ON workspace_templates (uid, namespace) WHERE is_archived = false;

-- +goose Down
DROP TABLE workspace_templates;
