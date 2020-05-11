-- +goose Up
ALTER TABLE workflow_templates ADD COLUMN is_system BOOLEAN DEFAULT false;
UPDATE workflow_templates SET is_system = false;
ALTER TABLE workflow_templates ALTER COLUMN is_system SET NOT NULL;

-- +goose Down
ALTER TABLE workflow_templates DROP COLUMN is_system;
