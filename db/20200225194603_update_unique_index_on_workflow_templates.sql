-- +goose Up
ALTER TABLE workflow_templates DROP CONSTRAINT workflow_templates_name_namespace_key;
CREATE UNIQUE INDEX workflow_templates_name_namespace_key ON workflow_templates (name, namespace) WHERE is_archived = false;

-- +goose Down
DROP INDEX workflow_templates_name_namespace_key;
ALTER TABLE workflow_templates ADD CONSTRAINT workflow_templates_name_namespace_key UNIQUE (name, namespace);