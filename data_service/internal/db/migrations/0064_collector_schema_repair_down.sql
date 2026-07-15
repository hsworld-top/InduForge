DELETE FROM collector_dev_tasks;

DROP INDEX IF EXISTS collector_dev_tasks_connection_created_idx;

ALTER TABLE collector_dev_tasks
    DROP CONSTRAINT IF EXISTS collector_dev_tasks_operation_check,
    DROP CONSTRAINT IF EXISTS collector_dev_tasks_connection_fkey,
    DROP COLUMN IF EXISTS connection_id;

ALTER TABLE collector_dev_tasks
    ADD CONSTRAINT collector_dev_tasks_operation_check
        CHECK (operation IN ('connection.test', 'opcua.browse', 'opcua.read'));

DROP TABLE IF EXISTS data_collector_import_sessions;
