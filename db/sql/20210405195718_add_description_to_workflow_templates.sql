-- +goose Up
-- SQL in this section is executed when the migration is applied.
ALTER TABLE workflow_templates ADD COLUMN description TEXT DEFAULT '';

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
ALTER TABLE workflow_templates DROP COLUMN description;