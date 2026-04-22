import { Storage } from './storage'
import { createBootstrapResponse } from './embeddedAppBridge'

type SupportedAppType = 'designer' | 'datacenter'

interface ProjectLike {
  id?: string | number | null
  projectId?: string | number | null
  tenantId?: string | number | null
}

/**
 * 解析当前宿主页面的真实 origin。
 * handoff URL 允许保持相对路径，但注册链路必须显式带上真实 origin，避免后续回落到错误默认值。
 * @returns {string|null} 当前页面 origin
 */
const resolveCurrentOrigin = (): string | null => {
  const locationLike = globalThis.window?.location ?? globalThis.location
  return typeof locationLike?.origin === 'string' && locationLike.origin ? locationLike.origin : null
}

/**
 * 构建设计中心或数据中心入口上下文。
 * URL 继续保持 handoff 驱动的正式入口，origin 单独返回给注册链路使用。
 * @param {string} appType - 应用类型（designer/datacenter）
 * @param {object} project - 工程信息
 * @returns {{url: string, origin: string|null}} 嵌入入口上下文
 */
export const buildAppEntry = (
  appType: SupportedAppType,
  project: ProjectLike = {}
): { url: string; origin: string | null } => {
  const response = createBootstrapResponse(appType, {
    projectId: project?.id ?? project?.projectId ?? null,
    tenantId: project?.tenantId ?? Storage.getTenantId(),
    token: Storage.getToken(),
    refreshToken: Storage.getRefreshToken(),
    theme: Storage.getTheme(),
    locale: Storage.getLanguage(),
  })

  return {
    url: response.url,
    origin: resolveCurrentOrigin(),
  }
}

/**
 * 构建设计中心或数据中心访问地址。
 * 正式入口只输出 handoff URL，敏感上下文留给后续 bootstrap message 交换。
 * @param {string} appType - 应用类型（designer/datacenter）
 * @param {object} project - 工程信息
 * @returns {string} 访问地址
 */
export const buildAppUrl = (appType: SupportedAppType, project: ProjectLike = {}): string =>
  buildAppEntry(appType, project).url
