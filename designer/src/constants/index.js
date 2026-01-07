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
