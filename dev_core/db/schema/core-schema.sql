CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE tenants (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  name text NOT NULL,
  code text NOT NULL UNIQUE,
  description text,
  status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'suspended')),
  contact_email text,
  contact_phone text,
  max_users integer NOT NULL DEFAULT 100 CHECK (max_users >= 0),
  max_projects integer NOT NULL DEFAULT 50 CHECK (max_projects >= 0),
  max_storage bigint NOT NULL DEFAULT 10737418240 CHECK (max_storage >= 0),
  used_storage bigint NOT NULL DEFAULT 0 CHECK (used_storage >= 0),
  logo_object_key text,
  login_background_object_key text,
  company_name text,
  company_address text,
  company_phone text,
  company_website text,
  settings jsonb NOT NULL DEFAULT '{}'::jsonb,
  expires_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX tenants_status_idx ON tenants (status);

CREATE TABLE users (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
  username text NOT NULL,
  password_hash text NOT NULL,
  email text,
  phone text,
  full_name text,
  avatar text,
  role text NOT NULL CHECK (role IN ('SUPER_ADMIN', 'SYSTEM_ADMIN', 'PROJECT_ADMIN', 'OPS_ADMIN', 'USER_ADMIN', 'DEVELOPER', 'OPERATOR', 'VIEWER')),
  status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'suspended')),
  preferences jsonb NOT NULL DEFAULT '{}'::jsonb,
  last_login_at timestamptz,
  last_login_ip text,
  password_changed_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, username)
);
CREATE INDEX users_tenant_role_idx ON users (tenant_id, role);
CREATE INDEX users_status_idx ON users (status);

CREATE TABLE refresh_tokens (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
  user_id uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  token_hash text NOT NULL UNIQUE,
  expires_at timestamptz NOT NULL,
  revoked_at timestamptz,
  replaced_by_token_id uuid REFERENCES refresh_tokens (id) ON DELETE SET NULL,
  last_used_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX refresh_tokens_user_idx ON refresh_tokens (user_id, expires_at);

CREATE TABLE tenant_dashboard_notes (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
  content text NOT NULL CHECK (char_length(content) <= 2000),
  created_by uuid NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
  updated_by uuid REFERENCES users (id) ON DELETE SET NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX tenant_dashboard_notes_tenant_idx ON tenant_dashboard_notes (tenant_id, updated_at DESC);

CREATE TABLE projects (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
  name text NOT NULL,
  code text,
  description text,
  icon text,
  workspace_path text NOT NULL UNIQUE,
  status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'archived', 'deleted')),
  visibility text NOT NULL DEFAULT 'private' CHECK (visibility IN ('private', 'internal')),
  created_by uuid NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
  updated_by uuid REFERENCES users (id) ON DELETE SET NULL,
  archived_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE NULLS NOT DISTINCT (tenant_id, code)
);
CREATE INDEX projects_tenant_status_idx ON projects (tenant_id, status, updated_at DESC);

CREATE TABLE project_tags (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
  name text NOT NULL,
  color text,
  description text,
  sort_order integer NOT NULL DEFAULT 0,
  created_by uuid NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, name)
);

CREATE TABLE project_tag_bindings (
  project_id uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
  tag_id uuid NOT NULL REFERENCES project_tags (id) ON DELETE CASCADE,
  created_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (project_id, tag_id)
);

CREATE TABLE project_groups (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
  name text NOT NULL,
  description text,
  sort_order integer NOT NULL DEFAULT 0,
  created_by uuid NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, name)
);

CREATE TABLE project_group_members (
  project_id uuid PRIMARY KEY REFERENCES projects (id) ON DELETE CASCADE,
  group_id uuid NOT NULL REFERENCES project_groups (id) ON DELETE CASCADE,
  sort_order integer NOT NULL DEFAULT 0,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX project_group_members_group_idx ON project_group_members (group_id, sort_order);

CREATE TABLE project_runtime_users (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  project_id uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
  username text NOT NULL,
  password_hash text NOT NULL,
  display_name text,
  email text,
  status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'disabled')),
  is_builtin_admin boolean NOT NULL DEFAULT false,
  last_login_at timestamptz,
  password_changed_at timestamptz,
  created_by uuid NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
  updated_by uuid REFERENCES users (id) ON DELETE SET NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (project_id, username)
);

CREATE TABLE project_roles (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  project_id uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
  code text NOT NULL,
  name text NOT NULL,
  description text,
  status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'disabled')),
  is_builtin boolean NOT NULL DEFAULT false,
  created_by uuid NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
  updated_by uuid REFERENCES users (id) ON DELETE SET NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (project_id, code)
);

CREATE TABLE project_user_role_bindings (
  runtime_user_id uuid NOT NULL REFERENCES project_runtime_users (id) ON DELETE CASCADE,
  role_id uuid NOT NULL REFERENCES project_roles (id) ON DELETE CASCADE,
  created_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (runtime_user_id, role_id)
);

