-- +goose Up
CREATE TABLE workflow_template_versions
(
    id                      serial PRIMARY KEY,
    workflow_template_id    integer NOT NULL REFERENCES workflow_templates ON DELETE CASCADE,
    version                 integer NOT NULL,
    is_latest               boolean NOT NULL,
    manifest                text NOT NULL,

    -- auditing info
    created_at              timestamp NOT NULL DEFAULT (NOW() at time zone 'utc')
);

ALTER TABLE workflow_templates DROP COLUMN versions;

-- +goose Down
ALTER TABLE workflow_templates ADD COLUMN versions INTEGER;
UPDATE workflow_templates SET versions = 1;
ALTER TABLE workflow_templates ALTER COLUMN versions SET NOT NULL;

DROP TABLE workflow_template_versions;

