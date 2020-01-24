-- +goose Up
ALTER TABLE workflow_template_versions ADD COLUMN is_latest boolean;

-- Set the latest version flag so only the latest versions are the latest.
UPDATE workflow_template_versions
SET is_latest = false;

UPDATE workflow_template_versions
SET is_latest = true
WHERE id IN (
    SELECT max(id)
    FROM workflow_template_versions
    GROUP BY workflow_template_id, id
);
-- end setting latest version

-- +goose Down
ALTER TABLE workflow_template_versions DROP COLUMN is_latest;
