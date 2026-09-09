CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE tenants (
  initialized boolean NOT NULL DEFAULT false,
  is_default boolean NOT NULL DEFAULT false,
  admin_user_id uuid,
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
  must_change_password boolean NOT NULL DEFAULT false,
  credential_version bigint NOT NULL DEFAULT 0,
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid REFERENCES tenants (id) ON DELETE CASCADE,
  username text NOT NULL,
  password_hash text NOT NULL,
  email text,
  phone text,
  full_name text,
  avatar text,
  role text NOT NULL CHECK (role IN ('SUPER_ADMIN', 'SYSTEM_ADMIN', 'PROJECT_ADMIN', 'OPS_ADMIN')),
  status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'suspended')),
  preferences jsonb NOT NULL DEFAULT '{}'::jsonb,
  last_login_at timestamptz,
  last_login_ip text,
  password_changed_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CHECK ((role = 'SUPER_ADMIN') = (tenant_id IS NULL)),
  UNIQUE (tenant_id, username)
);
CREATE INDEX users_tenant_role_idx ON users (tenant_id, role);
CREATE INDEX users_status_idx ON users (status);
CREATE UNIQUE INDEX tenant_user_username_idx ON users (tenant_id, lower(username)) WHERE tenant_id IS NOT NULL;
CREATE UNIQUE INDEX platform_user_username_idx ON users (lower(username)) WHERE tenant_id IS NULL;

CREATE TABLE refresh_tokens (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid REFERENCES tenants (id) ON DELETE CASCADE,
  user_id uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  token_hash text NOT NULL UNIQUE,
  remember_me boolean NOT NULL DEFAULT false,
  credential_version bigint NOT NULL DEFAULT 0,
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
  authoring_epoch bigint NOT NULL DEFAULT 1 CHECK (authoring_epoch > 0),
  created_by uuid NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
  updated_by uuid REFERENCES users (id) ON DELETE SET NULL,
  archived_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, code),
  UNIQUE (id, tenant_id)
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
  deletion_requested_at timestamptz,
  created_by uuid NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
  updated_by uuid NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (project_id, kind, scene_id),
  UNIQUE (id, tenant_id),
  UNIQUE (id, project_id),
  UNIQUE (id, project_id, tenant_id),
  FOREIGN KEY (project_id, tenant_id) REFERENCES projects (id, tenant_id) ON DELETE CASCADE,
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
  manifest_hash text,
  checksums_hash text,
  signing_key_id text,
  authoring_snapshot_schema text CHECK (authoring_snapshot_schema IS NULL OR authoring_snapshot_schema = 'authoring-snapshot.v1'),
  authoring_snapshot_bucket text,
  authoring_snapshot_key text,
  authoring_snapshot_hash text CHECK (authoring_snapshot_hash IS NULL OR authoring_snapshot_hash ~ '^[0-9a-f]{64}$'),
  authoring_snapshot_cipher_hash text CHECK (authoring_snapshot_cipher_hash IS NULL OR authoring_snapshot_cipher_hash ~ '^[0-9a-f]{64}$'),
  authoring_snapshot_size bigint CHECK (authoring_snapshot_size IS NULL OR authoring_snapshot_size > 0),
  authoring_snapshot_key_id text,
  authoring_project_revision text,
  restorable boolean NOT NULL DEFAULT false,
  build_log text,
  error_message text,
  created_by uuid NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
  completed_at timestamptz,
  deleted_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (project_id, version),
  UNIQUE (id, project_id),
  UNIQUE (id, project_id, tenant_id),
  FOREIGN KEY (project_id, tenant_id) REFERENCES projects (id, tenant_id) ON DELETE CASCADE,
  CHECK (NOT restorable OR (
    status = 'ready' AND authoring_snapshot_schema IS NOT NULL AND authoring_snapshot_bucket IS NOT NULL
    AND authoring_snapshot_key IS NOT NULL AND authoring_snapshot_hash IS NOT NULL
    AND authoring_snapshot_cipher_hash IS NOT NULL AND authoring_snapshot_size IS NOT NULL
    AND authoring_snapshot_key_id IS NOT NULL AND authoring_project_revision IS NOT NULL
  ))
);
CREATE INDEX application_versions_project_idx ON application_versions (project_id, created_at DESC);

