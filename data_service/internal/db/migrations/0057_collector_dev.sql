CREATE TABLE IF NOT EXISTS collector_dev_agents (
    id uuid PRIMARY KEY,
    tenant_id text NOT NULL,
    name text NOT NULL,
    os text NOT NULL,
    arch text NOT NULL,
    version text NOT NULL,
    credential_hash text NOT NULL UNIQUE,
    capabilities jsonb NOT NULL DEFAULT '[]'::jsonb CHECK (jsonb_typeof(capabilities) = 'array'),
    last_seen_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS collector_dev_agents_tenant_updated_idx
    ON collector_dev_agents (tenant_id, updated_at DESC);

CREATE TABLE IF NOT EXISTS collector_dev_registration_codes (
    id uuid PRIMARY KEY,
    tenant_id text NOT NULL,
    code_hash text NOT NULL UNIQUE,
    expires_at timestamptz NOT NULL,
    used_at timestamptz,
    used_by_agent_id uuid REFERENCES collector_dev_agents(id) ON DELETE SET NULL,
    created_by uuid NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS collector_dev_registration_codes_tenant_created_idx
    ON collector_dev_registration_codes (tenant_id, created_at DESC);

CREATE TABLE IF NOT EXISTS collector_dev_tasks (
    id uuid PRIMARY KEY,
    tenant_id text NOT NULL,
    project_id uuid NOT NULL,
    agent_id uuid NOT NULL REFERENCES collector_dev_agents(id) ON DELETE CASCADE,
    operation text NOT NULL CHECK (operation IN ('connection.test', 'opcua.browse', 'opcua.read')),
    status text NOT NULL CHECK (status IN ('queued', 'running', 'succeeded', 'failed', 'cancelled', 'expired')),
    request_payload jsonb NOT NULL CHECK (jsonb_typeof(request_payload) = 'object'),
    result_payload jsonb,
    error_code text,
    error_message text,
    deadline_at timestamptz NOT NULL,
    claimed_at timestamptz,
    finished_at timestamptz,
    created_by uuid NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS collector_dev_tasks_agent_claim_idx
    ON collector_dev_tasks (agent_id, status, created_at)
    WHERE status IN ('queued', 'running');

CREATE INDEX IF NOT EXISTS collector_dev_tasks_project_created_idx
    ON collector_dev_tasks (tenant_id, project_id, created_at DESC);

CREATE INDEX IF NOT EXISTS collector_dev_tasks_finished_idx
    ON collector_dev_tasks (finished_at)
    WHERE finished_at IS NOT NULL;

