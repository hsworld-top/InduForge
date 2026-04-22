import { ROLES } from '../constants/index.js'
import type { Role } from '../types/auth'

type Capability =
  | '*'
  | 'tenant:manage'
  | 'project:read'
  | 'project:write'
  | 'release:publish'
  | 'deploy:execute'
  | 'runtime:operate'
  | 'node:read'
  | 'node:approve'
  | 'user:write'

const ROLE_CAPABILITY_MAP: Record<Role, readonly Capability[]> = {
  [ROLES.SUPER_ADMIN]: ['tenant:manage'],
  [ROLES.SYSTEM_ADMIN]: ['*'],
  [ROLES.PROJECT_ADMIN]: [
    'project:read',
    'project:write',
    'release:publish',
    'deploy:execute',
    'runtime:operate',
    'node:read',
  ],
  [ROLES.OPS_ADMIN]: [
    'project:read',
    'release:publish',
    'deploy:execute',
    'runtime:operate',
    'node:read',
    'node:approve',
  ],
  [ROLES.USER_ADMIN]: ['user:write'],
  [ROLES.USER]: ['project:read'],
}

export const TAB_PERMISSION_MAP: Record<string, readonly Role[]> = {
  dashboard: [
    ROLES.SYSTEM_ADMIN,
    ROLES.PROJECT_ADMIN,
    ROLES.OPS_ADMIN,
    ROLES.USER_ADMIN,
    ROLES.USER,
  ],
  'tenant-management': [ROLES.SUPER_ADMIN],
  'user-management': [ROLES.SYSTEM_ADMIN, ROLES.USER_ADMIN],
  'project-management': [ROLES.SYSTEM_ADMIN, ROLES.PROJECT_ADMIN, ROLES.OPS_ADMIN],
  'ops-management': [ROLES.SYSTEM_ADMIN, ROLES.PROJECT_ADMIN, ROLES.OPS_ADMIN],
  'system-logs': [ROLES.SYSTEM_ADMIN, ROLES.OPS_ADMIN],
  'system-settings': [ROLES.SYSTEM_ADMIN],
}

export const TAB_DENIED_MESSAGE_MAP: Record<string, string> = {
  'tenant-management': '只有超级管理员才能访问租户管理',
  'user-management': '仅系统管理员或用户管理员可访问用户管理',
  'project-management': '仅系统管理员、工程管理员或运维管理员可访问工程管理',
  'ops-management': '仅系统管理员、工程管理员或运维管理员可访问运维管理',
  'system-logs': '仅系统管理员或运维管理员可访问系统日志',
  'system-settings': '只有系统管理员才能访问系统设置',
}

export const hasRole = (userRole: Role | null | undefined, allowedRoles: readonly Role[] = []) => {
  if (!userRole || !Array.isArray(allowedRoles) || allowedRoles.length === 0) {
    return false
  }
  return allowedRoles.includes(userRole)
}

export const can = (userRole: Role | null | undefined, capability: Capability | string | null | undefined) => {
  if (!userRole || !capability) return false
  const caps = ROLE_CAPABILITY_MAP[userRole] || []
  return caps.includes('*') || caps.includes(capability as Capability)
}

export const canAccessTab = (tabKey: string, userRole: Role | null | undefined) => {
  const allowedRoles = TAB_PERMISSION_MAP[tabKey]
  if (!allowedRoles) return true
  return hasRole(userRole, allowedRoles)
}

export const getTabAccessDeniedMessage = (tabKey: string) => {
  return TAB_DENIED_MESSAGE_MAP[tabKey] || '您没有访问该功能的权限'
}

export const canManageUsers = (userRole: Role | null | undefined) => {
  return can(userRole, 'user:write')
}

export const canApproveNodes = (userRole: Role | null | undefined) => {
  return can(userRole, 'node:approve')
}

/**
 * 判断当前角色是否允许请求租户统计。
 * @param userRole 当前用户角色
 * @returns 是否允许请求租户统计
 */
export const canRequestTenantStats = (userRole: Role | null | undefined) => {
  return userRole === ROLES.SUPER_ADMIN && canAccessTab('tenant-management', userRole)
}
