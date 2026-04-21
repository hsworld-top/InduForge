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

COMMENT ON TABLE tenants IS '租户表';
COMMENT ON COLUMN tenants."id" IS '租户ID';
COMMENT ON COLUMN tenants."name" IS '租户名称';
COMMENT ON COLUMN tenants."code" IS '租户代码';
COMMENT ON COLUMN tenants."description" IS '租户描述';
COMMENT ON COLUMN tenants."status" IS '租户状态';
COMMENT ON COLUMN tenants."contactEmail" IS '联系邮箱';
COMMENT ON COLUMN tenants."contactPhone" IS '联系电话';
COMMENT ON COLUMN tenants."maxUsers" IS '最大用户数';
COMMENT ON COLUMN tenants."maxProjects" IS '最大工程数';
COMMENT ON COLUMN tenants."maxStorage" IS '最大存储空间(字节)';
COMMENT ON COLUMN tenants."usedStorage" IS '已用存储空间(字节)';
COMMENT ON COLUMN tenants."logoUrl" IS 'Logo资源路径';
COMMENT ON COLUMN tenants."loginBackgroundUrl" IS '登录页背景图路径';
COMMENT ON COLUMN tenants."companyName" IS '公司名称';
COMMENT ON COLUMN tenants."companyAddress" IS '公司地址';
COMMENT ON COLUMN tenants."companyPhone" IS '公司电话';
COMMENT ON COLUMN tenants."companyWebsite" IS '公司网站';
COMMENT ON COLUMN tenants."settings" IS '租户级配置';
COMMENT ON COLUMN tenants."expiresAt" IS '租户到期时间';
COMMENT ON COLUMN tenants."createdAt" IS '创建时间';
COMMENT ON COLUMN tenants."updatedAt" IS '更新时间';

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

COMMENT ON TABLE users IS '用户表';
COMMENT ON COLUMN users."id" IS '用户ID';
COMMENT ON COLUMN users."tenantId" IS '所属租户ID';
COMMENT ON COLUMN users."username" IS '用户名';
COMMENT ON COLUMN users."password" IS '密码哈希';
COMMENT ON COLUMN users."email" IS '邮箱';
COMMENT ON COLUMN users."phone" IS '手机号';
COMMENT ON COLUMN users."fullName" IS '真实姓名';
COMMENT ON COLUMN users."avatar" IS '头像资源路径';
COMMENT ON COLUMN users."role" IS '系统角色';
COMMENT ON COLUMN users."status" IS '用户状态';
COMMENT ON COLUMN users."preferences" IS '用户偏好设置';
COMMENT ON COLUMN users."lastLoginAt" IS '最后登录时间';
COMMENT ON COLUMN users."lastLoginIp" IS '最后登录IP';
COMMENT ON COLUMN users."passwordChangedAt" IS '密码最后修改时间';
COMMENT ON COLUMN users."createdAt" IS '创建时间';
COMMENT ON COLUMN users."updatedAt" IS '更新时间';

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

COMMENT ON TABLE projects IS '工程表';
COMMENT ON COLUMN projects."id" IS '工程ID';
COMMENT ON COLUMN projects."tenantId" IS '所属租户ID';
COMMENT ON COLUMN projects."name" IS '工程名称';
COMMENT ON COLUMN projects."code" IS '工程代码';
COMMENT ON COLUMN projects."description" IS '工程描述';
COMMENT ON COLUMN projects."projectVariables" IS '工程级全局变量定义';
COMMENT ON COLUMN projects."entryConfig" IS '工程入口配置';
COMMENT ON COLUMN projects."colorTag" IS '颜色标签';
COMMENT ON COLUMN projects."icon" IS '工程图标';
COMMENT ON COLUMN projects."status" IS '工程状态';
COMMENT ON COLUMN projects."visibility" IS '可见性';
COMMENT ON COLUMN projects."createdBy" IS '创建者ID';
COMMENT ON COLUMN projects."updatedBy" IS '更新者ID';
COMMENT ON COLUMN projects."archivedAt" IS '归档时间';
COMMENT ON COLUMN projects."createdAt" IS '创建时间';
COMMENT ON COLUMN projects."updatedAt" IS '更新时间';

