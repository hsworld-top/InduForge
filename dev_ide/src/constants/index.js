// API 相关常量
export const API_BASE_URL = '/api/v1'
export const TIME_FORMAT = 'YYYY-MM-DD HH:mm:ss'

// 路由名称常量
export const ROUTE_NAMES = {
  LOGIN: 'login',
  DASHBOARD: 'dashboard',
  TENANT_MANAGEMENT: 'tenant-management',
  USER_MANAGEMENT: 'user-management',
  PROJECT_MANAGEMENT: 'project-management',
  SYSTEM_SETTINGS: 'system-settings',
  PROFILE: 'profile',
}

// 角色常量
export const ROLES = {
  SUPER_ADMIN: 'SUPER_ADMIN',
  SYSTEM_ADMIN: 'SYSTEM_ADMIN',
  PROJECT_ADMIN: 'PROJECT_ADMIN',
  OPS_ADMIN: 'OPS_ADMIN',
  USER_ADMIN: 'USER_ADMIN',
  USER: 'USER',
}

// 工程状态常量
export const PROJECT_STATUS = {
  PLANNING: 'planning',
  ACTIVE: 'active',
  MAINTENANCE: 'maintenance',
  COMPLETED: 'completed',
  ARCHIVED: 'archived',
}

// 优先级常量
export const PRIORITY_LEVELS = {
  LOW: 'low',
  MEDIUM: 'medium',
  HIGH: 'high',
  URGENT: 'urgent',
}

// 主题常量
export const THEMES = {
  LIGHT: 'light',
  DARK: 'dark',
}

// 语言常量
export const LANGUAGES = {
  ZH: 'zh',
  EN: 'en',
}

// 本地存储键名常量
export const STORAGE_KEYS = {
  TOKEN: 'auth_token',
  REFRESH_TOKEN: 'refresh_token',
  USER_INFO: 'user_info',
  THEME: 'theme',
  LANGUAGE: 'language',
  TENANT_ID: 'tenant_id',
  REMEMBER_ME: 'remember_me',
  SIDEBAR_COLLAPSED: 'sidebar_collapsed',
  DASHBOARD_TAB_STATE: 'dashboard_tab_state',
  OPS_OPEN_PENDING_REQUEST: 'ops_open_pending_request',
  SYSTEM_LOG_SAVED_VIEWS: 'system_log_saved_views',
  SYSTEM_LOG_LAST_VIEW_ID: 'system_log_last_view_id',
}

// 分页常量
export const PAGINATION = {
  DEFAULT_PAGE_SIZE: 20,
  PAGE_SIZE_OPTIONS: [10, 20, 50, 100],
}

export const SYSTEM_LOG_PAGE_SIZE_OPTIONS = [10, 20, 50, 100]

// 正则表达式常量
export const REGEX = {
  EMAIL: /^[^\s@]+@[^\s@]+\.[^\s@]+$/,
  PASSWORD: /^(?=.*[a-z])(?=.*[A-Z])(?=.*\d)[a-zA-Z\d@$!%*?&]{8,}$/,
  PHONE: /^1[3-9]\d{9}$/,
}
