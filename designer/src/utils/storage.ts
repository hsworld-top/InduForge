/**
 * 本地存储工具类
 */

import { STORAGE_KEYS } from "@/constants";

export class Storage {
  static get(key: string, defaultValue: unknown = null): unknown {
    try {
      const item = localStorage.getItem(key);
      return item ? (JSON.parse(item) as unknown) : defaultValue;
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

  static setToken(token: string | undefined): void {
    if (token === undefined) return;
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

  static getTenantId(): unknown {
    return this.get(STORAGE_KEYS.TENANT_ID);
  }

  static setTenantId(tenantId: string): void {
    this.set(STORAGE_KEYS.TENANT_ID, tenantId);
  }

  static getProjectId(): unknown {
    return this.get(STORAGE_KEYS.PROJECT_ID);
  }

  static setProjectId(projectId: string): void {
    this.set(STORAGE_KEYS.PROJECT_ID, projectId);
  }
}
