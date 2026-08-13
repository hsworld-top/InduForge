import { STORAGE_KEYS } from '../constants'

interface RememberMeCredentials {
  username?: string
  password?: string
  tenantCode?: string
  rememberMe?: boolean
}

/**
 * 本地存储工具类
 */
export class Storage {
  /**
   * 获取存储的值
   * @param key 存储键
   * @param defaultValue 默认值
   * @returns 存储的值或默认值
   */
  static get<T = unknown>(key: string, defaultValue: T | null = null): T | null {
    try {
      const item = localStorage.getItem(key)
      return item ? (JSON.parse(item) as T) : defaultValue
    } catch (error) {
      console.warn(`Storage get error for key "${key}":`, error)
      return defaultValue
    }
  }

  /**
   * 设置存储的值
   * @param key 存储键
   * @param value 要存储的值
   */
  static set(key: string, value: unknown): void {
    try {
      localStorage.setItem(key, JSON.stringify(value))
    } catch (error) {
      console.warn(`Storage set error for key "${key}":`, error)
    }
  }

  /**
   * 删除存储的值
   * @param key 存储键
   */
  static remove(key: string): void {
    try {
      localStorage.removeItem(key)
    } catch (error) {
      console.warn(`Storage remove error for key "${key}":`, error)
    }
  }

  /**
   * 清空所有存储
   */
  static clear(): void {
    try {
      localStorage.clear()
    } catch (error) {
      console.warn('Storage clear error:', error)
    }
  }

  /**
   * 按前缀枚举 localStorage 中的键。
   * 这个方法只负责扫描键名，不解析值，便于上层按业务前缀做映射恢复。
   * @param prefix 键名前缀
   * @returns 匹配到的键名列表
   */
  static getKeysByPrefix(prefix: string): string[] {
    try {
      const keys: string[] = []
      for (let index = 0; index < localStorage.length; index += 1) {
        const key = localStorage.key(index)
        if (key && key.startsWith(prefix)) {
          keys.push(key)
        }
      }
      return keys
    } catch (error) {
      console.warn(`Storage getKeysByPrefix error for prefix "${prefix}":`, error)
      return []
    }
  }

  /**
   * 获取用户信息
   * @returns 用户信息
   */
  static getUserInfo(): Record<string, unknown> | null {
    return this.get<Record<string, unknown>>(STORAGE_KEYS.USER_INFO)
  }

  /**
   * 设置用户信息
   * @param userInfo 用户信息
   */
  static setUserInfo(userInfo: Record<string, unknown>): void {
    this.set(STORAGE_KEYS.USER_INFO, userInfo)
  }

  /**
   * 获取租户ID
   * @returns 租户ID
   */
  static getTenantId(): string | null {
    return this.get<string>(STORAGE_KEYS.TENANT_ID)
  }

  /**
   * 设置租户ID
   * @param tenantId 租户ID
   */
  static setTenantId(tenantId: string): void {
    this.set(STORAGE_KEYS.TENANT_ID, tenantId)
  }

  /**
   * 获取主题设置
   * @returns 主题
   */
  static getTheme(): string {
    return this.get<string>(STORAGE_KEYS.THEME, 'light') ?? 'light'
  }

  /**
   * 设置主题
   * @param theme 主题
   */
  static setTheme(theme: string): void {
    this.set(STORAGE_KEYS.THEME, theme)
  }

  /**
   * 获取语言设置
   * @returns 语言
   */
  static getLanguage(): string {
    return this.get<string>(STORAGE_KEYS.LANGUAGE, 'zh') ?? 'zh'
  }

  /**
   * 设置语言
   * @param language 语言
   */
  static setLanguage(language: string): void {
    this.set(STORAGE_KEYS.LANGUAGE, language)
  }

  /**
   * 获取侧边栏折叠状态
   * @returns 是否折叠
   */
  static getSidebarCollapsed(): boolean {
    return this.get<boolean>(STORAGE_KEYS.SIDEBAR_COLLAPSED, true) ?? true
  }

  /**
   * 设置侧边栏折叠状态
   * @param collapsed 是否折叠
   */
  static setSidebarCollapsed(collapsed: boolean): void {
    this.set(STORAGE_KEYS.SIDEBAR_COLLAPSED, !!collapsed)
  }

  /**
   * 获取记住我的凭据
   * @returns 记住的凭据 {username, password, tenantCode, rememberMe}
   */
  static getRememberMeCredentials(): RememberMeCredentials | null {
    return this.get<RememberMeCredentials>(STORAGE_KEYS.REMEMBER_ME)
  }

  /**
   * 设置记住我的凭据
   * @param credentials 凭据 {username, password, tenantCode, rememberMe}
   */
  static setRememberMeCredentials(credentials?: RememberMeCredentials | null): void {
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
  static clearRememberMeCredentials(): void {
    this.remove(STORAGE_KEYS.REMEMBER_ME)
  }
}
