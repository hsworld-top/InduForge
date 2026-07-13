DROP INDEX IF EXISTS collector_dev_agents_tenant_active_idx;

ALTER TABLE collector_dev_tasks
    DROP CONSTRAINT IF EXISTS collector_dev_tasks_agent_id_fkey;

ALTER TABLE collector_dev_tasks
    ADD CONSTRAINT collector_dev_tasks_agent_id_fkey
    FOREIGN KEY (agent_id) REFERENCES collector_dev_agents(id) ON DELETE CASCADE;

ALTER TABLE collector_dev_agents
    DROP COLUMN IF EXISTS revoked_at;
