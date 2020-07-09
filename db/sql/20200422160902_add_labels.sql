-- +goose Up
CREATE TABLE labels
(
    id          serial PRIMARY KEY,
    key         varchar(255),
    value       varchar(255),
    resource    varchar(255),
    resource_id INTEGER,

    -- auditing info
    created_at  timestamp NOT NULL DEFAULT (NOW() at time zone 'utc')
);

-- +goose Down
DROP TABLE labels;
