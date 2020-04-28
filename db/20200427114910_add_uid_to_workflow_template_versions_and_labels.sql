-- +goose Up
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
ALTER TABLE workflow_template_versions ADD COLUMN uid varchar(36) UNIQUE CHECK(uid <> '');
UPDATE workflow_template_versions SET uid = uuid_generate_v4();
ALTER TABLE workflow_template_versions ALTER COLUMN uid SET NOT NULL;

ALTER TABLE labels ADD COLUMN uid varchar(36) UNIQUE CHECK(uid <> '');
UPDATE labels SET uid = uuid_generate_v4();
ALTER TABLE labels ALTER COLUMN uid SET NOT NULL;

-- +goose Down
ALTER TABLE workflow_template_versions DROP COLUMN uid;
ALTER TABLE labels DROP COLUMN uid;
DROP EXTENSION IF EXISTS "uuid-ossp"