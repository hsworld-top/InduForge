-- ============================================
-- InduForge dev_core PostgreSQL 初始化脚本
-- 说明：
-- 1. 仅保留 dev_core 当前仍在使用的控制面表。
-- 2. 历史数据域表已下线，相关能力由 data_service 承接。
-- 3. 物理列暂保留 camelCase 命名，降低 Sequelize 迁移改造范围。
-- ============================================

CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ============================================
-- 1. 基础平台表
-- ============================================

CREATE TABLE IF NOT EXISTS tenants (
  "id" uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  "name" text NOT NULL,
  "code" text NOT NULL,
  "description" text,
  "status" text NOT NULL DEFAULT 'active' CHECK ("status" IN ('active', 'inactive', 'suspended')),
  "contactEmail" text,
  "contactPhone" text,
  "maxUsers" integer NOT NULL DEFAULT 100 CHECK ("maxUsers" >= 0),
  "maxProjects" integer NOT NULL DEFAULT 50 CHECK ("maxProjects" >= 0),
  "maxStorage" bigint NOT NULL DEFAULT 10737418240 CHECK ("maxStorage" >= 0),
  "usedStorage" bigint NOT NULL DEFAULT 0 CHECK ("usedStorage" >= 0),
  "logoUrl" text,
  "loginBackgroundUrl" text,
  "companyName" text,
  "companyAddress" text,
  "companyPhone" text,
  "companyWebsite" text,
  "settings" jsonb,
  "expiresAt" timestamptz,
  "createdAt" timestamptz NOT NULL DEFAULT now(),
  "updatedAt" timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS tenants_code_uq ON tenants ("code");
CREATE INDEX IF NOT EXISTS tenants_status_idx ON tenants ("status");
CREATE INDEX IF NOT EXISTS tenants_expires_idx ON tenants ("expiresAt");

CREATE TABLE IF NOT EXISTS users (
  "id" uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  "tenantId" uuid NOT NULL REFERENCES tenants ("id") ON DELETE CASCADE,
  "username" text NOT NULL,
  "password" text NOT NULL,
  "email" text,
  "phone" text,
  "fullName" text,
  "avatar" text,
  "role" text NOT NULL CHECK (
    "role" IN (
      'SUPER_ADMIN',
      'SYSTEM_ADMIN',
      'PROJECT_ADMIN',
      'OPS_ADMIN',
      'USER_ADMIN',
      'DEVELOPER',
      'OPERATOR',
      'VIEWER'
    )
  ),
  "status" text NOT NULL DEFAULT 'active' CHECK ("status" IN ('active', 'inactive', 'suspended')),
  "preferences" jsonb,
  "lastLoginAt" timestamptz,
  "lastLoginIp" text,
  "passwordChangedAt" timestamptz,
  "createdAt" timestamptz NOT NULL DEFAULT now(),
  "updatedAt" timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS users_tenant_username_uq ON users ("tenantId", "username");
CREATE INDEX IF NOT EXISTS users_tenant_role_idx ON users ("tenantId", "role");
CREATE INDEX IF NOT EXISTS users_status_idx ON users ("status");
CREATE INDEX IF NOT EXISTS users_email_idx ON users ("email");

CREATE TABLE IF NOT EXISTS projects (
  "id" uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  "tenantId" uuid NOT NULL REFERENCES tenants ("id") ON DELETE CASCADE,
  "name" text NOT NULL,
  "code" text,
  "description" text,
  "projectVariables" jsonb,
  "entryConfig" jsonb DEFAULT '{}'::jsonb,
  "colorTag" text NOT NULL DEFAULT '#3b82f6' CHECK (
    "colorTag" IN ('#3b82f6', '#ef4444', '#10b981', '#f59e0b', '#8b5cf6', '#ec4899', '#6b7280')
  ),
  "icon" text,
  "status" text NOT NULL DEFAULT 'active' CHECK ("status" IN ('active', 'archived', 'deleted')),
  "visibility" text NOT NULL DEFAULT 'private' CHECK ("visibility" IN ('private', 'internal', 'public')),
  "createdBy" uuid NOT NULL REFERENCES users ("id"),
  "updatedBy" uuid REFERENCES users ("id") ON DELETE SET NULL,
  "archivedAt" timestamptz,
  "createdAt" timestamptz NOT NULL DEFAULT now(),
  "updatedAt" timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS projects_tenant_idx ON projects ("tenantId");
CREATE INDEX IF NOT EXISTS projects_status_idx ON projects ("status");
CREATE UNIQUE INDEX IF NOT EXISTS projects_tenant_code_uq
  ON projects ("tenantId", "code")
  WHERE "code" IS NOT NULL;

CREATE TABLE IF NOT EXISTS logs (
  "id" uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  "tenantId" uuid REFERENCES tenants ("id") ON DELETE SET NULL,
  "projectId" uuid REFERENCES projects ("id") ON DELETE SET NULL,
  "userId" uuid REFERENCES users ("id") ON DELETE SET NULL,
  "level" text NOT NULL DEFAULT 'info' CHECK ("level" IN ('debug', 'info', 'warning', 'error')),
  "category" text,
  "action" text NOT NULL,
  "resource" text,
  "resourceId" uuid,
  "message" text NOT NULL,
  "details" jsonb,
  "ip" text,
  "userAgent" text,
  "duration" integer,
  "createdAt" timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS logs_tenant_idx ON logs ("tenantId");
CREATE INDEX IF NOT EXISTS logs_project_idx ON logs ("projectId");
CREATE INDEX IF NOT EXISTS logs_user_idx ON logs ("userId");
CREATE INDEX IF NOT EXISTS logs_category_action_idx ON logs ("category", "action");
CREATE INDEX IF NOT EXISTS logs_created_at_idx ON logs ("createdAt");

-- ============================================
-- 2. 设计器相关表
-- ============================================

CREATE TABLE IF NOT EXISTS design_pages (
  "id" uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  "projectId" uuid NOT NULL REFERENCES projects ("id") ON DELETE CASCADE,
  "parentId" uuid REFERENCES design_pages ("id") ON DELETE CASCADE,
  "name" text NOT NULL,
  "path" text,
  "type" text NOT NULL DEFAULT 'page' CHECK ("type" IN ('page', 'folder', 'dialog', 'template')),
  "icon" text,
  "schemaVersion" text DEFAULT '1.0.0',
  "schemaContent" jsonb,
  "pageConfig" jsonb,
  "variables" jsonb,
  "dataSources" jsonb,
  "lifecycle" jsonb,
  "isHome" boolean DEFAULT false,
  "isPublished" boolean DEFAULT false,
  "status" text DEFAULT 'draft' CHECK ("status" IN ('draft', 'review', 'published', 'archived')),
  "thumbnailUrl" text,
  "sortOrder" integer NOT NULL DEFAULT 0,
  "lockedBy" uuid REFERENCES users ("id") ON DELETE SET NULL,
  "lockedAt" timestamptz,
  "publishedAt" timestamptz,
  "publishedBy" uuid REFERENCES users ("id") ON DELETE SET NULL,
  "createdBy" uuid NOT NULL REFERENCES users ("id"),
  "updatedBy" uuid REFERENCES users ("id") ON DELETE SET NULL,
  "createdAt" timestamptz NOT NULL DEFAULT now(),
  "updatedAt" timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS design_pages_project_idx ON design_pages ("projectId");
CREATE INDEX IF NOT EXISTS design_pages_parent_idx ON design_pages ("parentId");
CREATE INDEX IF NOT EXISTS design_pages_type_idx ON design_pages ("type");
CREATE INDEX IF NOT EXISTS design_pages_status_idx ON design_pages ("status");
CREATE INDEX IF NOT EXISTS design_pages_locked_idx ON design_pages ("lockedBy");
CREATE INDEX IF NOT EXISTS design_pages_sort_idx ON design_pages ("projectId", "parentId", "sortOrder");
CREATE UNIQUE INDEX IF NOT EXISTS design_pages_project_path_uq
  ON design_pages ("projectId", "path")
  WHERE "path" IS NOT NULL;

CREATE TABLE IF NOT EXISTS design_asset_folders (
  "id" uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  "projectId" uuid NOT NULL REFERENCES projects ("id") ON DELETE CASCADE,
  "parentId" uuid REFERENCES design_asset_folders ("id") ON DELETE CASCADE,
  "name" text NOT NULL,
  "sortOrder" integer DEFAULT 0,
  "createdAt" timestamptz NOT NULL DEFAULT now(),
  "updatedAt" timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS asset_folders_root_name_uq
  ON design_asset_folders ("projectId", "name")
  WHERE "parentId" IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS asset_folders_child_name_uq
  ON design_asset_folders ("projectId", "parentId", "name")
  WHERE "parentId" IS NOT NULL;

CREATE TABLE IF NOT EXISTS design_assets (
  "id" uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  "projectId" uuid NOT NULL REFERENCES projects ("id") ON DELETE CASCADE,
  "folderId" uuid REFERENCES design_asset_folders ("id") ON DELETE SET NULL,
  "name" text NOT NULL,
  "originalName" text,
  "type" text NOT NULL CHECK ("type" IN ('image', 'svg', 'video', 'audio', 'model_3d', 'font', 'json', 'other')),
  "mimeType" text,
  "url" text NOT NULL,
  "thumbnailUrl" text,
  "size" bigint DEFAULT 0,
  "width" integer,
  "height" integer,
  "duration" integer,
  "metadata" jsonb,
  "tags" jsonb,
  "usageCount" integer DEFAULT 0,
  "uploadedBy" uuid NOT NULL REFERENCES users ("id"),
  "createdAt" timestamptz NOT NULL DEFAULT now(),
  "updatedAt" timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS assets_project_type_idx ON design_assets ("projectId", "type");
CREATE INDEX IF NOT EXISTS assets_folder_idx ON design_assets ("folderId");

CREATE TABLE IF NOT EXISTS design_project_settings (
  "projectId" uuid PRIMARY KEY REFERENCES projects ("id") ON DELETE CASCADE,
  "schemaVersion" text DEFAULT '1.0.0',
  "globalVariables" jsonb,
  "globalStyles" jsonb,
  "globalScripts" jsonb,
  "globalDataSources" jsonb,
  "componentMappings" jsonb,
  "i18n" jsonb,
  "permissions" jsonb,
  "buildConfig" jsonb,
  "runtimeConfig" jsonb,
  "updatedBy" uuid REFERENCES users ("id") ON DELETE SET NULL,
  "updatedAt" timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS design_project_settings_updated_by_idx
  ON design_project_settings ("updatedBy");

-- ============================================
-- 3. 发布部署相关表
-- ============================================

CREATE TABLE IF NOT EXISTS deployments (
  "id" uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  "projectId" uuid NOT NULL REFERENCES projects ("id") ON DELETE CASCADE,
  "tenantId" uuid NOT NULL REFERENCES tenants ("id") ON DELETE CASCADE,
  "version" text NOT NULL,
  "name" text,
  "description" text,
  "type" text NOT NULL DEFAULT 'development' CHECK ("type" IN ('development', 'staging', 'production')),
  "mode" text NOT NULL DEFAULT 'RELEASE' CHECK ("mode" IN ('DEV', 'RELEASE')),
  "status" text NOT NULL DEFAULT 'pending' CHECK ("status" IN ('pending', 'building', 'success', 'failed')),
  "buildConfig" jsonb,
  "buildLog" jsonb,
  "artifactUrl" text,
  "artifactHash" text,
  "artifactSize" bigint,
  "snapshotUrl" text,
  "snapshotHash" text,
  "manifest" jsonb,
  "pageCount" integer DEFAULT 0,
  "componentCount" integer DEFAULT 0,
  "datapointCount" integer DEFAULT 0,
  "startedAt" timestamptz,
  "completedAt" timestamptz,
  "errorMessage" text,
  "deployedBy" uuid REFERENCES users ("id") ON DELETE SET NULL,
  "createdAt" timestamptz NOT NULL DEFAULT now(),
  "updatedAt" timestamptz NOT NULL DEFAULT now(),
  "deletedAt" timestamptz
);

CREATE INDEX IF NOT EXISTS deployments_project_idx ON deployments ("projectId");
CREATE INDEX IF NOT EXISTS deployments_tenant_idx ON deployments ("tenantId");
CREATE INDEX IF NOT EXISTS deployments_status_idx ON deployments ("status");
CREATE INDEX IF NOT EXISTS deployments_type_idx ON deployments ("type");
CREATE INDEX IF NOT EXISTS deployments_mode_idx ON deployments ("mode");
CREATE INDEX IF NOT EXISTS deployments_created_at_idx ON deployments ("createdAt");
CREATE UNIQUE INDEX IF NOT EXISTS deployments_version_uq
  ON deployments ("projectId", "version")
  WHERE "deletedAt" IS NULL;

CREATE TABLE IF NOT EXISTS nodes (
  "id" uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  "tenantId" uuid NOT NULL REFERENCES tenants ("id") ON DELETE CASCADE,
  "name" text NOT NULL,
  "description" text,
  "agentVersion" text,
  "status" text NOT NULL DEFAULT 'offline' CHECK ("status" IN ('online', 'offline', 'error')),
  "currentProjectId" uuid REFERENCES projects ("id") ON DELETE SET NULL,
  "currentVersion" text,
  "currentDeploymentId" uuid,
  "ipAddress" text,
  "port" integer DEFAULT 8080,
  "lastHeartbeatAt" timestamptz,
  "lastErrorMessage" text,
  "lastErrorAt" timestamptz,
  "metrics" jsonb,
  "config" jsonb,
  "registrationToken" text,
  "approvalStatus" text NOT NULL DEFAULT 'pending' CHECK ("approvalStatus" IN ('pending', 'approved', 'rejected')),
  "approvedAt" timestamptz,
  "approvedBy" uuid REFERENCES users ("id") ON DELETE SET NULL,
  "registeredBy" uuid REFERENCES users ("id") ON DELETE SET NULL,
  "mode" text NOT NULL DEFAULT 'online' CHECK ("mode" IN ('online', 'offline')),
  "createdBy" uuid REFERENCES users ("id") ON DELETE SET NULL,
  "updatedBy" uuid REFERENCES users ("id") ON DELETE SET NULL,
  "createdAt" timestamptz NOT NULL DEFAULT now(),
  "updatedAt" timestamptz NOT NULL DEFAULT now(),
  "deletedAt" timestamptz
);

CREATE INDEX IF NOT EXISTS nodes_tenant_idx ON nodes ("tenantId");
CREATE INDEX IF NOT EXISTS nodes_status_idx ON nodes ("status");
CREATE INDEX IF NOT EXISTS nodes_heartbeat_idx ON nodes ("lastHeartbeatAt");
CREATE UNIQUE INDEX IF NOT EXISTS nodes_tenant_name_uq
  ON nodes ("tenantId", "name")
  WHERE "deletedAt" IS NULL;

CREATE TABLE IF NOT EXISTS node_deployments (
  "id" uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  "nodeId" uuid NOT NULL REFERENCES nodes ("id") ON DELETE CASCADE,
  "deploymentId" uuid NOT NULL REFERENCES deployments ("id") ON DELETE CASCADE,
  "projectId" uuid NOT NULL REFERENCES projects ("id") ON DELETE CASCADE,
  "version" text NOT NULL,
  "mode" text NOT NULL DEFAULT 'RELEASE' CHECK ("mode" IN ('DEV', 'RELEASE')),
  "status" text NOT NULL DEFAULT 'pending' CHECK ("status" IN ('pending', 'deploying', 'running', 'stopped', 'error', 'rollback')),
  "runtimeConfig" jsonb,
  "deployedAt" timestamptz,
  "startedAt" timestamptz,
  "stoppedAt" timestamptz,
  "deployedBy" uuid REFERENCES users ("id") ON DELETE SET NULL,
  "errorMessage" text,
  "errorStack" text,
  "deployLog" jsonb,
  "runtimeMetrics" jsonb,
  "createdAt" timestamptz NOT NULL DEFAULT now(),
  "updatedAt" timestamptz NOT NULL DEFAULT now(),
  "deletedAt" timestamptz
);

CREATE INDEX IF NOT EXISTS node_deploy_node_idx ON node_deployments ("nodeId");
CREATE INDEX IF NOT EXISTS node_deploy_deployment_idx ON node_deployments ("deploymentId");
CREATE INDEX IF NOT EXISTS node_deploy_project_idx ON node_deployments ("projectId");
CREATE INDEX IF NOT EXISTS node_deploy_status_idx ON node_deployments ("status");
CREATE INDEX IF NOT EXISTS node_deploy_mode_idx ON node_deployments ("mode");
CREATE UNIQUE INDEX IF NOT EXISTS node_deploy_active_uq
  ON node_deployments ("nodeId", "projectId")
  WHERE "deletedAt" IS NULL AND "status" IN ('pending', 'deploying', 'running');

CREATE TABLE IF NOT EXISTS node_commands (
  "id" uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  "tenantId" uuid NOT NULL REFERENCES tenants ("id") ON DELETE CASCADE,
  "nodeId" uuid NOT NULL REFERENCES nodes ("id") ON DELETE CASCADE,
  "deploymentId" uuid NOT NULL REFERENCES node_deployments ("id") ON DELETE CASCADE,
  "projectId" uuid NOT NULL REFERENCES projects ("id") ON DELETE CASCADE,
  "type" text NOT NULL CHECK ("type" IN ('deploy', 'start', 'stop', 'restart')),
  "status" text NOT NULL DEFAULT 'pending' CHECK ("status" IN ('pending', 'issued', 'acknowledged', 'completed', 'failed', 'dead_letter')),
  "payload" jsonb,
  "attempts" integer NOT NULL DEFAULT 0,
  "maxAttempts" integer NOT NULL DEFAULT 3,
  "timeoutSeconds" integer NOT NULL DEFAULT 30,
  "requestedAt" timestamptz NOT NULL DEFAULT now(),
  "issuedAt" timestamptz,
  "acknowledgedAt" timestamptz,
  "completedAt" timestamptz,
  "lastError" text,
  "createdAt" timestamptz NOT NULL DEFAULT now(),
  "updatedAt" timestamptz NOT NULL DEFAULT now(),
  "deletedAt" timestamptz
);

CREATE INDEX IF NOT EXISTS node_command_tenant_idx ON node_commands ("tenantId");
CREATE INDEX IF NOT EXISTS node_command_node_idx ON node_commands ("nodeId");
CREATE INDEX IF NOT EXISTS node_command_deploy_idx ON node_commands ("deploymentId");
CREATE INDEX IF NOT EXISTS node_command_project_idx ON node_commands ("projectId");
CREATE INDEX IF NOT EXISTS node_command_status_idx ON node_commands ("status");
CREATE INDEX IF NOT EXISTS node_command_type_idx ON node_commands ("type");
CREATE INDEX IF NOT EXISTS node_command_requested_idx ON node_commands ("requestedAt");
