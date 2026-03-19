/**
 * 设计器常量定义
 *
 * 集中管理 API、存储键、数据模式、视图预设、网格配置等常量，
 * 便于维护与多环境配置。
 */

/** API 基础路径，与后端 dev_core 保持一致 */
export const API_BASE_URL = "/api/v1";

/** 时间展示格式，与 AGENTS.md 约定一致 */
export const TIME_FORMAT = "YYYY-MM-DD HH:mm:ss";

/**
 * 本地存储键名常量
 * 用于 Storage 工具类，避免硬编码
 */
export const STORAGE_KEYS = {
  TOKEN: "auth_token",
  REFRESH_TOKEN: "refresh_token",
  USER_INFO: "user_info",
  THEME: "theme",
  LANGUAGE: "language",
  TENANT_ID: "tenant_id",
  PROJECT_ID: "project_id",
};

/**
 * 数据模式
 * - edit: 设计态，使用 Mock 数据
 * - preview: 预览态，使用真实数据
 * - runtime: 运行态，由独立 RuntimeEngine 处理
 */
export const DATA_MODE = {
  EDIT: "edit",
  PREVIEW: "preview",
  RUNTIME: "runtime",
};

/**
 * 视图预设（编辑器/预览通用）
 * 用于画布尺寸切换与响应式预览
 * 对齐断点：大屏>=1200，平板<=992，手机竖向<=768，手机竖屏<=480
 */
export const VIEW_PRESETS = [
  { key: "bigscreen", label: "大屏", width: 1920, height: 1080 },
  { key: "pc", label: "PC", width: 1366, height: 768 },
  { key: "tablet", label: "平板", width: 992, height: 744 },
  { key: "phoneLandscape", label: "手机竖向", width: 768, height: 1024 },
  { key: "phonePortrait", label: "手机竖屏", width: 480, height: 800 },
];

/**
 * 画布网格配置
 * 用于拖拽吸附与辅助线显示
 */
export const GRID_CONFIG = {
  /** 网格大小（px），吸附步长 */
  size: 10,
  /** 是否启用拖拽吸附 */
  enabled: true,
  /** 是否显示网格线（暂不支持） */
  showGrid: false,
};
