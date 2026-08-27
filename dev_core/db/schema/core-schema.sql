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
  code text NOT NULL,
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
  UNIQUE (tenant_id, code)
);
CREATE INDEX projects_tenant_status_idx ON projects (tenant_id, status, updated_at DESC);

CREATE TABLE scene_provider_state (
  id smallint PRIMARY KEY DEFAULT 1 CHECK (id = 1),
  tenant_id uuid NOT NULL REFERENCES tenants (id) ON DELETE RESTRICT,
  provider text NOT NULL CHECK (provider ~ '^[a-z][a-z0-9_-]{1,31}$'),
  provider_version text NOT NULL,
  created_by uuid NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
  updated_by uuid NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE scene_documents (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
  project_id uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
  scene_id text NOT NULL CHECK (scene_id ~ '^[A-Za-z0-9][A-Za-z0-9_-]{0,127}$'),
  kind text NOT NULL CHECK (kind IN ('2d', '3d')),
  name text NOT NULL CHECK (char_length(name) BETWEEN 1 AND 200),
  provider text NOT NULL CHECK (provider ~ '^[a-z][a-z0-9_-]{1,31}$'),
  entry_path text NOT NULL,
  public_contract jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(public_contract) = 'object'),
  provider_contract jsonb NOT NULL DEFAULT '{"description":"","parameters":[],"events":[],"commands":[]}'::jsonb CHECK (jsonb_typeof(provider_contract) = 'object'),
  datapoint_refs jsonb NOT NULL DEFAULT '[]'::jsonb CHECK (jsonb_typeof(datapoint_refs) = 'array'),
  current_revision bigint NOT NULL DEFAULT 0 CHECK (current_revision >= 0),
  draft_version bigint NOT NULL DEFAULT 0 CHECK (draft_version >= 0),
  committed_draft_version bigint NOT NULL DEFAULT 0 CHECK (committed_draft_version >= 0 AND committed_draft_version <= draft_version),
  deleted_at timestamptz,
  created_by uuid NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
  updated_by uuid NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (project_id, kind, scene_id),
  UNIQUE (id, tenant_id),
  UNIQUE (id, project_id),
  CHECK (entry_path <> '' AND entry_path !~ '(^|/)\.\.(/|$)' AND left(entry_path, 1) <> '/')
);
CREATE INDEX scene_documents_project_idx ON scene_documents (project_id, kind, updated_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX scene_documents_uncommitted_idx ON scene_documents (project_id) WHERE deleted_at IS NULL AND draft_version <> committed_draft_version;
CREATE UNIQUE INDEX scene_documents_project_name_active_uidx ON scene_documents (project_id, lower(name)) WHERE deleted_at IS NULL;

CREATE TABLE scene_content_objects (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
  provider text NOT NULL CHECK (provider ~ '^[a-z][a-z0-9_-]{1,31}$'),
  content_hash text NOT NULL CHECK (content_hash ~ '^[0-9a-f]{64}$'),
  content_size bigint NOT NULL CHECK (content_size >= 0),
  content_type text NOT NULL,
  json_content jsonb,
  object_key text,
  orphaned_at timestamptz,
  created_by uuid NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
  updated_by uuid NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, provider, content_hash),
  UNIQUE (id, tenant_id),
  CHECK ((json_content IS NOT NULL AND object_key IS NULL) OR (json_content IS NULL AND object_key IS NOT NULL)),
  CHECK (object_key IS NULL OR object_key <> '')
);
CREATE INDEX scene_content_objects_orphan_idx ON scene_content_objects (orphaned_at) WHERE orphaned_at IS NOT NULL;

CREATE TABLE scene_assets (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
  project_id uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
  provider text NOT NULL CHECK (provider ~ '^[a-z][a-z0-9_-]{1,31}$'),
  asset_type text NOT NULL CHECK (asset_type IN ('image', 'font', 'model', 'material', 'symbol', 'component')),
  compatible_kind text NOT NULL CHECK (compatible_kind IN ('2d', '3d', 'both')),
  name text NOT NULL CHECK (char_length(name) BETWEEN 1 AND 200),
  entry_path text NOT NULL,
  current_generation bigint NOT NULL DEFAULT 0 CHECK (current_generation >= 0),
  draft_version bigint NOT NULL DEFAULT 0 CHECK (draft_version >= 0),
  committed_draft_version bigint NOT NULL DEFAULT 0 CHECK (committed_draft_version >= 0 AND committed_draft_version <= draft_version),
  archived_at timestamptz,
  created_by uuid NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
  updated_by uuid NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (id, tenant_id),
  UNIQUE (id, project_id),
  CHECK (entry_path <> '' AND entry_path !~ '(^|/)\.\.(/|$)' AND left(entry_path, 1) <> '/')
);
CREATE UNIQUE INDEX scene_assets_active_name_uidx
  ON scene_assets (project_id, provider, asset_type, lower(name)) WHERE archived_at IS NULL;
CREATE INDEX scene_assets_project_idx
  ON scene_assets (project_id, provider, asset_type, updated_at DESC) WHERE archived_at IS NULL;

