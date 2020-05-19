-- +goose Up
ALTER TABLE workflow_templates DROP CONSTRAINT IF EXISTS workflow_templates_uid_key;
ALTER TABLE workflow_templates DROP CONSTRAINT IF EXISTS workflow_templates_uid_namespace_key;
CREATE UNIQUE INDEX workflow_templates_name_namespace_key ON workflow_templates (name, namespace) WHERE is_archived = false;
CREATE UNIQUE INDEX workflow_templates_uid_namespace_key ON workflow_templates (uid, namespace) WHERE is_archived = false;

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
DROP INDEX workflow_templates_name_namespace_key;
DROP INDEX workflow_templates_uid_namespace_key;
ALTER TABLE workflow_templates ADD CONSTRAINT workflow_templates_uid_key UNIQUE (uid);
ALTER TABLE  workflow_templates ADD CONSTRAINT  workflow_templates_uid_namespace_key UNIQUE (uid, namespace);