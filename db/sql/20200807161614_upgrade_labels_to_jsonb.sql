-- +goose Up
-- SQL in this section is executed when the migration is applied.
ALTER TABLE workflow_templates ADD COLUMN labels JSONB DEFAULT '{}'::JSONB;
ALTER TABLE workflow_template_versions ADD COLUMN labels JSONB DEFAULT '{}'::JSONB;
ALTER TABLE workflow_executions ADD COLUMN labels JSONB DEFAULT '{}'::JSONB;
ALTER TABLE workspaces ADD COLUMN labels JSONB DEFAULT '{}'::JSONB;
ALTER TABLE workspace_templates ADD COLUMN labels JSONB DEFAULT '{}'::JSONB;
ALTER TABLE workspace_template_versions ADD COLUMN labels JSONB DEFAULT '{}'::JSONB;


-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
ALTER TABLE workspace_template_versions DROP COLUMN labels;
ALTER TABLE workspace_templates DROP COLUMN labels;
ALTER TABLE workspaces DROP COLUMN labels;
ALTER TABLE workflow_executions DROP COLUMN labels;
ALTER TABLE workflow_template_versions DROP COLUMN labels;
ALTER TABLE workflow_templates DROP COLUMN labels;
