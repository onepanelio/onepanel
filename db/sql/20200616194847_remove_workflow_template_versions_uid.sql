-- +goose Up
ALTER TABLE workflow_template_versions DROP COLUMN uid;

-- +goose Down
UPDATE workflow_template_versions SET uid = version::text;