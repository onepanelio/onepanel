-- +goose Up
ALTER TABLE workflow_template_versions ADD COLUMN parameters JSONB;
UPDATE workflow_template_versions SET parameters = '[]'::JSONB;
ALTER TABLE workflow_template_versions ALTER COLUMN parameters SET NOT NULL;

-- +goose Down
ALTER TABLE workflow_template_versions DROP COLUMN parameters;
