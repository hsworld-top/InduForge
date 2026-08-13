/**
 * Wujie 子应用类型。
 */
export type MicroAppType = 'designer' | 'datacenter'

/**
 * Wujie 子应用关联的工程上下文。
 */
export interface MicroAppProjectContext {
  id: string
  tenantId?: string
  name?: string
}

/**
 * Dashboard Wujie 标签页 props。
 */
export interface MicroAppTabProps {
  appType: MicroAppType
  project: MicroAppProjectContext
}
