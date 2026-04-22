ALTER TABLE data_points
    ADD COLUMN IF NOT EXISTS runtime_permissions jsonb NOT NULL DEFAULT '{"write":{"allowRoles":[],"denyRoles":[],"inherit":true}}'::jsonb
    CHECK (jsonb_typeof(runtime_permissions) = 'object');

UPDATE data_points
SET runtime_permissions = '{"write":{"allowRoles":[],"denyRoles":[],"inherit":true}}'::jsonb
WHERE runtime_permissions IS NULL
   OR runtime_permissions = '{}'::jsonb;
