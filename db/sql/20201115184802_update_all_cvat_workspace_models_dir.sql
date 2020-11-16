-- +goose Up
-- SQL in this section is executed when the migration is applied.
UPDATE workspace_template_versions
SET manifest = REPLACE(manifest, '/cvat/models', '/cvat/data/models' )
WHERE manifest LIKE '%/cvat/models%' AND manifest LIKE '%onepanel/cvat:0.15.0_cvat.1.0.0%';

UPDATE workflow_template_versions
SET manifest = REPLACE(manifest, '/cvat/models', '/cvat/data/models' )
WHERE manifest LIKE '%/cvat/models%' AND manifest LIKE '%onepanel/cvat:0.15.0_cvat.1.0.0%';


-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
UPDATE workspace_template_versions
SET manifest = REPLACE(manifest, '/cvat/data/models', '/cvat/models' )
WHERE manifest LIKE '%/cvat/data/models%' AND manifest LIKE '%onepanel/cvat:0.15.0_cvat.1.0.0%';

UPDATE workflow_template_versions
SET manifest = REPLACE(manifest, '/cvat/data/models', '/cvat/models' )
WHERE manifest LIKE '%/cvat/data/models%' AND manifest LIKE '%onepanel/cvat:0.15.0_cvat.1.0.0%';