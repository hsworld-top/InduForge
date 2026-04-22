import { describe, expect, it } from 'vitest'

import { ROLES } from '@/constants'
import { canAccessTab, canManageUsers, hasRole } from '@/permissions'
import type { Role, UserInfo } from '@/types/auth'

const createUser = (role: Role): UserInfo => ({
  id: 'user-1',
  username: 'tester',
  email: 'tester@example.com',
  nickname: '测试用户',
  role,
})

describe('permissions/rules', () => {
  describe('hasRole', () => {
    it('角色命中时返回 true，未命中时返回 false', () => {
      const user = createUser(ROLES.SYSTEM_ADMIN)

      expect(hasRole(user.role, [ROLES.SYSTEM_ADMIN, ROLES.OPS_ADMIN])).toBe(true)
      expect(hasRole(user.role, [ROLES.USER, ROLES.PROJECT_ADMIN])).toBe(false)
    })
  })

  describe('canAccessTab', () => {
    it('根据标签角色映射返回访问结果', () => {
      expect(canAccessTab('tenant-management', ROLES.SUPER_ADMIN)).toBe(true)
      expect(canAccessTab('tenant-management', ROLES.SYSTEM_ADMIN)).toBe(false)
      expect(canAccessTab('dashboard', ROLES.USER)).toBe(true)
      expect(canAccessTab('unknown-tab', ROLES.USER)).toBe(true)
    })
  })

  describe('canManageUsers', () => {
    it('用户管理员和系统管理员具备用户管理能力', () => {
      expect(canManageUsers(ROLES.USER_ADMIN)).toBe(true)
      expect(canManageUsers(ROLES.SYSTEM_ADMIN)).toBe(true)
      expect(canManageUsers(ROLES.USER)).toBe(false)
    })
  })
})
