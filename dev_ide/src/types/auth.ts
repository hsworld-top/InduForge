/**
 * 角色选项常量，用于表单或下拉框场景。
 */
export const ROLE_OPTIONS = [
  'SUPER_ADMIN',
  'SYSTEM_ADMIN',
  'PROJECT_ADMIN',
  'OPS_ADMIN',
  'USER_ADMIN',
  'USER',
] as const

/**
 * 平台支持的角色集合。
 */
export type Role = (typeof ROLE_OPTIONS)[number]

/**
 * 登录态 token 数据。
 */
export interface AuthTokens {
  token: string
  refreshToken?: string
}

/**
 * 用户所属租户的精简信息。
 */
export interface UserTenant {
  id?: string | number
  code?: string
  name?: string
}

/**
 * 当前登录用户核心信息。
 */
export interface UserInfo {
  id: string | number
  username: string
  role: Role
  email?: string
  nickname?: string
  avatar?: string
  avatarUrl?: string
  tenantId?: string | number | null
  tenant?: UserTenant
}