CREATE TABLE authoring_restore_tasks (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
  project_id uuid NOT NULL,
  application_version_id uuid NOT NULL,
  requested_by uuid NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
  current_project_revision text,
  expected_authoring_epoch bigint NOT NULL CHECK (expected_authoring_epoch > 0),
  target_authoring_epoch bigint NOT NULL CHECK (target_authoring_epoch = expected_authoring_epoch + 1),
  state text NOT NULL DEFAULT 'queued' CHECK (state IN (
    'queued', 'staging', 'restoring_workspace', 'restoring_scenes', 'restoring_data',
    'finalizing', 'compensating', 'succeeded', 'failed'
  )),
  stage text NOT NULL DEFAULT 'queued',
  backup_schema text CHECK (backup_schema IS NULL OR backup_schema = 'authoring-snapshot.v1'),
  backup_bucket text,
  backup_key text,
  backup_hash text CHECK (backup_hash IS NULL OR backup_hash ~ '^[0-9a-f]{64}$'),
  backup_cipher_hash text CHECK (backup_cipher_hash IS NULL OR backup_cipher_hash ~ '^[0-9a-f]{64}$'),
  backup_size bigint CHECK (backup_size IS NULL OR backup_size > 0),
  backup_key_id text,
  backup_project_revision text,
  rolled_back boolean NOT NULL DEFAULT false,
  workspace_was_running boolean,
  cleanup_completed_at timestamptz,
  error_message text,
  started_at timestamptz,
  completed_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (id, tenant_id),
  FOREIGN KEY (project_id, tenant_id) REFERENCES projects (id, tenant_id) ON DELETE CASCADE,
  FOREIGN KEY (application_version_id, project_id, tenant_id) REFERENCES application_versions (id, project_id, tenant_id) ON DELETE RESTRICT
);
CREATE UNIQUE INDEX authoring_restore_tasks_active_project_uidx
  ON authoring_restore_tasks (project_id)
  WHERE state IN ('queued', 'staging', 'restoring_workspace', 'restoring_scenes', 'restoring_data', 'finalizing', 'compensating');
CREATE INDEX authoring_restore_tasks_project_idx ON authoring_restore_tasks (project_id, created_at DESC, id DESC);

CREATE TABLE authoring_restore_task_events (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
  task_id uuid NOT NULL,
  stage text NOT NULL,
  message text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  FOREIGN KEY (task_id, tenant_id) REFERENCES authoring_restore_tasks (id, tenant_id) ON DELETE CASCADE
);
CREATE INDEX authoring_restore_task_events_task_idx ON authoring_restore_task_events (task_id, created_at, id);

CREATE TABLE authoring_project_fences (
  project_id uuid PRIMARY KEY,
  tenant_id uuid NOT NULL,
  owner_kind text NOT NULL CHECK (owner_kind IN ('build','restore')),
  owner_id uuid NOT NULL,
  fence_token_hash text NOT NULL CHECK (fence_token_hash ~ '^[0-9a-f]{64}$'),
  expires_at timestamptz NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  FOREIGN KEY (project_id, tenant_id) REFERENCES projects (id, tenant_id) ON DELETE CASCADE
);
CREATE INDEX authoring_project_fences_expiry_idx ON authoring_project_fences (expires_at);

CREATE FUNCTION enforce_authoring_project_write() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE pid uuid;
BEGIN
  IF TG_TABLE_NAME='projects' THEN pid := COALESCE(NEW.id, OLD.id); ELSE pid := COALESCE(NEW.project_id, OLD.project_id); END IF;
  IF current_setting('induforge.authoring_restore', true) = 'on' THEN IF TG_OP='DELETE' THEN RETURN OLD; ELSE RETURN NEW; END IF; END IF;
  PERFORM pg_advisory_xact_lock_shared(hashtextextended(pid::text, 0));
  IF EXISTS (SELECT 1 FROM authoring_project_fences WHERE project_id=pid AND expires_at>now()) OR EXISTS (SELECT 1 FROM authoring_restore_tasks WHERE project_id=pid AND state IN ('queued','staging','restoring_workspace','restoring_scenes','restoring_data','finalizing','compensating')) THEN
    RAISE EXCEPTION 'authoring project is fenced' USING ERRCODE='55000';
  END IF;
  IF TG_OP='DELETE' THEN RETURN OLD; ELSE RETURN NEW; END IF;
