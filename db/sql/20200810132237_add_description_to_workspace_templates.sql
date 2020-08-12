-- +goose Up
-- SQL in this section is executed when the migration is applied.
ALTER TABLE workspace_templates ADD COLUMN description TEXT DEFAULT '';

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
ALTER TABLE workspace_templates DROP COLUMN description;
