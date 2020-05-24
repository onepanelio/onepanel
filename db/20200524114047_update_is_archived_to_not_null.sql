-- +goose Up
ALTER TABLE workflow_templates ALTER COLUMN is_archived SET DEFAULT false;
ALTER TABLE workflow_templates ALTER COLUMN is_archived SET NOT NULL;

ALTER TABLE workspace_templates ALTER COLUMN is_archived SET DEFAULT false;
ALTER TABLE workspace_templates ALTER COLUMN is_archived SET NOT NULL;

-- +goose Down
ALTER TABLE workspace_templates ALTER COLUMN is_archived DROP NOT NULL;
ALTER TABLE workspace_templates ALTER COLUMN is_archived DROP DEFAULT;

ALTER TABLE workflow_templates ALTER COLUMN is_archived DROP NOT NULL;
ALTER TABLE workflow_templates ALTER COLUMN is_archived DROP DEFAULT;
