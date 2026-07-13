ALTER TABLE collector_dev_agents
    ADD COLUMN IF NOT EXISTS revoked_at timestamptz;

ALTER TABLE collector_dev_tasks
    DROP CONSTRAINT IF EXISTS collector_dev_tasks_agent_id_fkey;

ALTER TABLE collector_dev_tasks
    ADD CONSTRAINT collector_dev_tasks_agent_id_fkey
    FOREIGN KEY (agent_id) REFERENCES collector_dev_agents(id);

CREATE INDEX IF NOT EXISTS collector_dev_agents_tenant_active_idx
    ON collector_dev_agents (tenant_id, updated_at DESC)
    WHERE revoked_at IS NULL;