CREATE TABLE project_role_grants (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  project_id uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
  role_id uuid NOT NULL REFERENCES project_roles (id) ON DELETE CASCADE,
  capability text NOT NULL,
  effect text NOT NULL DEFAULT 'allow' CHECK (effect IN ('allow', 'deny')),
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (role_id, capability)
);
CREATE INDEX project_role_grants_project_idx ON project_role_grants (project_id, capability);

CREATE TABLE audit_logs (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid REFERENCES tenants (id) ON DELETE SET NULL,
  user_id uuid REFERENCES users (id) ON DELETE SET NULL,
  level text NOT NULL DEFAULT 'INFO' CHECK (level IN ('DEBUG', 'INFO', 'WARN', 'ERROR')),
  action text,
  resource text,
  resource_id text,
  message text NOT NULL,
  request_id text,
  method text,
  path text,
  result text,
  ip text,
  user_agent text,
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX audit_logs_tenant_time_idx ON audit_logs (tenant_id, created_at DESC);
CREATE INDEX audit_logs_level_time_idx ON audit_logs (level, created_at DESC);

CREATE TABLE application_versions (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
  project_id uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
  version text NOT NULL,
  name text,
  description text,
  status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'building', 'ready', 'failed', 'deleted')),
  source_hash text,
  artifact_bucket text,
  artifact_key text,
  artifact_hash text,
  artifact_size bigint,
  manifest jsonb NOT NULL DEFAULT '{}'::jsonb,
  build_log text,
  error_message text,
  created_by uuid NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
  completed_at timestamptz,
  deleted_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (project_id, version)
);
CREATE INDEX application_versions_project_idx ON application_versions (project_id, created_at DESC);

CREATE TABLE nodes (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid REFERENCES tenants (id) ON DELETE CASCADE,
  name text NOT NULL,
  code text,
  node_type text NOT NULL DEFAULT 'edge',
  status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'online', 'offline', 'rejected', 'disabled')),
  approval_status text NOT NULL DEFAULT 'pending' CHECK (approval_status IN ('pending', 'approved', 'rejected')),
  registration_token_hash text,
  agent_version text,
  os text,
  architecture text,
  hostname text,
  ip_address text,
  capabilities jsonb NOT NULL DEFAULT '{}'::jsonb,
  metrics jsonb NOT NULL DEFAULT '{}'::jsonb,
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  last_heartbeat_at timestamptz,
  approved_at timestamptz,
  approved_by uuid REFERENCES users (id) ON DELETE SET NULL,
  rejected_at timestamptz,
  rejected_by uuid REFERENCES users (id) ON DELETE SET NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX nodes_tenant_status_idx ON nodes (tenant_id, status, updated_at DESC);

CREATE TABLE node_deployments (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
  node_id uuid NOT NULL REFERENCES nodes (id) ON DELETE CASCADE,
  project_id uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
  application_version_id uuid REFERENCES application_versions (id) ON DELETE SET NULL,
  version text,
  mode text NOT NULL CHECK (mode IN ('development', 'release')),
  status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'deploying', 'running', 'stopped', 'failed', 'removed')),
  runtime_config jsonb NOT NULL DEFAULT '{}'::jsonb,
  runtime_metrics jsonb NOT NULL DEFAULT '{}'::jsonb,
  deploy_log text,
  error_message text,
  deployed_by uuid NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
  deployed_at timestamptz,
  started_at timestamptz,
  stopped_at timestamptz,
  deleted_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX node_deployments_project_idx ON node_deployments (project_id, created_at DESC);
CREATE INDEX node_deployments_node_idx ON node_deployments (node_id, created_at DESC);
CREATE UNIQUE INDEX node_deployments_active_uq ON node_deployments (node_id, project_id)
  WHERE deleted_at IS NULL AND status IN ('pending', 'deploying', 'running');

CREATE TABLE node_commands (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
  node_id uuid NOT NULL REFERENCES nodes (id) ON DELETE CASCADE,
  node_deployment_id uuid NOT NULL REFERENCES node_deployments (id) ON DELETE CASCADE,
  project_id uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
  command_type text NOT NULL CHECK (command_type IN ('deploy', 'start', 'stop', 'restart', 'remove')),
  status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'issued', 'acknowledged', 'completed', 'failed', 'dead_letter')),
  payload jsonb NOT NULL DEFAULT '{}'::jsonb,
  attempts integer NOT NULL DEFAULT 0,
  max_attempts integer NOT NULL DEFAULT 3,
  timeout_seconds integer NOT NULL DEFAULT 30,
  requested_at timestamptz NOT NULL DEFAULT now(),
  issued_at timestamptz,
  acknowledged_at timestamptz,
  completed_at timestamptz,
  last_error text,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX node_commands_pending_idx ON node_commands (node_id, status, requested_at);
