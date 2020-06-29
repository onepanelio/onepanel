-- +goose Up
ALTER TABLE workflow_template_versions DROP COLUMN uid;

-- +goose Down
ALTER TABLE workflow_template_versions ADD COLUMN uid VARCHAR(30);
UPDATE workflow_template_versions SET uid = version::text;