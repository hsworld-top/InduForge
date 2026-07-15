DROP INDEX IF EXISTS collector_dev_agents_tenant_machine_idx;

ALTER TABLE collector_dev_agents
    DROP COLUMN IF EXISTS disconnected_at;

ALTER TABLE collector_dev_agents
    DROP COLUMN IF EXISTS machine_id;

CREATE INDEX IF NOT EXISTS collector_dev_agents_tenant_active_idx
    ON collector_dev_agents (tenant_id, updated_at DESC)
    WHERE revoked_at IS NULL;
