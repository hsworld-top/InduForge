/**
 * 嵌入式应用类型。
 */
export type EmbeddedAppType = 'designer' | 'datacenter'

/**
 * 嵌入应用关联的工程上下文。
 */
export interface EmbeddedProjectContext {
  id: string
  tenantId?: string
  name?: string
}

/**
 * Dashboard 嵌入标签页 props。
 */
export interface EmbeddedTabProps {
  appType: EmbeddedAppType
  project: EmbeddedProjectContext
}

/**
 * bootstrap 响应载荷。
 */
export interface EmbeddedBootstrapPayload {
  handoffId: string
  appType: EmbeddedAppType
  projectId: string
  tenantId?: string
  token?: string
  refreshToken?: string
  theme?: string
  locale?: string
}
