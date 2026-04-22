import { ROLES } from '../constants'

// 角色枚举
export const RoleEnum = ROLES

// 工程状态枚举
export const ProjectStatusEnum = {
  PLANNING: 'planning',
  ACTIVE: 'active',
  MAINTENANCE: 'maintenance',
  COMPLETED: 'completed',
  ARCHIVED: 'archived',
} as const

// 优先级枚举
export const PriorityEnum = {
  LOW: 'low',
  MEDIUM: 'medium',
  HIGH: 'high',
  URGENT: 'urgent',
} as const

// 租户状态枚举
export const TenantStatusEnum = {
  ACTIVE: 'active',
  INACTIVE: 'inactive',
  SUSPENDED: 'suspended',
} as const

// 用户状态枚举
export const UserStatusEnum = {
  ACTIVE: 'active',
  INACTIVE: 'inactive',
  SUSPENDED: 'suspended',
} as const

export const ColorTagEnum = {
  BLUE: '#3b82f6',
  RED: '#ef4444',
  GREEN: '#10b981',
  YELLOW: '#f59e0b',
  PURPLE: '#8b5cf6',
  PINK: '#ec4899',
  GRAY: '#6b7280',
} as const

// 主题枚举
export const ThemeEnum = {
  LIGHT: 'light',
  DARK: 'dark',
} as const

// 语言枚举
export const LanguageEnum = {
  ZH: 'zh',
  EN: 'en',
} as const

const ROLE_LABELS: Record<string, string> = {
  [RoleEnum.SUPER_ADMIN]: '超级管理员',
  [RoleEnum.SYSTEM_ADMIN]: '系统管理员',
  [RoleEnum.PROJECT_ADMIN]: '工程管理员',
  [RoleEnum.OPS_ADMIN]: '运维管理员',
  [RoleEnum.USER_ADMIN]: '用户管理员',
  [RoleEnum.USER]: '普通用户',
}

const PROJECT_STATUS_LABELS: Record<string, string> = {
  [ProjectStatusEnum.PLANNING]: '规划中',
  [ProjectStatusEnum.ACTIVE]: '进行中',
  [ProjectStatusEnum.MAINTENANCE]: '维护中',
  [ProjectStatusEnum.COMPLETED]: '已完成',
  [ProjectStatusEnum.ARCHIVED]: '已归档',
}

const PRIORITY_LABELS: Record<string, string> = {
  [PriorityEnum.LOW]: '低',
  [PriorityEnum.MEDIUM]: '中',
  [PriorityEnum.HIGH]: '高',
  [PriorityEnum.URGENT]: '紧急',
}

const TENANT_STATUS_LABELS: Record<string, string> = {
  [TenantStatusEnum.ACTIVE]: '激活',
  [TenantStatusEnum.INACTIVE]: '未激活',
  [TenantStatusEnum.SUSPENDED]: '暂停',
}

const USER_STATUS_LABELS: Record<string, string> = {
  [UserStatusEnum.ACTIVE]: '激活',
  [UserStatusEnum.INACTIVE]: '未激活',
  [UserStatusEnum.SUSPENDED]: '暂停',
}

const COLOR_TAG_LABELS: Record<string, string> = {
  [ColorTagEnum.BLUE]: '蓝色',
  [ColorTagEnum.RED]: '红色',
  [ColorTagEnum.GREEN]: '绿色',
  [ColorTagEnum.YELLOW]: '黄色',
  [ColorTagEnum.PURPLE]: '紫色',
  [ColorTagEnum.PINK]: '粉色',
  [ColorTagEnum.GRAY]: '灰色',
}

const THEME_LABELS: Record<string, string> = {
  [ThemeEnum.LIGHT]: '浅色',
  [ThemeEnum.DARK]: '深色',
}

const LANGUAGE_LABELS: Record<string, string> = {
  [LanguageEnum.ZH]: '中文',
  [LanguageEnum.EN]: 'English',
}

// 枚举映射（用于显示）
export const ENUM_LABELS: Record<string, string> = {
  ...ROLE_LABELS,
  ...PROJECT_STATUS_LABELS,
  ...PRIORITY_LABELS,
  ...TENANT_STATUS_LABELS,
  ...USER_STATUS_LABELS,
  ...COLOR_TAG_LABELS,
  ...THEME_LABELS,
  ...LANGUAGE_LABELS,
}