CREATE TABLE scene_asset_generations (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
  project_id uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
  asset_id uuid NOT NULL,
  generation bigint NOT NULL CHECK (generation > 0),
  entry_path text NOT NULL,
  root_hash text NOT NULL CHECK (root_hash ~ '^[0-9a-f]{64}$'),
  manifest jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(manifest) = 'object'),
  thumbnail_content_object_id uuid,
  created_by uuid NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
  updated_by uuid NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (asset_id, generation),
  UNIQUE (id, tenant_id),
  UNIQUE (id, tenant_id, asset_id),
  FOREIGN KEY (asset_id, tenant_id) REFERENCES scene_assets (id, tenant_id) ON DELETE CASCADE,
  FOREIGN KEY (asset_id, project_id) REFERENCES scene_assets (id, project_id) ON DELETE CASCADE,
  FOREIGN KEY (thumbnail_content_object_id, tenant_id) REFERENCES scene_content_objects (id, tenant_id) ON DELETE RESTRICT,
  CHECK (entry_path <> '' AND entry_path !~ '(^|/)\.\.(/|$)' AND left(entry_path, 1) <> '/')
);
CREATE INDEX scene_asset_generations_asset_idx ON scene_asset_generations (asset_id, generation DESC);

CREATE TABLE scene_asset_generation_files (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
  project_id uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
  asset_id uuid NOT NULL,
  generation_id uuid NOT NULL,
  logical_path text NOT NULL,
  file_role text NOT NULL DEFAULT 'dependency' CHECK (file_role IN ('entry', 'dependency')),
  content_object_id uuid NOT NULL,
  content_hash text NOT NULL CHECK (content_hash ~ '^[0-9a-f]{64}$'),
  created_by uuid NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
  updated_by uuid NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (generation_id, logical_path),
  FOREIGN KEY (asset_id, tenant_id) REFERENCES scene_assets (id, tenant_id) ON DELETE CASCADE,
  FOREIGN KEY (generation_id, tenant_id) REFERENCES scene_asset_generations (id, tenant_id) ON DELETE CASCADE,
  FOREIGN KEY (content_object_id, tenant_id) REFERENCES scene_content_objects (id, tenant_id) ON DELETE RESTRICT,
  CHECK (logical_path <> '' AND logical_path !~ '(^|/)\.\.(/|$)' AND left(logical_path, 1) <> '/')
);
CREATE INDEX scene_asset_generation_files_content_idx ON scene_asset_generation_files (content_object_id);

CREATE TABLE scene_asset_draft_files (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
  project_id uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
  asset_id uuid NOT NULL,
  logical_path text NOT NULL,
  content_object_id uuid NOT NULL,
  content_hash text NOT NULL CHECK (content_hash ~ '^[0-9a-f]{64}$'),
  created_by uuid NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
  updated_by uuid NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (asset_id, logical_path),
  FOREIGN KEY (asset_id, tenant_id) REFERENCES scene_assets (id, tenant_id) ON DELETE CASCADE,
  FOREIGN KEY (content_object_id, tenant_id) REFERENCES scene_content_objects (id, tenant_id) ON DELETE RESTRICT,
  CHECK (logical_path <> '' AND logical_path !~ '(^|/)\.\.(/|$)' AND left(logical_path, 1) <> '/')
);
CREATE INDEX scene_asset_draft_files_content_idx ON scene_asset_draft_files (content_object_id);

CREATE TABLE scene_file_nodes (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
  project_id uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
  scene_document_id uuid NOT NULL,
  provider text NOT NULL CHECK (provider ~ '^[a-z][a-z0-9_-]{1,31}$'),
  logical_path text NOT NULL,
  parent_path text NOT NULL DEFAULT '',
  node_type text NOT NULL CHECK (node_type IN ('file', 'directory')),
  content_object_id uuid,
  content_hash text,
  deleted_at timestamptz,
  created_by uuid NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
  updated_by uuid NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (scene_document_id, logical_path),
  FOREIGN KEY (scene_document_id, tenant_id) REFERENCES scene_documents (id, tenant_id) ON DELETE CASCADE,
  FOREIGN KEY (scene_document_id, project_id) REFERENCES scene_documents (id, project_id) ON DELETE CASCADE,
  FOREIGN KEY (content_object_id, tenant_id) REFERENCES scene_content_objects (id, tenant_id) ON DELETE RESTRICT,
  CHECK (logical_path <> '' AND logical_path !~ '(^|/)\.\.(/|$)' AND left(logical_path, 1) <> '/'),
  CHECK (parent_path = '' OR (parent_path !~ '(^|/)\.\.(/|$)' AND left(parent_path, 1) <> '/')),
  CHECK ((deleted_at IS NOT NULL AND content_object_id IS NULL AND content_hash IS NULL)
    OR (deleted_at IS NULL AND node_type = 'directory' AND content_object_id IS NULL AND content_hash IS NULL)
    OR (deleted_at IS NULL AND node_type = 'file' AND content_object_id IS NOT NULL AND content_hash ~ '^[0-9a-f]{64}$'))
);
CREATE INDEX scene_file_nodes_directory_idx ON scene_file_nodes (scene_document_id, parent_path, logical_path) WHERE deleted_at IS NULL;
CREATE INDEX scene_file_nodes_content_idx ON scene_file_nodes (content_object_id) WHERE deleted_at IS NULL;

