-- +goose Up
ALTER TABLE workflow_templates ADD COLUMN is_archived boolean DEFAULT false;

-- +goose Down
ALTER TABLE workflow_templates DROP COLUMN is_archived;
