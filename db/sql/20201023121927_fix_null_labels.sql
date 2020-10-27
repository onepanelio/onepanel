-- +goose Up
-- SQL in this section is executed when the migration is applied.
UPDATE workflow_executions
SET labels = '{}'::jsonb
WHERE labels = 'null'::jsonb;

UPDATE workflow_templates
SET labels = '{}'::jsonb
WHERE labels = 'null'::jsonb;

UPDATE workspace_templates
SET labels = '{}'::jsonb
WHERE labels = 'null'::jsonb;

UPDATE workflow_template_versions
SET labels = '{}'::jsonb
WHERE labels = 'null'::jsonb;

UPDATE workspace_template_Versions
SET labels = '{}'::jsonb
WHERE labels = 'null'::jsonb;

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
