import request from '@/utils/request'

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
    return request.post('/auth/login', {
      ...others,
      captchaKey,
      captchaCode,
    })
  },

  /**
   * 获取验证码
   * @returns {Promise} 验证码数据 (key, image, expireSeconds)
   */
  getCaptcha() {
    return request.get('/auth/captcha')
  },

  /**
   * 获取应用配置
   * @returns {Promise} 应用配置
   */
  getConfig(tenantCode?: string) {
    return request.get('/auth/config', {
      params: tenantCode ? { tenantCode } : undefined,
    })
  },

  /**
   * 刷新访问令牌
   * @param {string} refreshToken - 刷新令牌
   * @returns {Promise} 新的访问令牌
   */
  refreshToken(refreshToken: string) {
    return request.post('/auth/refresh', { refreshToken })
  },

  /**
   * 用户登出
   * @returns {Promise} 登出结果
   */
  logout() {
    return request.post('/auth/logout')
  },

  /**
   * 获取当前用户信息
   * @returns {Promise} 用户信息
   */
  getCurrentUser() {
    return request.get('/auth/me')
  },

  /**
   * 修改密码
   * @param {object} passwordData - 密码数据
   * @param {string} passwordData.oldPassword - 旧密码
   * @param {string} passwordData.newPassword - 新密码
   * @returns {Promise} 修改结果
   */
  changePassword(passwordData: PasswordData) {
    return request.put('/auth/password', passwordData)
  },
}
