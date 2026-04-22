import request from '@/utils/request'
import type { ApiResponse, ApiRecord } from '@/types/api'

type LoginCredentials = {
  username: string
  password: string
  captchaKey?: string
  captchaCode?: string
  tenantCode?: string
}

type PasswordData = {
  oldPassword: string
  newPassword: string
}

export interface AuthTokenPayload {
  token?: string
  accessToken?: string
  refreshToken?: string
  expiresIn?: number
}

export interface AuthConfigPayload extends ApiRecord {
  tenantCode?: string
  tenantName?: string
  appName?: string
  logoUrl?: string
  loginBackgroundUrl?: string
}

export interface AuthUserPayload extends ApiRecord {
  id?: string | number
  username?: string
  fullName?: string
  email?: string
  role?: string
  tenantId?: string | number
}

export interface AuthChangePasswordPayload extends ApiRecord {
  success?: boolean
}

export interface AuthCaptchaPayload extends ApiRecord {
  key?: string
  image?: string
  expireSeconds?: number
}

type ApiResult<T> = Promise<ApiResponse<T>>
const asApiResult = <T>(promise: unknown): ApiResult<T> => promise as ApiResult<T>

/**
 * 认证相关 API
 */
export const authAPI = {
  /**
   * 用户登录
   * @param {object} credentials - 登录凭证
   * @param {string} credentials.username - 用户名
   * @param {string} credentials.password - 密码
   * @param {string} credentials.captchaKey - 验证码Key
   * @param {string} credentials.captchaCode - 验证码内容
   * @param {string} credentials.tenantCode - 租户代码（多租户模式下可选）
   * @returns {Promise} 登录结果
   */
  login(credentials: LoginCredentials) {
    const { captchaKey, captchaCode, ...others } = credentials
    return asApiResult<AuthTokenPayload>(request.post<ApiResponse<AuthTokenPayload>>('/auth/login', {
      ...others,
      captchaKey,
      captchaCode,
    }))
  },

  /**
   * 获取验证码
   * @returns {Promise} 验证码数据 (key, image, expireSeconds)
   */
  getCaptcha() {
    return asApiResult<AuthCaptchaPayload>(request.get<ApiResponse<AuthCaptchaPayload>>('/auth/captcha'))
  },

  /**
   * 获取应用配置
   * @returns {Promise} 应用配置
   */
  getConfig(tenantCode?: string) {
    if (!tenantCode) {
      return asApiResult<AuthConfigPayload>(request.get<ApiResponse<AuthConfigPayload>>('/auth/config'))
    }
    return asApiResult<AuthConfigPayload>(request.get<ApiResponse<AuthConfigPayload>>('/auth/config', {
      params: { tenantCode },
    }))
  },

  /**
   * 刷新访问令牌
   * @param {string} refreshToken - 刷新令牌
   * @returns {Promise} 新的访问令牌
   */
  refreshToken(refreshToken: string) {
    return asApiResult<AuthTokenPayload>(request.post<ApiResponse<AuthTokenPayload>>('/auth/refresh', {
      refreshToken,
    }))
  },

  /**
   * 用户登出
   * @returns {Promise} 登出结果
   */
  logout() {
    return asApiResult<ApiRecord>(request.post<ApiResponse<ApiRecord>>('/auth/logout'))
  },

  /**
   * 获取当前用户信息
   * @returns {Promise} 用户信息
   */
  getCurrentUser() {
    return asApiResult<AuthUserPayload>(request.get<ApiResponse<AuthUserPayload>>('/auth/me'))
  },

  /**
   * 修改密码
   * @param {object} passwordData - 密码数据
   * @param {string} passwordData.oldPassword - 旧密码
   * @param {string} passwordData.newPassword - 新密码
   * @returns {Promise} 修改结果
   */
  changePassword(passwordData: PasswordData) {
    return asApiResult<AuthChangePasswordPayload>(request.put<ApiResponse<AuthChangePasswordPayload>>(
      '/auth/password',
      passwordData
    ))
  },
}
