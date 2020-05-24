-- +goose Up
-- +goose StatementBegin
CREATE TABLE workflow_executions
(
    id                   serial PRIMARY KEY,
    uid                  varchar(30) UNIQUE NOT NULL CHECK(uid <> ''),
    workflow_template_id integer     NOT NULL REFERENCES workflow_templates ON DELETE CASCADE,
    name                 text        NOT NULL CHECK (name <> ''),
    namespace            varchar(30) NOT NULL,

    -- auditing info
    created_at           timestamp   NOT NULL DEFAULT (NOW() at time zone 'utc'),
    finished_at          timestamp            DEFAULT NULL,
    failed_at            timestamp            DEFAULT NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE workflow_executions;
-- +goose StatementEnd
