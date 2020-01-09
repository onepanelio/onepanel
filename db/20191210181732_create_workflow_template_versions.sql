-- +goose Up
CREATE TABLE workflow_template_versions
(
    id                      serial PRIMARY KEY,
    workflow_template_id    integer NOT NULL REFERENCES workflow_templates ON DELETE CASCADE,
    version                 integer NOT NULL,
    manifest                text NOT NULL,

    -- auditing info
    created_at              timestamp NOT NULL DEFAULT (NOW() at time zone 'utc'),
    modified_at             timestamp
);

-- +goose Down
DROP TABLE workflow_template_versions;