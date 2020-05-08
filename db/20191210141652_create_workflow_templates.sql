-- +goose Up
CREATE TABLE workflow_templates
(
    id          serial PRIMARY KEY,
    uid         varchar(30) UNIQUE NOT NULL CHECK(uid <> ''),
    name        text NOT NULL CHECK(name <> ''),
    namespace   varchar(30) NOT NULL,

    -- auditing info
    created_at  timestamp NOT NULL DEFAULT (NOW() at time zone 'utc'),
    modified_at timestamp,

    UNIQUE (uid, namespace)
);

-- +goose Down
DROP TABLE workflow_templates;