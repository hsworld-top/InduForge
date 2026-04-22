// @ts-nocheck
import { STORAGE_KEYS } from "../constants/index";

const PROJECT_ID_STORAGE_KEY = STORAGE_KEYS.PROJECT_ID || "project_id";

/**
 * 本地存储工具类
 */
export class Storage {
  /**
   * 获取存储的值
   * @param {string} key - 存储键
   * @param {*} defaultValue - 默认值
   * @returns {*} 存储的值或默认值
   */
  static get(key, defaultValue = null) {
    try {
      const item = localStorage.getItem(key);
      return item ? JSON.parse(item) : defaultValue;
    } catch (error) {
      console.warn(`Storage get error for key "${key}":`, error);
      return defaultValue;
    }
  }

  /**
   * 设置存储的值
   * @param {string} key - 存储键
   * @param {*} value - 要存储的值
   */
  static set(key, value) {
    try {
      localStorage.setItem(key, JSON.stringify(value));
    } catch (error) {
      console.warn(`Storage set error for key "${key}":`, error);
    }
  }

  /**
   * 删除存储的值
   * @param {string} key - 存储键
   */
  static remove(key) {
    try {
      localStorage.removeItem(key);
    } catch (error) {
      console.warn(`Storage remove error for key "${key}":`, error);
    }
  }

  /**
   * 清空所有存储
   */
  static clear() {
    try {
      localStorage.clear();
    } catch (error) {
      console.warn("Storage clear error:", error);
    }
  }

  /**
   * 获取认证令牌
   * @returns {string|null} 令牌
   */
  static getToken() {
    return localStorage.getItem(STORAGE_KEYS.TOKEN);
  }

  /**
   * 设置认证令牌
   * @param {string} token - 令牌
   */
  static setToken(token) {
    localStorage.setItem(STORAGE_KEYS.TOKEN, token);
  }

  /**
   * 获取刷新令牌
   * @returns {string|null} 刷新令牌
   */
  static getRefreshToken() {
    return localStorage.getItem(STORAGE_KEYS.REFRESH_TOKEN);
  }

  /**
   * 设置刷新令牌
   * @param {string} refreshToken - 刷新令牌
   */
  static setRefreshToken(refreshToken) {
    localStorage.setItem(STORAGE_KEYS.REFRESH_TOKEN, refreshToken);
  }

  /**
   * 获取用户信息
   * @returns {object|null} 用户信息
   */
  static getUserInfo() {
    return this.get(STORAGE_KEYS.USER_INFO);
  }

  /**
   * 设置用户信息
   * @param {object} userInfo - 用户信息
   */
  static setUserInfo(userInfo) {
    this.set(STORAGE_KEYS.USER_INFO, userInfo);
  }

  /**
   * 获取租户ID
   * @returns {string|null} 租户ID
   */
  static getTenantId() {
    return this.get(STORAGE_KEYS.TENANT_ID);
  }

  /**
   * 设置租户ID
   * @param {string} tenantId - 租户ID
   */
  static setTenantId(tenantId) {
    this.set(STORAGE_KEYS.TENANT_ID, tenantId);
  }

  /**
   * 获取工程 ID。
   * 当前 constants 尚未暴露 PROJECT_ID，故这里保留固定键兜底，
   * 避免 bootstrap 恢复链路因为常量缺失而无法写回工程上下文。
   * @returns {string|null} 工程 ID
   */
  static getProjectId() {
    return this.get(PROJECT_ID_STORAGE_KEY);
  }

  /**
   * 设置工程 ID。
   * @param {string} projectId - 工程 ID
   */
  static setProjectId(projectId) {
    this.set(PROJECT_ID_STORAGE_KEY, projectId);
  }

  /**
   * 删除工程 ID。
   */
  static removeProjectId() {
    this.remove(PROJECT_ID_STORAGE_KEY);
  }

  /**
   * 获取主题。
   * @returns {string} 主题
   */
  static getTheme() {
    return this.get(STORAGE_KEYS.THEME, "light");
  }

  /**
   * 设置主题。
   * @param {string} theme - 主题
   */
  static setTheme(theme) {
    this.set(STORAGE_KEYS.THEME, theme);
  }

  /**
   * 获取语言。
   * @returns {string|null} 语言
   */
  static getLanguage() {
    return this.get(STORAGE_KEYS.LANGUAGE);
  }

  /**
   * 设置语言。
   * @param {string} language - 语言
   */
  static setLanguage(language) {
    this.set(STORAGE_KEYS.LANGUAGE, language);
  }
}