END $$;
DO $$ DECLARE table_name text; BEGIN
  FOREACH table_name IN ARRAY ARRAY['projects','scene_documents','scene_assets','scene_asset_generations','scene_asset_generation_files','scene_asset_draft_files','scene_file_nodes','scene_asset_bindings','scene_revisions','scene_revision_files','scene_revision_assets','project_tag_bindings','project_group_members'] LOOP
    EXECUTE format('CREATE TRIGGER %I_authoring_gate BEFORE INSERT OR UPDATE OR DELETE ON %I FOR EACH ROW EXECUTE FUNCTION enforce_authoring_project_write()',table_name,table_name);
  END LOOP;
END $$;

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

CREATE TABLE node_enrollments (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
  platform text NOT NULL CHECK (platform IN ('linux', 'windows')),
  capabilities jsonb NOT NULL CHECK (jsonb_typeof(capabilities) = 'array'),
  display_name text,
  code_hash text NOT NULL UNIQUE,
  status text NOT NULL DEFAULT 'created' CHECK (status IN ('created', 'claimed', 'approved', 'rejected', 'expired')),
  expires_at timestamptz NOT NULL,
  claimed_at timestamptz,
  claimed_by_node_id uuid,
  approved_at timestamptz,
  approved_by uuid REFERENCES users (id) ON DELETE SET NULL,
  rejected_at timestamptz,
  rejected_by uuid REFERENCES users (id) ON DELETE SET NULL,
  deleted_at timestamptz,
  deleted_by uuid REFERENCES users (id) ON DELETE SET NULL,
  created_by uuid NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX node_enrollments_tenant_idx ON node_enrollments (tenant_id, status, expires_at DESC) WHERE deleted_at IS NULL;

CREATE TABLE host_nodes (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
  node_source text NOT NULL DEFAULT 'agent' CHECK (node_source IN ('built_in', 'agent')),
  enrollment_id uuid UNIQUE REFERENCES node_enrollments (id) ON DELETE RESTRICT,
  display_name text NOT NULL,
  hostname text NOT NULL,
  platform text NOT NULL CHECK (platform IN ('linux', 'windows')),
  architecture text NOT NULL,
  agent_version text,
  agent_token_hash text UNIQUE,
  machine_fingerprint text,
  ip_address text,
  desired_status text NOT NULL DEFAULT 'active' CHECK (desired_status IN ('active', 'maintenance', 'revoked')),
  observed_status text NOT NULL DEFAULT 'offline' CHECK (observed_status IN ('offline', 'online', 'degraded', 'revoked')),
  resource_summary jsonb NOT NULL DEFAULT '{}'::jsonb,
  capabilities jsonb NOT NULL DEFAULT '[]'::jsonb CHECK (jsonb_typeof(capabilities) = 'array'),
  last_heartbeat_at timestamptz,
  approved_at timestamptz,
  approved_by uuid REFERENCES users (id) ON DELETE SET NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT host_nodes_source_identity_check CHECK (
    (node_source = 'built_in' AND enrollment_id IS NULL AND agent_token_hash IS NULL)
    OR
    (node_source = 'agent' AND enrollment_id IS NOT NULL AND agent_token_hash IS NOT NULL)
  )
);
CREATE INDEX host_nodes_tenant_idx ON host_nodes (tenant_id, observed_status, updated_at DESC);
CREATE UNIQUE INDEX host_nodes_tenant_built_in_key ON host_nodes (tenant_id) WHERE node_source = 'built_in';

-- 每个中心只维护一套 K3s 集群。集群节点身份独立于运行环境：中心节点固定
-- 承载 Server/embedded etcd，外部 Linux 节点固定作为工作节点。
CREATE TABLE runtime_clusters (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL UNIQUE REFERENCES tenants (id) ON DELETE CASCADE,
  name text NOT NULL DEFAULT '中心运行集群',
  k3s_version text NOT NULL,
  api_port integer NOT NULL CHECK (api_port BETWEEN 1 AND 65535),
  desired_status text NOT NULL DEFAULT 'active' CHECK (desired_status IN ('active', 'maintenance')),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE runtime_cluster_nodes (
  tenant_id uuid NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
  cluster_id uuid NOT NULL REFERENCES runtime_clusters (id) ON DELETE CASCADE,
  node_id uuid PRIMARY KEY REFERENCES host_nodes (id) ON DELETE RESTRICT,
  node_kind text NOT NULL CHECK (node_kind IN ('center', 'worker')),
  desired_action text NOT NULL DEFAULT 'active' CHECK (desired_action IN ('active', 'removing')),
  desired_generation bigint NOT NULL DEFAULT 1 CHECK (desired_generation > 0),
  observed_generation bigint NOT NULL DEFAULT 0 CHECK (observed_generation >= 0),
  cluster_status text NOT NULL DEFAULT 'pending' CHECK (cluster_status IN ('pending', 'starting', 'ready', 'failed', 'removing', 'not-installed')),
  cluster_message text,
  cluster_observed_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX runtime_cluster_nodes_center_key ON runtime_cluster_nodes (cluster_id) WHERE node_kind='center';
CREATE INDEX runtime_cluster_nodes_tenant_idx ON runtime_cluster_nodes (tenant_id, cluster_id, created_at DESC);

CREATE TABLE runtime_cluster_events (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
  cluster_id uuid NOT NULL REFERENCES runtime_clusters (id) ON DELETE CASCADE,
  node_id uuid REFERENCES host_nodes (id) ON DELETE SET NULL,
  event_type text NOT NULL,
  name text NOT NULL,
  target text NOT NULL DEFAULT '',
  result text NOT NULL CHECK (result IN ('success', 'failed')),
  message text,
  created_by uuid REFERENCES users (id) ON DELETE SET NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX runtime_cluster_events_tenant_created_idx ON runtime_cluster_events (tenant_id, created_at DESC, id DESC);

CREATE TABLE runtime_environments (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
  name text NOT NULL CHECK (length(btrim(name)) BETWEEN 1 AND 80),
  code text NOT NULL,
  is_default boolean NOT NULL DEFAULT false,
  desired_status text NOT NULL DEFAULT 'active' CHECK (desired_status IN ('active', 'maintenance', 'deleting')),
  created_by uuid NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz,
  CONSTRAINT runtime_environments_tenant_code_key UNIQUE (tenant_id, code)
);
CREATE UNIQUE INDEX runtime_environments_tenant_name_key ON runtime_environments (tenant_id, lower(name)) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX runtime_environments_tenant_default_key ON runtime_environments (tenant_id) WHERE is_default AND deleted_at IS NULL;
CREATE INDEX runtime_environments_tenant_updated_idx ON runtime_environments (tenant_id, updated_at DESC);

CREATE TABLE runtime_environment_nodes (
  tenant_id uuid NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
  environment_id uuid NOT NULL REFERENCES runtime_environments (id) ON DELETE CASCADE,
  node_id uuid NOT NULL REFERENCES host_nodes (id) ON DELETE RESTRICT,
  created_by uuid NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
  created_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (environment_id, node_id)
);
CREATE INDEX runtime_environment_nodes_tenant_idx ON runtime_environment_nodes (tenant_id, environment_id, created_at DESC);

CREATE TABLE runtime_environment_services (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
  environment_id uuid NOT NULL REFERENCES runtime_environments (id) ON DELETE CASCADE,
  node_id uuid NOT NULL REFERENCES host_nodes (id) ON DELETE RESTRICT,
  previous_node_id uuid REFERENCES host_nodes (id) ON DELETE RESTRICT,
  service_type text NOT NULL CHECK (service_type IN ('if_realtime', 'if_history', 'if_timeseries', 'if_message', 'if_object', 'nats_jetstream', 'nginx', 'traefik')),
  deployment_mode text NOT NULL DEFAULT 'single' CHECK (deployment_mode IN ('single', 'primary', 'replica')),
  desired_status text NOT NULL DEFAULT 'running' CHECK (desired_status IN ('running', 'stopped')),
  observed_status text NOT NULL DEFAULT 'pending' CHECK (observed_status IN ('pending', 'running', 'stopped', 'degraded', 'failed')),
  desired_generation bigint NOT NULL DEFAULT 1 CHECK (desired_generation > 0),
  observed_generation bigint NOT NULL DEFAULT 0 CHECK (observed_generation >= 0),
  operation text NOT NULL DEFAULT 'apply' CHECK (operation IN ('apply', 'migrate')),
  storage_claim text NOT NULL DEFAULT '',
  previous_storage_claim text,
  -- 仅登记受控 Kubernetes 资源/Secret 引用，绝不保存连接串、令牌或密码。
  resource_refs jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(resource_refs) = 'object'),
  last_message text,
  observed_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT runtime_environment_services_type_key UNIQUE (environment_id, service_type)
);
CREATE INDEX runtime_environment_services_node_idx ON runtime_environment_services (node_id, observed_status, updated_at DESC);

CREATE TABLE runtime_environment_events (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
  environment_id uuid NOT NULL REFERENCES runtime_environments (id) ON DELETE CASCADE,
  event_type text NOT NULL,
  name text NOT NULL,
  target text NOT NULL DEFAULT '',
  result text NOT NULL CHECK (result IN ('success', 'failed')),
  message text,
  created_by uuid REFERENCES users (id) ON DELETE SET NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX runtime_environment_events_environment_idx ON runtime_environment_events (environment_id, created_at DESC, id DESC);

CREATE TABLE project_deployments (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
  project_id uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
  -- 一个工程在一个运行环境只有一个部署槽位。DEV 与 RELEASE 都是替换该槽位，
  -- 不能以模式为维度并行运行两个版本。
  environment_id uuid NOT NULL REFERENCES runtime_environments (id) ON DELETE RESTRICT,
  -- 生产记录引用不可变 Release；开发记录引用同工程可替换的内部 __DEV__ 制品。
  application_version_id uuid REFERENCES application_versions (id) ON DELETE RESTRICT,
  mode text NOT NULL CHECK (mode IN ('development', 'release')),
  artifact_descriptor jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(artifact_descriptor) = 'object'),
  last_ready_mode text CHECK (last_ready_mode IN ('development', 'release')),
  last_ready_application_version_id uuid REFERENCES application_versions (id) ON DELETE RESTRICT,
  last_ready_artifact_descriptor jsonb CHECK (last_ready_artifact_descriptor IS NULL OR jsonb_typeof(last_ready_artifact_descriptor) = 'object'),
  last_ready_generation bigint,
  last_ready_at timestamptz,
  access_port integer NOT NULL CHECK (access_port BETWEEN 1024 AND 65532),
  deleted_at timestamptz,
  deletion_requested_at timestamptz,
  desired_status text NOT NULL DEFAULT 'running' CHECK (desired_status IN ('running', 'stopped')),
  observed_status text NOT NULL DEFAULT 'pending' CHECK (observed_status IN ('pending', 'running', 'stopped', 'degraded', 'failed')),
  created_by uuid NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT project_deployments_artifact_source_check CHECK (
    (mode = 'release' AND application_version_id IS NOT NULL AND artifact_descriptor = '{}'::jsonb) OR
    (mode = 'development' AND application_version_id IS NULL AND artifact_descriptor ? 'releaseId')
  ),
  CONSTRAINT project_deployments_last_ready_check CHECK (
    (last_ready_mode IS NULL AND last_ready_application_version_id IS NULL AND last_ready_artifact_descriptor IS NULL AND last_ready_generation IS NULL AND last_ready_at IS NULL) OR
    (last_ready_mode = 'release' AND last_ready_application_version_id IS NOT NULL AND last_ready_artifact_descriptor IS NULL AND last_ready_generation IS NOT NULL AND last_ready_at IS NOT NULL) OR
    (last_ready_mode = 'development' AND last_ready_application_version_id IS NULL AND last_ready_artifact_descriptor IS NOT NULL AND last_ready_artifact_descriptor ? 'releaseId' AND last_ready_generation IS NOT NULL AND last_ready_at IS NOT NULL)
  )
);
CREATE INDEX project_deployments_tenant_idx ON project_deployments (tenant_id, project_id, updated_at DESC);
CREATE INDEX project_deployments_environment_active_idx ON project_deployments (tenant_id, environment_id) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX project_deployments_tenant_project_environment_active_key ON project_deployments (tenant_id, project_id, environment_id) WHERE deleted_at IS NULL;

CREATE TABLE deployment_runs (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
  project_deployment_id uuid NOT NULL REFERENCES project_deployments (id) ON DELETE CASCADE,
  operation text NOT NULL CHECK (operation IN ('deploy', 'start', 'stop', 'restart', 'delete')),
  desired_status text NOT NULL CHECK (desired_status IN ('running', 'stopped')),
  observed_status text NOT NULL DEFAULT 'pending' CHECK (observed_status IN ('pending', 'running', 'stopped', 'failed')),
  progress integer NOT NULL DEFAULT 0 CHECK (progress BETWEEN 0 AND 100),
  message text,
  started_at timestamptz NOT NULL DEFAULT now(),
  completed_at timestamptz,
  created_by uuid NOT NULL REFERENCES users (id) ON DELETE RESTRICT
);
CREATE INDEX deployment_runs_deployment_idx ON deployment_runs (project_deployment_id, started_at DESC);
-- 同一部署只能有一条待调和记录；控制面会锁定部署行，索引同时防止绕过服务层的并发写入。
CREATE UNIQUE INDEX deployment_runs_one_pending_idx ON deployment_runs (project_deployment_id)
  WHERE observed_status = 'pending';

CREATE TABLE deployment_run_events (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  deployment_run_id uuid NOT NULL REFERENCES deployment_runs (id) ON DELETE CASCADE,
  stage text NOT NULL CHECK (stage IN ('queued', 'dispatched', 'observed', 'failed')),
  message text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX deployment_run_events_run_idx ON deployment_run_events (deployment_run_id, created_at);

CREATE TABLE deployment_services (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
  project_deployment_id uuid NOT NULL REFERENCES project_deployments (id) ON DELETE CASCADE,
  node_id uuid NOT NULL REFERENCES host_nodes (id) ON DELETE RESTRICT,
  -- 基础引擎包含工程网关与 Runtime API，始终存在；其余引擎由工程内容自动推导。
  service_type text NOT NULL CHECK (service_type IN ('base', 'compute', 'alarm', 'collector')),
  public_port integer CHECK (public_port IS NULL OR public_port BETWEEN 1024 AND 65535),
  desired_status text NOT NULL DEFAULT 'running' CHECK (desired_status IN ('running', 'stopped')),
  observed_status text NOT NULL DEFAULT 'pending' CHECK (observed_status IN ('pending', 'running', 'stopped', 'failed')),
  replicas_desired integer NOT NULL DEFAULT 1 CHECK (replicas_desired >= 0),
  replicas_observed integer NOT NULL DEFAULT 0 CHECK (replicas_observed >= 0),
  desired_generation bigint NOT NULL DEFAULT 1 CHECK (desired_generation > 0),
  observed_generation bigint NOT NULL DEFAULT 0 CHECK (observed_generation >= 0),
  last_operation text NOT NULL DEFAULT 'deploy' CHECK (last_operation IN ('deploy', 'start', 'stop', 'restart', 'delete')),
  last_message text,
  endpoint text NOT NULL DEFAULT '' CHECK (
    endpoint = '' OR (
      endpoint ~ '^https?://[^/@#[:space:]]+' AND
      endpoint !~ '^https?://[^/]*@' AND
      position('#' in endpoint) = 0
    )
  ),
  observed_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (project_deployment_id, service_type)
);
CREATE INDEX deployment_services_node_idx ON deployment_services (node_id, desired_status);
-- 工程入口采用 hostPort + nodeSelector，入口只绑定基础引擎所在的物理节点。
-- 因此端口冲突是节点级，不再沿用旧实现的连续四端口假设。
CREATE UNIQUE INDEX deployment_services_base_access_port_key
  ON deployment_services (node_id, public_port)
  WHERE service_type = 'base' AND public_port IS NOT NULL;
CREATE TABLE deployment_bindings (
  id uuid PRIMARY KEY, tenant_id uuid NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
  project_deployment_id uuid NOT NULL REFERENCES project_deployments (id) ON DELETE CASCADE,
  deployment_service_id uuid NOT NULL REFERENCES deployment_services (id) ON DELETE CASCADE,
  project_id uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE, node_id uuid NOT NULL REFERENCES host_nodes (id) ON DELETE RESTRICT,
  application_version_id uuid REFERENCES application_versions (id) ON DELETE RESTRICT,
  artifact_mode text NOT NULL CHECK (artifact_mode IN ('release', 'development')),
  artifact_descriptor jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(artifact_descriptor) = 'object'), revision integer NOT NULL CHECK (revision > 0), binding jsonb NOT NULL CHECK (jsonb_typeof(binding) = 'object'),
  created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(), UNIQUE (deployment_service_id, revision),
  CONSTRAINT deployment_bindings_artifact_source_check CHECK ((artifact_mode = 'release' AND application_version_id IS NOT NULL AND artifact_descriptor = '{}'::jsonb) OR (artifact_mode = 'development' AND application_version_id IS NULL AND artifact_descriptor ? 'releaseId'))
);
CREATE INDEX deployment_bindings_agent_lookup_idx ON deployment_bindings (node_id, deployment_service_id, revision DESC);
