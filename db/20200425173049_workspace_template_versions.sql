-- +goose Up
CREATE TABLE workspace_template_versions
(
    id                              serial PRIMARY KEY,
    workspace_template_id           integer NOT NULL REFERENCES workspace_templates ON DELETE CASCADE,
    version                         integer NOT NULL,
    manifest                        text NOT NULL,
    is_latest                       boolean DEFAULT false,

    -- auditing info
    created_at                      timestamp NOT NULL DEFAULT (NOW() at time zone 'utc'),
    modified_at                     timestamp
);

-- +goose Down
DROP TABLE workspace_template_versions;