-- +goose Up
ALTER TABLE workflow_template_versions DROP COLUMN manifest;

-- +goose Down
ALTER TABLE workflow_template_versions ADD COLUMN manifest TEXT NOT NULL;
