import { STORAGE_KEYS } from '@/constants'

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
      const item = localStorage.getItem(key)
      return item ? JSON.parse(item) : defaultValue
    } catch (error) {
      console.warn(`Storage get error for key "${key}":`, error)
      return defaultValue
    }
  }

  /**
   * 设置存储的值
   * @param {string} key - 存储键
   * @param {*} value - 要存储的值
   */
  static set(key, value) {
    try {
      localStorage.setItem(key, JSON.stringify(value))
    } catch (error) {
      console.warn(`Storage set error for key "${key}":`, error)
    }
  }

  /**
   * 删除存储的值
   * @param {string} key - 存储键
   */
  static remove(key) {
    try {
      localStorage.removeItem(key)
    } catch (error) {
      console.warn(`Storage remove error for key "${key}":`, error)
    }
  }

  /**
   * 清空所有存储
   */
  static clear() {
    try {
      localStorage.clear()
    } catch (error) {
      console.warn('Storage clear error:', error)
    }
  }

  /**
   * 获取认证令牌
   * @returns {string|null} 令牌
   */
  static getToken() {
    return localStorage.getItem(STORAGE_KEYS.TOKEN)
  }

  /**
   * 设置认证令牌
   * @param {string} token - 令牌
   */
  static setToken(token) {
    localStorage.setItem(STORAGE_KEYS.TOKEN, token)
  }

  /**
   * 获取刷新令牌
   * @returns {string|null} 刷新令牌
   */
  static getRefreshToken() {
    return localStorage.getItem(STORAGE_KEYS.REFRESH_TOKEN)
  }

  /**
   * 设置刷新令牌
   * @param {string} refreshToken - 刷新令牌
   */
  static setRefreshToken(refreshToken) {
    localStorage.setItem(STORAGE_KEYS.REFRESH_TOKEN, refreshToken)
  }

  /**
   * 获取用户信息
   * @returns {object|null} 用户信息
   */
  static getUserInfo() {
    return this.get(STORAGE_KEYS.USER_INFO)
  }

  /**
   * 设置用户信息
   * @param {object} userInfo - 用户信息
   */
  static setUserInfo(userInfo) {
    this.set(STORAGE_KEYS.USER_INFO, userInfo)
  }

  /**
   * 获取租户ID
   * @returns {string|null} 租户ID
   */
  static getTenantId() {
    return this.get(STORAGE_KEYS.TENANT_ID)
  }

  /**
   * 设置租户ID
   * @param {string} tenantId - 租户ID
   */
  static setTenantId(tenantId) {
    this.set(STORAGE_KEYS.TENANT_ID, tenantId)
  }

  /**
   * 获取主题设置
   * @returns {string} 主题
   */
  static getTheme() {
    return this.get(STORAGE_KEYS.THEME, 'light')
  }

  /**
   * 设置主题
   * @param {string} theme - 主题
   */
  static setTheme(theme) {
    this.set(STORAGE_KEYS.THEME, theme)
  }

  /**
   * 获取语言设置
   * @returns {string} 语言
   */
  static getLanguage() {
    return this.get(STORAGE_KEYS.LANGUAGE, 'zh')
  }

  /**
   * 设置语言
   * @param {string} language - 语言
   */
  static setLanguage(language) {
    this.set(STORAGE_KEYS.LANGUAGE, language)
  }

  /**
   * 获取记住我的凭据
   * @returns {object|null} 记住的凭据 {username, password, tenantCode, rememberMe}
   */
  static getRememberMeCredentials() {
    return this.get(STORAGE_KEYS.REMEMBER_ME)
  }

  /**
   * 设置记住我的凭据
   * @param {object} credentials - 凭据 {username, password, tenantCode, rememberMe}
   */
  static setRememberMeCredentials(credentials) {
    if (credentials && credentials.rememberMe) {
      this.set(STORAGE_KEYS.REMEMBER_ME, {
        username: credentials.username,
        password: credentials.password,
        tenantCode: credentials.tenantCode,
        rememberMe: true,
      })
    } else {
      this.remove(STORAGE_KEYS.REMEMBER_ME)
    }
  }

  /**
   * 清除记住我的凭据
   */
  static clearRememberMeCredentials() {
    this.remove(STORAGE_KEYS.REMEMBER_ME)
  }
}
