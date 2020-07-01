-- +goose Up
-- SQL in this section is executed when the migration is applied.
ALTER TABLE workflow_template_versions ALTER COLUMN version TYPE BIGINT;
ALTER TABLE workspace_template_versions ALTER COLUMN version TYPE BIGINT;
ALTER TABLE workspaces ALTER COLUMN workspace_template_version TYPE BIGINT;

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
ALTER TABLE workflow_template_versions ALTER COLUMN version TYPE INT;
ALTER TABLE workspace_template_versions ALTER COLUMN version TYPE INT;
ALTER TABLE workspaces ALTER COLUMN workspace_template_version TYPE INT;