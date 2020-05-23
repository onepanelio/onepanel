-- +goose Up
ALTER TABLE cron_workflows ALTER COLUMN uid TYPE varchar(63);
ALTER TABLE cron_workflows ALTER COLUMN name TYPE varchar(63);

-- +goose Down
ALTER TABLE cron_workflows ALTER COLUMN uid TYPE varchar(30);
ALTER TABLE cron_workflows ALTER COLUMN name TYPE varchar(30);