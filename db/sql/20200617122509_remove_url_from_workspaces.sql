-- +goose Up
ALTER TABLE workspaces DROP COLUMN url;

-- +goose Down
ALTER TABLE workspaces ADD COLUMN url TEXT;
UPDATE workspaces set url = '';
ALTER TABLE workspaces ALTER COLUMN url SET NOT NULL;
