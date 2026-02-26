import { ROLES } from '@/constants'

const TAB_PERMISSION_MAP = {
  dashboard: [ROLES.SUPER_ADMIN, ROLES.SYSTEM_ADMIN, ROLES.PROJECT_ADMIN, ROLES.OPS_ADMIN, ROLES.USER_ADMIN, ROLES.USER],
  'tenant-management': [ROLES.SUPER_ADMIN],
  'user-management': [ROLES.SUPER_ADMIN, ROLES.SYSTEM_ADMIN, ROLES.USER_ADMIN],
  'project-management': [ROLES.SUPER_ADMIN, ROLES.SYSTEM_ADMIN, ROLES.PROJECT_ADMIN],
  'ops-management': [ROLES.SUPER_ADMIN, ROLES.SYSTEM_ADMIN, ROLES.OPS_ADMIN],
  'system-logs': [ROLES.SUPER_ADMIN, ROLES.SYSTEM_ADMIN, ROLES.OPS_ADMIN],
  'system-settings': [ROLES.SYSTEM_ADMIN],
}

const TAB_DENIED_MESSAGE_MAP = {
  'tenant-management': '只有超级管理员才能访问租户管理',
  'user-management': '仅超级管理员、系统管理员或用户管理员可访问用户管理',
  'project-management': '仅超级管理员、系统管理员或工程管理员可访问工程管理',
  'ops-management': '仅超级管理员、系统管理员或运维管理员可访问运维管理',
  'system-logs': '仅超级管理员、系统管理员或运维管理员可访问系统日志',
  'system-settings': '只有系统管理员才能访问系统设置',
}

/**
 * 检查当前角色是否在允许列表内。
 * @param {string} userRole - 用户角色
 * @param {string[]} allowedRoles - 允许的角色列表
 * @returns {boolean}
 */
export const hasRole = (userRole, allowedRoles = []) => {
  if (!userRole || !Array.isArray(allowedRoles) || allowedRoles.length === 0) {
    return false
  }
  return allowedRoles.includes(userRole)
}

/**
 * 检查用户是否有指定标签页访问权限。
 * @param {string} tabKey - 标签页标识
 * @param {string} userRole - 用户角色
 * @returns {boolean}
 */
export const canAccessTab = (tabKey, userRole) => {
  const allowedRoles = TAB_PERMISSION_MAP[tabKey]
  if (!allowedRoles) return true
  return hasRole(userRole, allowedRoles)
}

/**
 * 获取标签页权限不足提示文案。
 * @param {string} tabKey - 标签页标识
 * @returns {string}
 */
export const getTabAccessDeniedMessage = (tabKey) => {
  return TAB_DENIED_MESSAGE_MAP[tabKey] || '您没有访问该功能的权限'
}

/**
 * 检查当前用户是否可管理用户。
 * @param {string} userRole - 用户角色
 * @returns {boolean}
 */
export const canManageUsers = (userRole) => {
  return hasRole(userRole, [ROLES.SUPER_ADMIN, ROLES.SYSTEM_ADMIN, ROLES.USER_ADMIN])
}

/**
 * 检查当前用户是否可审批节点接入。
 * @param {string} userRole - 用户角色
 * @returns {boolean}
 */
export const canApproveNodes = (userRole) => {
  return hasRole(userRole, [ROLES.SYSTEM_ADMIN, ROLES.OPS_ADMIN])
}
