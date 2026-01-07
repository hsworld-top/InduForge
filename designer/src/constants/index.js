/**
 * 常量定义
 */

// API 相关常量
export const API_BASE_URL = "/api/v1";
export const TIME_FORMAT = "YYYY-MM-DD HH:mm:ss";

// 本地存储键名常量
export const STORAGE_KEYS = {
  TOKEN: "auth_token",
  REFRESH_TOKEN: "refresh_token",
  USER_INFO: "user_info",
  THEME: "theme",
  LANGUAGE: "language",
  TENANT_ID: "tenant_id",
  PROJECT_ID: "project_id",
};

// 数据模式
export const DATA_MODE = {
  EDIT: "edit",
  PREVIEW: "preview",
  RUNTIME: "runtime",
};

/**
 * 视图预设（编辑器/预览通用）
 * 对齐断点：大屏>=1200，平板<=992，手机竖向<=768，手机竖屏<=480
 */
export const VIEW_PRESETS = [
  { key: "bigscreen", label: "大屏", width: 1920, height: 1080 },
  { key: "pc", label: "PC", width: 1366, height: 768 },
  { key: "tablet", label: "平板", width: 992, height: 744 },
  { key: "phoneLandscape", label: "手机竖向", width: 768, height: 1024 },
  { key: "phonePortrait", label: "手机竖屏", width: 480, height: 800 },
];
