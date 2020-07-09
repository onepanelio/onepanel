-- +goose Up
DROP TABLE workflow_template_versions;

-- +goose Down
CREATE TABLE workflow_template_versions
(
    id                      serial PRIMARY KEY,
    workflow_template_id    integer NOT NULL REFERENCES workflow_templates ON DELETE CASCADE,
    version                 integer NOT NULL,
    is_latest               boolean NOT NULL,
    manifest                text NOT NULL,

    -- auditing info
    created_at              timestamp NOT NULL DEFAULT (NOW() at time zone 'utc'),
    modified_at             timestamp
);

