/**
 * 设计器常量定义
 */

export const API_BASE_URL = '/api/v1'

export const TIME_FORMAT = 'YYYY-MM-DD HH:mm:ss'

export const STORAGE_KEYS = {
  USER_INFO: 'user_info',
  THEME: 'theme',
  LANGUAGE: 'language',
  TENANT_ID: 'tenant_id',
  PROJECT_ID: 'project_id',
} as const

export const DATA_MODE = {
  EDIT: 'edit',
  PREVIEW: 'preview',
  RUNTIME: 'runtime',
} as const

export const VIEW_PRESETS = [
  { key: 'bigscreen', label: '大屏', width: 1920, height: 1080 },
  { key: 'pc', label: 'PC', width: 1366, height: 768 },
  { key: 'tablet', label: '平板', width: 992, height: 744 },
  { key: 'phoneLandscape', label: '手机竖向', width: 768, height: 1024 },
  { key: 'phonePortrait', label: '手机竖屏', width: 480, height: 800 },
] as const

/** 画布视图预设（与 VIEW_PRESETS 单项结构一致） */
export type ViewPreset = (typeof VIEW_PRESETS)[number]

export const GRID_CONFIG = {
  size: 10,
  enabled: true,
  showGrid: true,
} as const
