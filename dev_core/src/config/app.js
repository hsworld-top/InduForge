require('dotenv').config();

/**
 * 应用配置
 * 从环境变量读取配置，提供默认值
 */
const appConfig = {
  // 超级管理员配置
  superAdmin: {
    username: process.env.SUPER_ADMIN_USERNAME || 'superadmin',
    password: process.env.SUPER_ADMIN_PASSWORD || 'admin123',
    userId: process.env.SUPER_ADMIN_USER_ID || '550e8400-e29b-41d4-a716-446655440001',
    role: 'SUPER_ADMIN',
  },

  // 默认租户配置
  defaultTenant: {
    id: process.env.DEFAULT_TENANT_ID || '550e8400-e29b-41d4-a716-446655440000',
    name: process.env.DEFAULT_TENANT_NAME || '默认租户',
    code: process.env.DEFAULT_TENANT_CODE || 'default',
    description: process.env.DEFAULT_TENANT_DESCRIPTION || '系统默认租户',
    contactEmail: process.env.DEFAULT_TENANT_CONTACT_EMAIL || 'admin@example.com',
    maxUsers: Number(process.env.DEFAULT_TENANT_MAX_USERS || 100),
    maxProjects: Number(process.env.DEFAULT_TENANT_MAX_PROJECTS || 50),
  },

  // 租户创建时的默认配置
  tenantDefaults: {
    defaultAdminUsername: process.env.TENANT_DEFAULT_ADMIN_USERNAME || 'admin',
    defaultAdminPassword: process.env.TENANT_DEFAULT_ADMIN_PASSWORD || 'admin123',
    defaultMaxUsers: Number(process.env.TENANT_DEFAULT_MAX_USERS || 100),
    defaultMaxProjects: Number(process.env.TENANT_DEFAULT_MAX_PROJECTS || 50),
  },

  // 分页配置
  pagination: {
    defaultPage: Number(process.env.PAGINATION_DEFAULT_PAGE || 1),
    defaultLimit: Number(process.env.PAGINATION_DEFAULT_LIMIT || 10),
    maxLimit: Number(process.env.PAGINATION_MAX_LIMIT || 100),
  },

  // JWT 配置
  jwt: {
    accessExpiresIn: process.env.JWT_ACCESS_EXPIRES_IN || process.env.JWT_EXPIRES_IN || '15m',
    refreshExpiresIn: process.env.JWT_REFRESH_EXPIRES_IN || '7d',
  },
};

module.exports = appConfig;

