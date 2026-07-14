ALTER TABLE collector_dev_agents
    ADD COLUMN IF NOT EXISTS last_ip text NOT NULL DEFAULT '';
