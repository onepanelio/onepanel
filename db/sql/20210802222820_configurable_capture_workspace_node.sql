-- +goose Up
-- SQL in this section is executed when the migration is applied.
ALTER TABLE workspaces ADD COLUMN capture_node boolean;
UPDATE workspaces SET capture_node = false;

-- +goose Down
ALTER TABLE workspaces DROP COLUMN capture_node;