CREATE TABLE scene_asset_bindings (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
  project_id uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
  scene_document_id uuid NOT NULL,
  asset_id uuid NOT NULL,
  generation_id uuid NOT NULL,
  mount_path text NOT NULL,
  created_by uuid NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
  updated_by uuid NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (scene_document_id, asset_id),
  UNIQUE (scene_document_id, mount_path),
  FOREIGN KEY (scene_document_id, tenant_id) REFERENCES scene_documents (id, tenant_id) ON DELETE CASCADE,
  FOREIGN KEY (scene_document_id, project_id) REFERENCES scene_documents (id, project_id) ON DELETE CASCADE,
  FOREIGN KEY (asset_id, tenant_id) REFERENCES scene_assets (id, tenant_id) ON DELETE RESTRICT,
  FOREIGN KEY (generation_id, tenant_id, asset_id) REFERENCES scene_asset_generations (id, tenant_id, asset_id) ON DELETE RESTRICT,
  CHECK (mount_path <> '' AND mount_path !~ '(^|/)\.\.(/|$)' AND left(mount_path, 1) <> '/')
);
CREATE INDEX scene_asset_bindings_asset_idx ON scene_asset_bindings (asset_id, generation_id);

CREATE TABLE scene_revisions (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
  project_id uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
  scene_document_id uuid NOT NULL,
  revision bigint NOT NULL CHECK (revision > 0),
  draft_version bigint NOT NULL CHECK (draft_version >= 0),
  provider text NOT NULL,
  provider_version text NOT NULL,
  entry_path text NOT NULL,
  public_contract jsonb NOT NULL CHECK (jsonb_typeof(public_contract) = 'object'),
  datapoint_refs jsonb NOT NULL CHECK (jsonb_typeof(datapoint_refs) = 'array'),
  root_hash text NOT NULL CHECK (root_hash ~ '^[0-9a-f]{64}$'),
  created_by uuid NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
  updated_by uuid NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (scene_document_id, revision),
  UNIQUE (id, tenant_id),
  FOREIGN KEY (scene_document_id, tenant_id) REFERENCES scene_documents (id, tenant_id) ON DELETE CASCADE,
  FOREIGN KEY (scene_document_id, project_id) REFERENCES scene_documents (id, project_id) ON DELETE CASCADE
);
CREATE INDEX scene_revisions_scene_idx ON scene_revisions (scene_document_id, revision DESC);

CREATE TABLE scene_revision_files (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
  project_id uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
  scene_document_id uuid NOT NULL,
  revision_id uuid NOT NULL,
  logical_path text NOT NULL,
  content_object_id uuid NOT NULL,
  content_hash text NOT NULL CHECK (content_hash ~ '^[0-9a-f]{64}$'),
  created_by uuid NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
  updated_by uuid NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (revision_id, logical_path),
  FOREIGN KEY (scene_document_id, tenant_id) REFERENCES scene_documents (id, tenant_id) ON DELETE CASCADE,
  FOREIGN KEY (revision_id, tenant_id) REFERENCES scene_revisions (id, tenant_id) ON DELETE CASCADE,
  FOREIGN KEY (content_object_id, tenant_id) REFERENCES scene_content_objects (id, tenant_id) ON DELETE RESTRICT,
  CHECK (logical_path <> '' AND logical_path !~ '(^|/)\.\.(/|$)' AND left(logical_path, 1) <> '/')
);
CREATE INDEX scene_revision_files_content_idx ON scene_revision_files (content_object_id);

CREATE TABLE scene_revision_assets (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
  project_id uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
  scene_document_id uuid NOT NULL,
  revision_id uuid NOT NULL,
  asset_id uuid NOT NULL,
  generation_id uuid NOT NULL,
  mount_path text NOT NULL,
  created_by uuid NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
  updated_by uuid NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (revision_id, asset_id),
  FOREIGN KEY (scene_document_id, tenant_id) REFERENCES scene_documents (id, tenant_id) ON DELETE CASCADE,
  FOREIGN KEY (revision_id, tenant_id) REFERENCES scene_revisions (id, tenant_id) ON DELETE CASCADE,
  FOREIGN KEY (asset_id, tenant_id) REFERENCES scene_assets (id, tenant_id) ON DELETE RESTRICT,
  FOREIGN KEY (generation_id, tenant_id, asset_id) REFERENCES scene_asset_generations (id, tenant_id, asset_id) ON DELETE RESTRICT,
  CHECK (mount_path <> '' AND mount_path !~ '(^|/)\.\.(/|$)' AND left(mount_path, 1) <> '/')
);
CREATE INDEX scene_revision_assets_generation_idx ON scene_revision_assets (generation_id);

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
