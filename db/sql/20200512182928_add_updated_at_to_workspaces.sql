-- +goose Up
ALTER TABLE workspaces ADD COLUMN updated_at timestamp;

-- +goose Down
ALTER TABLE workspaces DROP COLUMN updated_at;