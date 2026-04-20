/**
 * 本地存储工具类
 *
 * 封装 localStorage，支持 JSON 序列化/反序列化，
 * 提供 Token、用户信息、租户、工程等常用键的便捷方法。
 * 异常时静默降级，不抛出错误。
 */

import { STORAGE_KEYS } from "@/constants";

export class Storage {
  static getRaw(key: string): string | null {
    try {
      return localStorage.getItem(key);
    } catch (error) {
      console.warn(`Storage get error for key "${key}":`, error);
      return null;
    }
  }

  static get<T = unknown>(key: string, defaultValue: T | null = null): T | null {
    try {
      const item = this.getRaw(key);
      return item ? (JSON.parse(item) as T) : defaultValue;
    } catch (error) {
      console.warn(`Storage get error for key "${key}":`, error);
      return defaultValue;
    }
  }

  static set(key: string, value: unknown): void {
    try {
      localStorage.setItem(key, JSON.stringify(value));
    } catch (error) {
      console.warn(`Storage set error for key "${key}":`, error);
    }
  }

  static remove(key: string): void {
    try {
      localStorage.removeItem(key);
    } catch (error) {
      console.warn(`Storage remove error for key "${key}":`, error);
    }
  }

  static clear(): void {
    try {
      localStorage.clear();
    } catch (error) {
      console.warn("Storage clear error:", error);
    }
  }

  static getToken(): string | null {
    return localStorage.getItem(STORAGE_KEYS.TOKEN);
  }

  static setToken(token: string): void {
    localStorage.setItem(STORAGE_KEYS.TOKEN, token);
  }

  static getRefreshToken(): string | null {
    return localStorage.getItem(STORAGE_KEYS.REFRESH_TOKEN);
  }

  static setRefreshToken(refreshToken: string): void {
    localStorage.setItem(STORAGE_KEYS.REFRESH_TOKEN, refreshToken);
  }

  static getUserInfo(): unknown {
    return this.get(STORAGE_KEYS.USER_INFO);
  }

  static setUserInfo(userInfo: unknown): void {
    this.set(STORAGE_KEYS.USER_INFO, userInfo);
  }

  static getTenantId(): string | null {
    return this.get(STORAGE_KEYS.TENANT_ID) as string | null;
  }

  static setTenantId(tenantId: string): void {
    this.set(STORAGE_KEYS.TENANT_ID, tenantId);
  }

  static getProjectId(): string | null {
    return this.get(STORAGE_KEYS.PROJECT_ID) as string | null;
  }

  static setProjectId(projectId: string): void {
    this.set(STORAGE_KEYS.PROJECT_ID, projectId);
  }

  static getTheme(): "light" | "dark" {
    const theme = this.get<"light" | "dark">(STORAGE_KEYS.THEME, "light");
    return theme === "dark" ? "dark" : "light";
  }

  static setTheme(theme: "light" | "dark"): void {
    this.set(STORAGE_KEYS.THEME, theme);
  }

  static getLanguage(): "zh" | "en" {
    const language = this.get<"zh" | "en">(STORAGE_KEYS.LANGUAGE, "zh");
    return language === "en" ? "en" : "zh";
  }

  static setLanguage(language: "zh" | "en"): void {
    this.set(STORAGE_KEYS.LANGUAGE, language);
  }

  static getDesignerTheme(): "light" | "dark" {
    const designerTheme = this.get<"light" | "dark">(STORAGE_KEYS.DESIGNER_THEME, null);
    if (designerTheme === "light" || designerTheme === "dark") {
      return designerTheme;
    }

    return this.getTheme();
  }

  static setDesignerTheme(theme: "light" | "dark"): void {
    this.set(STORAGE_KEYS.DESIGNER_THEME, theme);
  }

  static getDesignerLanguage(): "zh" | "en" {
    const designerLanguage = this.get<"zh" | "en">(STORAGE_KEYS.DESIGNER_LANGUAGE, null);
    if (designerLanguage === "zh" || designerLanguage === "en") {
      return designerLanguage;
    }

    return this.getLanguage();
  }

  static setDesignerLanguage(language: "zh" | "en"): void {
    this.set(STORAGE_KEYS.DESIGNER_LANGUAGE, language);
  }
}
