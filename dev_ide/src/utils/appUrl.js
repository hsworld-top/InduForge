import { Storage } from './storage.js'

/**
 * 构建设计中心或数据中心访问地址。
 * @param {string} appType - 应用类型（designer/datacenter）
 * @param {object} project - 工程信息
 * @returns {string} 访问地址
 */
export const buildAppUrl = (appType, project = {}) => {
  const params = new URLSearchParams()
  if (project?.id) params.set('pid', project.id)
  if (project?.tenantId) params.set('tenant', project.tenantId)

  const token = Storage.getToken()
  const refreshToken = Storage.getRefreshToken()
  const theme = Storage.getTheme()
  const locale = Storage.getLanguage()

  if (token) params.set('token', token)
  if (refreshToken) params.set('refreshToken', refreshToken)
  if (theme) params.set('theme', theme)

  if (appType === "designer") {
    if (locale) params.set('locale', locale)
    params.set('type', 'app')
    return `/designer/?${params.toString()}`
  }

  return `/datacenter/?${params.toString()}`
};
