-- +goose Up
ALTER TABLE workflow_templates ADD COLUMN versions INTEGER;
UPDATE workflow_templates SET versions = 1;
ALTER TABLE workflow_templates ALTER COLUMN versions SET NOT NULL;

-- +goose Down
ALTER TABLE workflow_templates DROP COLUMN versions;