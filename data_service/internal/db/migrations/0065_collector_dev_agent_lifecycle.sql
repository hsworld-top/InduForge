DELETE FROM collector_dev_tasks;
DELETE FROM collector_dev_registration_codes;
DELETE FROM collector_dev_agents;

ALTER TABLE collector_dev_agents
    ADD COLUMN IF NOT EXISTS machine_id text;

ALTER TABLE collector_dev_agents
    ADD COLUMN IF NOT EXISTS disconnected_at timestamptz;

ALTER TABLE collector_dev_agents
    ALTER COLUMN machine_id SET NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS collector_dev_agents_tenant_machine_idx
    ON collector_dev_agents (tenant_id, machine_id);

DROP INDEX IF EXISTS collector_dev_agents_tenant_active_idx;
