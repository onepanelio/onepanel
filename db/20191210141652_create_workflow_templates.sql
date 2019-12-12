-- +goose Up
CREATE TABLE workflow_templates
(
    id          serial PRIMARY KEY,
    uid         varchar(36) UNIQUE NOT NULL CHECK(uid <> ''),
    name        text UNIQUE NOT NULL CHECK(name <> ''),

    -- auditing info
    created_at  timestamp NOT NULL DEFAULT (NOW() at time zone 'utc'),
    modified_at timestamp
);

-- +goose Down
DROP TABLE workflow_templates;