-- ============================================
-- 1.1 运行态授权表
-- ============================================

CREATE TABLE IF NOT EXISTS project_runtime_users (
  "id" uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  "projectId" uuid NOT NULL REFERENCES projects ("id") ON DELETE CASCADE,
  "createdBy" uuid REFERENCES users ("id") ON DELETE SET NULL,
  "updatedBy" uuid REFERENCES users ("id") ON DELETE SET NULL,
  "username" text NOT NULL,
  "passwordHash" text NOT NULL,
  "displayName" text,
  "status" text NOT NULL DEFAULT 'active' CHECK ("status" IN ('active', 'inactive', 'suspended')),
  "lastLoginAt" timestamptz,
  "lastLoginIp" text,
  "createdAt" timestamptz NOT NULL DEFAULT now(),
  "updatedAt" timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS project_runtime_users_project_username_uq
  ON project_runtime_users ("projectId", "username");
CREATE INDEX IF NOT EXISTS project_runtime_users_project_status_idx
  ON project_runtime_users ("projectId", "status");
CREATE INDEX IF NOT EXISTS project_runtime_users_created_by_idx
  ON project_runtime_users ("createdBy");
CREATE INDEX IF NOT EXISTS project_runtime_users_updated_by_idx
  ON project_runtime_users ("updatedBy");

COMMENT ON TABLE project_runtime_users IS '工程运行态用户表';
COMMENT ON COLUMN project_runtime_users."id" IS '运行态用户ID';
COMMENT ON COLUMN project_runtime_users."projectId" IS '所属工程ID';
COMMENT ON COLUMN project_runtime_users."createdBy" IS '创建者ID';
COMMENT ON COLUMN project_runtime_users."updatedBy" IS '更新者ID';
COMMENT ON COLUMN project_runtime_users."username" IS '运行态用户名';
COMMENT ON COLUMN project_runtime_users."passwordHash" IS '密码哈希';
COMMENT ON COLUMN project_runtime_users."displayName" IS '显示名称';
COMMENT ON COLUMN project_runtime_users."status" IS '账号状态';
COMMENT ON COLUMN project_runtime_users."lastLoginAt" IS '最后登录时间';
COMMENT ON COLUMN project_runtime_users."lastLoginIp" IS '最后登录IP';
COMMENT ON COLUMN project_runtime_users."createdAt" IS '创建时间';
COMMENT ON COLUMN project_runtime_users."updatedAt" IS '更新时间';

CREATE TABLE IF NOT EXISTS project_roles (
  "id" uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  "projectId" uuid NOT NULL REFERENCES projects ("id") ON DELETE CASCADE,
  "createdBy" uuid REFERENCES users ("id") ON DELETE SET NULL,
  "updatedBy" uuid REFERENCES users ("id") ON DELETE SET NULL,
  "code" text NOT NULL,
  "name" text NOT NULL,
  "description" text,
  "isSystem" boolean NOT NULL DEFAULT false,
  "status" text NOT NULL DEFAULT 'active' CHECK ("status" IN ('active', 'inactive')),
  "createdAt" timestamptz NOT NULL DEFAULT now(),
  "updatedAt" timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS project_roles_project_code_uq
  ON project_roles ("projectId", "code");
CREATE INDEX IF NOT EXISTS project_roles_project_status_idx
  ON project_roles ("projectId", "status");
CREATE INDEX IF NOT EXISTS project_roles_created_by_idx
  ON project_roles ("createdBy");
CREATE INDEX IF NOT EXISTS project_roles_updated_by_idx
  ON project_roles ("updatedBy");

COMMENT ON TABLE project_roles IS '工程运行态角色表';
COMMENT ON COLUMN project_roles."id" IS '角色ID';
COMMENT ON COLUMN project_roles."projectId" IS '所属工程ID';
COMMENT ON COLUMN project_roles."createdBy" IS '创建者ID';
COMMENT ON COLUMN project_roles."updatedBy" IS '更新者ID';
COMMENT ON COLUMN project_roles."code" IS '角色编码';
COMMENT ON COLUMN project_roles."name" IS '角色名称';
COMMENT ON COLUMN project_roles."description" IS '角色描述';
COMMENT ON COLUMN project_roles."isSystem" IS '是否系统内置角色';
COMMENT ON COLUMN project_roles."status" IS '角色状态';
COMMENT ON COLUMN project_roles."createdAt" IS '创建时间';
COMMENT ON COLUMN project_roles."updatedAt" IS '更新时间';

CREATE TABLE IF NOT EXISTS project_user_role_bindings (
  "id" uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  "projectId" uuid NOT NULL REFERENCES projects ("id") ON DELETE CASCADE,
  "createdBy" uuid REFERENCES users ("id") ON DELETE SET NULL,
  "runtimeUserId" uuid NOT NULL REFERENCES project_runtime_users ("id") ON DELETE CASCADE,
  "roleId" uuid NOT NULL REFERENCES project_roles ("id") ON DELETE CASCADE,
  "assignedBy" uuid REFERENCES users ("id") ON DELETE SET NULL,
  "assignedAt" timestamptz,
  "createdAt" timestamptz NOT NULL DEFAULT now(),
  "updatedAt" timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS project_user_role_bindings_unique_idx
  ON project_user_role_bindings ("projectId", "runtimeUserId", "roleId");
CREATE INDEX IF NOT EXISTS project_user_role_bindings_user_idx
  ON project_user_role_bindings ("projectId", "runtimeUserId");
CREATE INDEX IF NOT EXISTS project_user_role_bindings_role_idx
  ON project_user_role_bindings ("projectId", "roleId");
CREATE INDEX IF NOT EXISTS project_user_role_bindings_created_by_idx
  ON project_user_role_bindings ("createdBy");

COMMENT ON TABLE project_user_role_bindings IS '工程运行态用户角色绑定表';
COMMENT ON COLUMN project_user_role_bindings."id" IS '绑定ID';
COMMENT ON COLUMN project_user_role_bindings."projectId" IS '所属工程ID';
COMMENT ON COLUMN project_user_role_bindings."createdBy" IS '创建者ID';
COMMENT ON COLUMN project_user_role_bindings."runtimeUserId" IS '运行态用户ID';
COMMENT ON COLUMN project_user_role_bindings."roleId" IS '角色ID';
COMMENT ON COLUMN project_user_role_bindings."assignedBy" IS '分配人ID';
COMMENT ON COLUMN project_user_role_bindings."assignedAt" IS '分配时间';
COMMENT ON COLUMN project_user_role_bindings."createdAt" IS '创建时间';
COMMENT ON COLUMN project_user_role_bindings."updatedAt" IS '更新时间';

CREATE TABLE IF NOT EXISTS project_role_grants (
  "id" uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  "projectId" uuid NOT NULL REFERENCES projects ("id") ON DELETE CASCADE,
  "createdBy" uuid REFERENCES users ("id") ON DELETE SET NULL,
  "updatedBy" uuid REFERENCES users ("id") ON DELETE SET NULL,
  "roleId" uuid NOT NULL REFERENCES project_roles ("id") ON DELETE CASCADE,
  "resourceType" text NOT NULL,
  "resourceId" text NOT NULL DEFAULT '*',
  "action" text NOT NULL,
  "effect" text NOT NULL DEFAULT 'allow' CHECK ("effect" IN ('allow', 'deny')),
  "scopeConfig" jsonb NOT NULL DEFAULT '{}'::jsonb,
  "condition" jsonb,
  "createdAt" timestamptz NOT NULL DEFAULT now(),
  "updatedAt" timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS project_role_grants_unique_idx
  ON project_role_grants ("projectId", "roleId", "resourceType", "resourceId", "action", "effect");
CREATE INDEX IF NOT EXISTS project_role_grants_role_idx
  ON project_role_grants ("projectId", "roleId");
CREATE INDEX IF NOT EXISTS project_role_grants_resource_action_idx
  ON project_role_grants ("projectId", "resourceType", "resourceId", "action");
CREATE INDEX IF NOT EXISTS project_role_grants_created_by_idx
  ON project_role_grants ("createdBy");
CREATE INDEX IF NOT EXISTS project_role_grants_updated_by_idx
  ON project_role_grants ("updatedBy");

COMMENT ON TABLE project_role_grants IS '工程运行态角色授权表';
COMMENT ON COLUMN project_role_grants."id" IS '授权ID';
COMMENT ON COLUMN project_role_grants."projectId" IS '所属工程ID';
COMMENT ON COLUMN project_role_grants."roleId" IS '角色ID';
COMMENT ON COLUMN project_role_grants."resourceType" IS '资源类型';
COMMENT ON COLUMN project_role_grants."action" IS '动作名称';
COMMENT ON COLUMN project_role_grants."effect" IS '授权效果';
COMMENT ON COLUMN project_role_grants."condition" IS '授权条件';
COMMENT ON COLUMN project_role_grants."createdAt" IS '创建时间';
COMMENT ON COLUMN project_role_grants."updatedAt" IS '更新时间';
COMMENT ON COLUMN project_role_grants."createdBy" IS '创建者ID';
COMMENT ON COLUMN project_role_grants."updatedBy" IS '更新者ID';
COMMENT ON COLUMN project_role_grants."resourceId" IS '资源实例ID，* 表示全部实例';
COMMENT ON COLUMN project_role_grants."scopeConfig" IS '授权范围配置';

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

COMMENT ON TABLE logs IS '系统日志表';
COMMENT ON COLUMN logs."id" IS '日志ID';
COMMENT ON COLUMN logs."tenantId" IS '租户ID';
COMMENT ON COLUMN logs."projectId" IS '工程ID';
COMMENT ON COLUMN logs."userId" IS '操作用户ID';
COMMENT ON COLUMN logs."level" IS '日志级别';
COMMENT ON COLUMN logs."category" IS '日志分类';
COMMENT ON COLUMN logs."action" IS '操作类型';
COMMENT ON COLUMN logs."resource" IS '资源类型';
COMMENT ON COLUMN logs."resourceId" IS '资源ID';
COMMENT ON COLUMN logs."message" IS '日志消息';
COMMENT ON COLUMN logs."details" IS '详细信息';
COMMENT ON COLUMN logs."ip" IS 'IP地址';
COMMENT ON COLUMN logs."userAgent" IS '用户代理';
COMMENT ON COLUMN logs."duration" IS '操作耗时(毫秒)';
COMMENT ON COLUMN logs."createdAt" IS '创建时间';

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

COMMENT ON TABLE design_pages IS '设计页面表';
COMMENT ON COLUMN design_pages."id" IS '页面ID';
COMMENT ON COLUMN design_pages."projectId" IS '所属工程ID';
COMMENT ON COLUMN design_pages."parentId" IS '父页面或文件夹ID';
COMMENT ON COLUMN design_pages."name" IS '页面名称';
COMMENT ON COLUMN design_pages."path" IS '路由路径';
COMMENT ON COLUMN design_pages."type" IS '页面类型';
COMMENT ON COLUMN design_pages."icon" IS '菜单图标';
COMMENT ON COLUMN design_pages."schemaVersion" IS 'DSL版本号';
COMMENT ON COLUMN design_pages."schemaContent" IS '页面DSL内容';
COMMENT ON COLUMN design_pages."pageConfig" IS '页面配置';
COMMENT ON COLUMN design_pages."variables" IS '页面级变量定义';
COMMENT ON COLUMN design_pages."dataSources" IS '页面级数据源配置';
COMMENT ON COLUMN design_pages."lifecycle" IS '页面生命周期配置';
COMMENT ON COLUMN design_pages."isHome" IS '是否首页';
COMMENT ON COLUMN design_pages."isPublished" IS '是否已发布';
COMMENT ON COLUMN design_pages."status" IS '页面状态';
COMMENT ON COLUMN design_pages."thumbnailUrl" IS '页面缩略图路径';
COMMENT ON COLUMN design_pages."sortOrder" IS '排序顺序';
COMMENT ON COLUMN design_pages."lockedBy" IS '当前锁定用户ID';
COMMENT ON COLUMN design_pages."lockedAt" IS '锁定时间';
COMMENT ON COLUMN design_pages."publishedAt" IS '最后发布时间';
COMMENT ON COLUMN design_pages."publishedBy" IS '最后发布人ID';
COMMENT ON COLUMN design_pages."createdBy" IS '创建者ID';
COMMENT ON COLUMN design_pages."updatedBy" IS '更新者ID';
COMMENT ON COLUMN design_pages."createdAt" IS '创建时间';
COMMENT ON COLUMN design_pages."updatedAt" IS '更新时间';

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

COMMENT ON TABLE design_asset_folders IS '设计资源文件夹表';
COMMENT ON COLUMN design_asset_folders."id" IS '文件夹ID';
COMMENT ON COLUMN design_asset_folders."projectId" IS '所属工程ID';
COMMENT ON COLUMN design_asset_folders."parentId" IS '父文件夹ID';
COMMENT ON COLUMN design_asset_folders."name" IS '文件夹名称';
COMMENT ON COLUMN design_asset_folders."sortOrder" IS '排序顺序';
COMMENT ON COLUMN design_asset_folders."createdAt" IS '创建时间';
COMMENT ON COLUMN design_asset_folders."updatedAt" IS '更新时间';

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

COMMENT ON TABLE design_assets IS '设计资源文件表';
COMMENT ON COLUMN design_assets."id" IS '资源ID';
COMMENT ON COLUMN design_assets."projectId" IS '所属工程ID';
COMMENT ON COLUMN design_assets."folderId" IS '所属文件夹ID';
COMMENT ON COLUMN design_assets."name" IS '资源名称';
COMMENT ON COLUMN design_assets."originalName" IS '原始文件名';
COMMENT ON COLUMN design_assets."type" IS '资源类型';
COMMENT ON COLUMN design_assets."mimeType" IS 'MIME类型';
COMMENT ON COLUMN design_assets."url" IS '资源访问路径';
COMMENT ON COLUMN design_assets."thumbnailUrl" IS '缩略图路径';
COMMENT ON COLUMN design_assets."size" IS '文件大小(字节)';
COMMENT ON COLUMN design_assets."width" IS '宽度';
COMMENT ON COLUMN design_assets."height" IS '高度';
COMMENT ON COLUMN design_assets."duration" IS '时长(毫秒或秒级元数据)';
COMMENT ON COLUMN design_assets."metadata" IS '扩展元数据';
COMMENT ON COLUMN design_assets."tags" IS '资源标签';
COMMENT ON COLUMN design_assets."usageCount" IS '使用次数';
COMMENT ON COLUMN design_assets."uploadedBy" IS '上传者ID';
COMMENT ON COLUMN design_assets."createdAt" IS '创建时间';
COMMENT ON COLUMN design_assets."updatedAt" IS '更新时间';

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

COMMENT ON TABLE design_project_settings IS '工程级设计配置表';
COMMENT ON COLUMN design_project_settings."projectId" IS '工程ID';
COMMENT ON COLUMN design_project_settings."schemaVersion" IS 'DSL版本号';
COMMENT ON COLUMN design_project_settings."globalVariables" IS '全局变量定义';
COMMENT ON COLUMN design_project_settings."globalStyles" IS '全局样式配置';
COMMENT ON COLUMN design_project_settings."globalScripts" IS '全局脚本配置';
COMMENT ON COLUMN design_project_settings."globalDataSources" IS '全局数据源配置';
COMMENT ON COLUMN design_project_settings."componentMappings" IS '自定义组件映射';
COMMENT ON COLUMN design_project_settings."i18n" IS '国际化配置';
COMMENT ON COLUMN design_project_settings."permissions" IS '全局权限配置';
COMMENT ON COLUMN design_project_settings."buildConfig" IS '构建配置';
COMMENT ON COLUMN design_project_settings."runtimeConfig" IS '运行时配置';
COMMENT ON COLUMN design_project_settings."updatedBy" IS '最后更新者ID';
COMMENT ON COLUMN design_project_settings."updatedAt" IS '最后更新时间';

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

COMMENT ON TABLE deployments IS '发布版本表';
COMMENT ON COLUMN deployments."id" IS '发布版本ID';
COMMENT ON COLUMN deployments."projectId" IS '工程ID';
COMMENT ON COLUMN deployments."tenantId" IS '租户ID';
COMMENT ON COLUMN deployments."version" IS '版本号';
COMMENT ON COLUMN deployments."name" IS '版本名称';
COMMENT ON COLUMN deployments."description" IS '版本描述';
COMMENT ON COLUMN deployments."type" IS '部署类型';
COMMENT ON COLUMN deployments."mode" IS '运行模式';
COMMENT ON COLUMN deployments."status" IS '构建状态';
COMMENT ON COLUMN deployments."buildConfig" IS '构建配置';
COMMENT ON COLUMN deployments."buildLog" IS '构建日志';
COMMENT ON COLUMN deployments."artifactUrl" IS '构建产物URL';
COMMENT ON COLUMN deployments."artifactHash" IS '产物SHA256哈希';
COMMENT ON COLUMN deployments."artifactSize" IS '产物大小(字节)';
COMMENT ON COLUMN deployments."snapshotUrl" IS '快照文件URL';
COMMENT ON COLUMN deployments."snapshotHash" IS '快照SHA256哈希';
COMMENT ON COLUMN deployments."manifest" IS '发布清单';
COMMENT ON COLUMN deployments."pageCount" IS '页面数量';
COMMENT ON COLUMN deployments."componentCount" IS '组件数量';
COMMENT ON COLUMN deployments."datapointCount" IS '数据点数量';
COMMENT ON COLUMN deployments."startedAt" IS '构建开始时间';
COMMENT ON COLUMN deployments."completedAt" IS '构建完成时间';
COMMENT ON COLUMN deployments."errorMessage" IS '错误信息';
COMMENT ON COLUMN deployments."deployedBy" IS '发布者ID';
COMMENT ON COLUMN deployments."createdAt" IS '创建时间';
COMMENT ON COLUMN deployments."updatedAt" IS '更新时间';
COMMENT ON COLUMN deployments."deletedAt" IS '软删除时间';

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

COMMENT ON TABLE nodes IS '运行时节点表';
COMMENT ON COLUMN nodes."id" IS '节点ID';
COMMENT ON COLUMN nodes."tenantId" IS '所属租户ID';
COMMENT ON COLUMN nodes."name" IS '节点名称';
COMMENT ON COLUMN nodes."description" IS '节点描述';
COMMENT ON COLUMN nodes."agentVersion" IS 'NodeAgent版本';
COMMENT ON COLUMN nodes."status" IS '节点状态';
COMMENT ON COLUMN nodes."currentProjectId" IS '当前运行工程ID';
COMMENT ON COLUMN nodes."currentVersion" IS '当前运行版本号';
COMMENT ON COLUMN nodes."currentDeploymentId" IS '当前部署记录ID';
COMMENT ON COLUMN nodes."ipAddress" IS '节点IP地址';
COMMENT ON COLUMN nodes."port" IS 'RuntimeEngine运行端口';
COMMENT ON COLUMN nodes."lastHeartbeatAt" IS '最后心跳时间';
COMMENT ON COLUMN nodes."lastErrorMessage" IS '最后错误信息';
COMMENT ON COLUMN nodes."lastErrorAt" IS '最后错误时间';
COMMENT ON COLUMN nodes."metrics" IS '运行指标';
COMMENT ON COLUMN nodes."config" IS '节点配置';
COMMENT ON COLUMN nodes."registrationToken" IS '注册令牌';
COMMENT ON COLUMN nodes."approvalStatus" IS '审批状态';
COMMENT ON COLUMN nodes."approvedAt" IS '审批时间';
COMMENT ON COLUMN nodes."approvedBy" IS '审批人ID';
COMMENT ON COLUMN nodes."registeredBy" IS '注册申请人ID';
COMMENT ON COLUMN nodes."mode" IS '节点模式';
COMMENT ON COLUMN nodes."createdBy" IS '创建者ID';
COMMENT ON COLUMN nodes."updatedBy" IS '更新者ID';
COMMENT ON COLUMN nodes."createdAt" IS '创建时间';
COMMENT ON COLUMN nodes."updatedAt" IS '更新时间';
COMMENT ON COLUMN nodes."deletedAt" IS '软删除时间';

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

COMMENT ON TABLE node_deployments IS '节点部署关系表';
COMMENT ON COLUMN node_deployments."id" IS '部署记录ID';
COMMENT ON COLUMN node_deployments."nodeId" IS '节点ID';
COMMENT ON COLUMN node_deployments."deploymentId" IS '发布版本ID';
COMMENT ON COLUMN node_deployments."projectId" IS '工程ID';
COMMENT ON COLUMN node_deployments."version" IS '版本号';
COMMENT ON COLUMN node_deployments."mode" IS '运行模式';
COMMENT ON COLUMN node_deployments."status" IS '部署状态';
COMMENT ON COLUMN node_deployments."runtimeConfig" IS '运行时配置';
COMMENT ON COLUMN node_deployments."deployedAt" IS '部署完成时间';
COMMENT ON COLUMN node_deployments."startedAt" IS '启动时间';
COMMENT ON COLUMN node_deployments."stoppedAt" IS '停止时间';
COMMENT ON COLUMN node_deployments."deployedBy" IS '部署操作者ID';
COMMENT ON COLUMN node_deployments."errorMessage" IS '错误信息';
COMMENT ON COLUMN node_deployments."errorStack" IS '错误堆栈';
COMMENT ON COLUMN node_deployments."deployLog" IS '部署日志';
COMMENT ON COLUMN node_deployments."runtimeMetrics" IS '运行时指标';
COMMENT ON COLUMN node_deployments."createdAt" IS '创建时间';
COMMENT ON COLUMN node_deployments."updatedAt" IS '更新时间';
COMMENT ON COLUMN node_deployments."deletedAt" IS '软删除时间';

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

COMMENT ON TABLE node_commands IS '节点命令表';
COMMENT ON COLUMN node_commands."id" IS '命令ID';
COMMENT ON COLUMN node_commands."tenantId" IS '租户ID';
COMMENT ON COLUMN node_commands."nodeId" IS '节点ID';
COMMENT ON COLUMN node_commands."deploymentId" IS '节点部署记录ID';
COMMENT ON COLUMN node_commands."projectId" IS '工程ID';
COMMENT ON COLUMN node_commands."type" IS '命令类型';
COMMENT ON COLUMN node_commands."status" IS '命令状态';
COMMENT ON COLUMN node_commands."payload" IS '命令负载';
COMMENT ON COLUMN node_commands."attempts" IS '已重试次数';
COMMENT ON COLUMN node_commands."maxAttempts" IS '最大重试次数';
COMMENT ON COLUMN node_commands."timeoutSeconds" IS '超时秒数';
COMMENT ON COLUMN node_commands."requestedAt" IS '请求时间';
COMMENT ON COLUMN node_commands."issuedAt" IS '下发时间';
COMMENT ON COLUMN node_commands."acknowledgedAt" IS '确认时间';
COMMENT ON COLUMN node_commands."completedAt" IS '完成时间';
COMMENT ON COLUMN node_commands."lastError" IS '最近错误信息';
COMMENT ON COLUMN node_commands."createdAt" IS '创建时间';
COMMENT ON COLUMN node_commands."updatedAt" IS '更新时间';
COMMENT ON COLUMN node_commands."deletedAt" IS '软删除时间';

COMMENT ON COLUMN project_role_grants."createdBy" IS '创建者ID';
COMMENT ON COLUMN project_role_grants."updatedBy" IS '更新者ID';
COMMENT ON COLUMN project_role_grants."resourceId" IS '资源实例ID，* 表示全部实例';
COMMENT ON COLUMN project_role_grants."scopeConfig" IS '授权范围配置';
