const EMBEDDED_APP_TAB_META = {
  designer: {
    keyPrefix: 'design-center',
    titleKey: 'projectManagement.designCenter',
    icon: 'design',
  },
  datacenter: {
    keyPrefix: 'data-center',
    titleKey: 'projectManagement.dataCenter',
    icon: 'database',
  },
}

export const EMBEDDED_APP_COMPONENT = 'EMBEDDED_APP'

type QueryValue = string | string[] | null | undefined
type DashboardQuery = Record<string, QueryValue>
type EmbeddedAppType = keyof typeof EMBEDDED_APP_TAB_META

interface RestoredTabPayload {
  appType?: EmbeddedAppType | null
  projectId?: string | number | null
  pid?: string | number | null
  tenantId?: string | number | null
  projectName?: string | null
}

/**
 * 规范化查询参数值。
 * Vue Router 可能把同名参数解析成数组，这里统一只取第一个有效值。
 * @param {string|string[]|null|undefined} value - 查询参数原值
 * @returns {string|null} 单值字符串
 */
const normalizeQueryValue = (value: QueryValue): string | null => {
  if (Array.isArray(value)) {
    return normalizeQueryValue(value[0])
  }

  return typeof value === 'string' && value ? value : null
}

/**
 * 构建根路径跳转到仪表盘时的目标位置。
 * 这里必须保留查询参数，避免 handoffId 在 "/" -> "/dashboard" 时被静态 redirect 吞掉。
 * @param {object} to - Vue Router 的目标路由对象
 * @returns {{path: string, query: object}} 重定向位置
 */
export const buildDashboardRedirectLocation = (to: { query?: DashboardQuery } = {}): {
  path: string
  query: DashboardQuery
} => ({
  path: '/dashboard',
  query: { ...(to?.query || {}) },
})

/**
 * 从当前查询参数中提取 handoffId。
 * @param {object} query - 当前路由查询参数
 * @returns {string|null} handoffId
 */
export const extractDashboardHandoffId = (query: DashboardQuery = {}): string | null =>
  normalizeQueryValue(query?.handoffId)

/**
 * 删除已经消费过的 handoffId，避免刷新或重复导航时再次触发恢复。
 * @param {object} query - 当前路由查询参数
 * @returns {object} 去掉 handoffId 后的新查询对象
 */
export const stripDashboardHandoffQuery = (query: DashboardQuery = {}): DashboardQuery => {
  const nextQuery = { ...(query || {}) }
  delete nextQuery.handoffId
  return nextQuery
}

/**
 * 把 handoff 恢复载荷映射成 Dashboard 可直接打开的嵌入标签配置。
 * 这里只负责生成稳定的 tab key、标题键和工程上下文，不直接依赖 Vue 组件实例，便于测试。
 * @param {object} payload - handoff 恢复载荷
 * @returns {object|null} 可打开的标签配置
 */
export const createRestoredEmbeddedTab = (payload: RestoredTabPayload = {}) => {
  if (payload?.appType !== 'designer' && payload?.appType !== 'datacenter') {
    return null
  }
  const appMeta = EMBEDDED_APP_TAB_META[payload.appType]
  const projectId = payload?.projectId ?? payload?.pid ?? null

  if (!appMeta || projectId === null || projectId === undefined || projectId === '') {
    return null
  }

  const project: {
    id: string | number
    tenantId?: string | number | null
    name?: string
  } = {
    id: projectId,
  }

  if (payload?.tenantId !== null && payload?.tenantId !== undefined && payload?.tenantId !== '') {
    project.tenantId = payload.tenantId
  }

  if (typeof payload?.projectName === 'string' && payload.projectName) {
    project.name = payload.projectName
  }

  const tab: {
    key: string
    titleKey: string
    component: string
    props: {
      appType: EmbeddedAppType
      project: {
        id: string | number
        tenantId?: string | number | null
        name?: string
      }
    }
    icon: string
    titlePrefix?: string
  } = {
    key: `${appMeta.keyPrefix}-${projectId}`,
    titleKey: appMeta.titleKey,
    component: EMBEDDED_APP_COMPONENT,
    props: {
      appType: payload.appType,
      project,
    },
    icon: appMeta.icon,
  }

  if (project.name) {
    tab.titlePrefix = project.name
  }

  return tab
}
