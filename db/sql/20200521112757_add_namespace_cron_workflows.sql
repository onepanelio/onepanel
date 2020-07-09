-- +goose Up
-- SQL in this section is executed when the migration is applied.
ALTER TABLE cron_workflows ADD COLUMN namespace varchar(30);
UPDATE cron_workflows cwfs
SET namespace = q.namespace
FROM (
         SELECT wft.id, wft.namespace, wtv.id as wtv_id, wtv.workflow_template_id, cw.id
         FROM workflow_templates wft
                  INNER JOIN workflow_template_versions wtv on wft.id = wtv.workflow_template_id
                  INNER JOIN cron_workflows cw on wtv.id = cw.workflow_template_version_id) q
WHERE cwfs.workflow_template_version_id = q.wtv_id;
ALTER TABLE cron_workflows ALTER COLUMN namespace SET NOT NULL;
-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
ALTER TABLE cron_workflows DROP COLUMN namespace;
