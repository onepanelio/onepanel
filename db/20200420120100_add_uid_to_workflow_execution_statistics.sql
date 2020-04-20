-- +goose Up
-- SQL in this section is executed when the migration is applied.
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
ALTER TABLE workflow_executions ADD COLUMN uid varchar(36) UNIQUE NOT NULL CHECK(uid <> '') DEFAULT uuid_generate_v1();

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
ALTER TABLE workflow_executions DROP COLUMN uid;
DROP EXTENSION IF EXISTS "uuid-ossp";
