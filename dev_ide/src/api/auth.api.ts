import request from '@/utils/request'
import type { AxiosRequestConfig } from 'axios'
import type { ApiResponse, ApiRecord } from '@/types/api'

type LoginCredentials = {
  username: string
  password: string
  sliderChallengeId?: string
  sliderOffset?: number
  tenantCode?: string
}

type PasswordData = {
  oldPassword: string
  newPassword: string
}

export interface AuthLoginPayload {
  user: AuthUserPayload
}

export interface AuthConfigPayload extends ApiRecord {
  name?: string
  title?: string
  tenantCode?: string
  tenantName?: string
  appName?: string
  logoUrl?: string
  loginBackgroundUrl?: string
  multiTenant?: boolean
}

export interface AuthUserPayload extends ApiRecord {
  id?: string | number
  username?: string
  fullName?: string
  email?: string
  role?: string
  tenantId?: string | number
  tenant?: {
    id?: string | number
    code?: string
    name?: string
    logoUrl?: string
  }
}

export type AuthChangePasswordPayload = ApiRecord

export interface AuthCaptchaPayload extends ApiRecord {
  challengeId?: string
  trackWidth?: number
  thumbWidth?: number
  expireSeconds?: number
}

/**
 * 认证相关 API
 */
export const authAPI = {
  /**
   * 用户登录
   * @param {object} credentials - 登录凭证
   * @param {string} credentials.username - 用户名
   * @param {string} credentials.password - 密码
   * @param {string} credentials.sliderChallengeId - 滑块挑战标识
   * @param {number} credentials.sliderOffset - 滑块停留位置
   * @param {string} credentials.tenantCode - 租户代码（多租户模式下可选）
   * @returns {Promise} 登录结果
   */
  login(credentials: LoginCredentials) {
    return request.post<ApiResponse<AuthLoginPayload>>('/auth/login', credentials)
  },

  /**
   * 获取登录滑块挑战。仅在同一登录上下文已触发人机验证时可用。
   * @returns {Promise} 滑块挑战数据 (challengeId, trackWidth, thumbWidth, expireSeconds)
   */
  getCaptcha(username: string, tenantCode?: string) {
    return request.get<ApiResponse<AuthCaptchaPayload>>('/auth/captcha', {
      params: { username, tenantCode: tenantCode || undefined },
    })
  },

  /**
   * 获取应用配置
   * @returns {Promise} 应用配置
   */
  getConfig(tenantCode?: string) {
    if (!tenantCode) {
      return request.get<ApiResponse<AuthConfigPayload>>('/auth/config')
    }
    return request.get<ApiResponse<AuthConfigPayload>>('/auth/config', {
      params: { tenantCode },
    })
  },

  /** 刷新同源 Cookie 会话。 */
  refreshToken() {
    return request.post<ApiResponse<ApiRecord>>('/auth/refresh')
  },

  /**
   * 用户登出
   * @returns {Promise} 登出结果
   */
  logout() {
    return request.post<ApiResponse<ApiRecord>>('/auth/logout')
  },

  /**
   * 获取当前用户信息
   * @returns {Promise} 用户信息
   */
  getCurrentUser(config?: AxiosRequestConfig & { skipAuthRedirect?: boolean }) {
    return request.get<ApiResponse<AuthUserPayload>>('/auth/me', config)
  },

  /**
   * 修改密码
   * @param {object} passwordData - 密码数据
   * @param {string} passwordData.oldPassword - 旧密码
   * @param {string} passwordData.newPassword - 新密码
   * @returns {Promise} 修改结果
   */
  changePassword(passwordData: PasswordData) {
    return request.put<ApiResponse<AuthChangePasswordPayload>>('/auth/password', passwordData)
  },
}
