-- +goose Up
-- SQL in this section is executed when the migration is applied.
ALTER TABLE workflow_templates ADD COLUMN labels JSONB DEFAULT '{}'::JSONB;
ALTER TABLE workflow_template_versions ADD COLUMN labels JSONB DEFAULT '{}'::JSONB;
ALTER TABLE workflow_executions ADD COLUMN labels JSONB DEFAULT '{}'::JSONB;
ALTER TABLE workspaces ADD COLUMN labels JSONB DEFAULT '{}'::JSONB;
ALTER TABLE workspace_templates ADD COLUMN labels JSONB DEFAULT '{}'::JSONB;
ALTER TABLE workspace_template_versions ADD COLUMN labels JSONB DEFAULT '{}'::JSONB;
ALTER TABLE cron_workflows ADD COLUMN labels JSONB DEFAULT '{}'::JSONB;

-- We take the old labels and put them into the new jsonb columns
UPDATE workflow_templates wt
SET labels =
        (
            SELECT jsonb_object_agg(key, value)
            FROM labels l
            WHERE resource = 'workflow_template'
              AND wt.id = l.resource_id
        )
;
UPDATE workflow_templates SET labels = '{}'::jsonb WHERE labels IS NULL;

UPDATE workflow_template_versions wtv
SET labels =
        (
            SELECT jsonb_object_agg(key, value)
            FROM labels l
            WHERE resource = 'workflow_template_version'
              AND wtv.id = l.resource_id
        )
;
UPDATE workflow_template_versions SET labels = '{}'::jsonb WHERE labels IS NULL;

UPDATE workflow_executions we
SET labels =
    (
        SELECT jsonb_object_agg(key, value)
        FROM labels l
        WHERE resource = 'workflow_execution'
          AND we.id = l.resource_id
    )
;
UPDATE workflow_executions SET labels = '{}'::jsonb WHERE labels IS NULL;

UPDATE workspaces w
SET labels =
        (
            SELECT jsonb_object_agg(key, value)
            FROM labels l
            WHERE resource = 'workspace'
              AND w.id = l.resource_id
        )
;
UPDATE workspaces SET labels = '{}'::jsonb WHERE labels IS NULL;

UPDATE workspace_templates wt
SET labels =
        (
            SELECT jsonb_object_agg(key, value)
            FROM labels l
            WHERE resource = 'workspace_template'
              AND wt.id = l.resource_id
        )
;
UPDATE workspace_templates SET labels = '{}'::jsonb WHERE labels IS NULL;

UPDATE workspace_template_versions wtv
SET labels =
        (
            SELECT jsonb_object_agg(key, value)
            FROM labels l
            WHERE resource = 'workspace_template_version'
              AND wtv.id = l.resource_id
        )
;
UPDATE workspace_template_versions SET labels = '{}'::jsonb WHERE labels IS NULL;

UPDATE cron_workflows cw
SET labels =
        (
            SELECT jsonb_object_agg(key, value)
            FROM labels l
            WHERE resource = 'cron_workflow'
              AND cw.id = l.resource_id
        )
;
UPDATE cron_workflows SET labels = '{}'::jsonb WHERE labels IS NULL;

DROP table labels;

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.

-- We take the jsonb column labels and put them back into a separate labels table
CREATE TABLE labels
(
    id serial PRIMARY KEY,
    key character varying(255),
    value character varying(255),
    resource character varying(255),
    resource_id integer,
    created_at timestamp without time zone NOT NULL DEFAULT timezone('utc'::text, now())
);

INSERT INTO labels(key, value, resource, resource_id, created_at)
SELECT key, value, 'cron_workflow', id, now()
FROM (
         SELECT wt1.id as id, wt1.key as key, wt2.labels->>wt1.key as value
         FROM cron_workflows wt2
                  JOIN (
             SELECT wt.id, jsonb_object_keys(wt.labels) as key
             FROM cron_workflows wt
         ) wt1 on wt2.id = wt1.id
     ) subquery
;

INSERT INTO labels(key, value, resource, resource_id, created_at)
SELECT key, value, 'workflow_template_version', id, now()
FROM (
         SELECT wt1.id as id, wt1.key as key, wt2.labels->>wt1.key as value
         FROM workflow_template_versions wt2
                  JOIN (
             SELECT wt.id, jsonb_object_keys(wt.labels) as key
             FROM workflow_template_versions wt
         ) wt1 on wt2.id = wt1.id
     ) subquery
;

INSERT INTO labels(key, value, resource, resource_id, created_at)
SELECT key, value, 'workspace_template', id, now()
FROM (
         SELECT wt1.id as id, wt1.key as key, wt2.labels->>wt1.key as value
         FROM workspace_templates wt2
                  JOIN (
             SELECT wt.id, jsonb_object_keys(wt.labels) as key
             FROM workspace_templates wt
         ) wt1 on wt2.id = wt1.id
     ) subquery
;

INSERT INTO labels(key, value, resource, resource_id, created_at)
SELECT key, value, 'workspace', id, now()
FROM (
         SELECT wt1.id as id, wt1.key as key, wt2.labels->>wt1.key as value
         FROM workspaces wt2
                  JOIN (
             SELECT wt.id, jsonb_object_keys(wt.labels) as key
             FROM workspaces wt
         ) wt1 on wt2.id = wt1.id
     ) subquery
;

INSERT INTO labels(key, value, resource, resource_id, created_at)
SELECT key, value, 'workflow_execution', id, now()
FROM (
         SELECT wt1.id as id, wt1.key as key, wt2.labels->>wt1.key as value
         FROM workflow_executions wt2
                  JOIN (
             SELECT wt.id, jsonb_object_keys(wt.labels) as key
             FROM workflow_executions wt
         ) wt1 on wt2.id = wt1.id
     ) subquery
;

INSERT INTO labels(key, value, resource, resource_id, created_at)
SELECT key, value, 'workflow_template_version', id, now()
FROM (
         SELECT wt1.id as id, wt1.key as key, wt2.labels->>wt1.key as value
         FROM workflow_template_versions wt2
                  JOIN (
             SELECT wt.id, jsonb_object_keys(wt.labels) as key
             FROM workflow_template_versions wt
         ) wt1 on wt2.id = wt1.id
     ) subquery
;

INSERT INTO labels(key, value, resource, resource_id, created_at)
SELECT key, value, 'workflow_template', id, now()
FROM (
         SELECT wt1.id as id, wt1.key as key, wt2.labels->>wt1.key as value
         FROM workflow_templates wt2
                  JOIN (
             SELECT wt.id, jsonb_object_keys(wt.labels) as key
             FROM workflow_templates wt
         ) wt1 on wt2.id = wt1.id
     ) subquery
;

ALTER TABLE cron_workflows DROP COLUMN labels;
ALTER TABLE workspace_template_versions DROP COLUMN labels;
ALTER TABLE workspace_templates DROP COLUMN labels;
ALTER TABLE workspaces DROP COLUMN labels;
ALTER TABLE workflow_executions DROP COLUMN labels;
ALTER TABLE workflow_template_versions DROP COLUMN labels;
ALTER TABLE workflow_templates DROP COLUMN labels;
