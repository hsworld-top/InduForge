/* global __DATACENTER_DEBUG_ROUTE_ENABLED__ */

// v2 模块 ID 常量
export const MODULE_DATAPOINT = 'datapoint' as const
export const MODULE_ACCESS_SOURCE = 'access-source' as const
export const MODULE_INDUSTRIAL_COLLECTOR = 'industrial-collector' as const
export const MODULE_HISTORY_STORAGE = 'history-storage' as const
export const MODULE_COMPUTE = 'compute' as const
export const MODULE_ALARM = 'alarm' as const

export type V2ModuleId =
  | typeof MODULE_DATAPOINT
  | typeof MODULE_ACCESS_SOURCE
  | typeof MODULE_INDUSTRIAL_COLLECTOR
  | typeof MODULE_HISTORY_STORAGE
  | typeof MODULE_COMPUTE
  | typeof MODULE_ALARM

/** v2 模块 ID 列表，用于路由正则 */
export const V2_MODULE_IDS: readonly V2ModuleId[] = [
  MODULE_DATAPOINT,
  MODULE_ACCESS_SOURCE,
  MODULE_INDUSTRIAL_COLLECTOR,
  MODULE_HISTORY_STORAGE,
  MODULE_COMPUTE,
  MODULE_ALARM,
] as const

/** 默认模块 */
export const DEFAULT_MODULE: V2ModuleId = MODULE_DATAPOINT

const DEFAULT_DEBUG_ROUTE_ENABLED =
  typeof __DATACENTER_DEBUG_ROUTE_ENABLED__ !== 'undefined'
    ? __DATACENTER_DEBUG_ROUTE_ENABLED__
    : true

/**
 * 数据中心路由配置工厂。
 *
 * v2 路由结构：
 *   /datacenter/:module(datapoint|access-source|industrial-collector|history-storage|compute|alarm)/:objectId?/:tab?
 *   /datacenter/debug/:module(...)/:objectId?/:tab?
 *
 */
export function createDatacenterRoutes({
  DataCenterComponent,
  enableDebugRoute = DEFAULT_DEBUG_ROUTE_ENABLED,
}: {
  DataCenterComponent: unknown
  enableDebugRoute?: boolean
}) {
  // 正式入口路由（需要认证）
  const routes: unknown[] = [
    // 根路径重定向到默认模块
    {
      path: '/',
      redirect: (to: { query?: Record<string, unknown> }) => ({
        path: `/${DEFAULT_MODULE}`,
        query: to.query,
      }),
    },
    // v2 正式模块路由
    {
      path: '/:module(datapoint|access-source|industrial-collector|history-storage|compute|alarm)/:objectId?/:tab?',
      name: 'datacenter',
      component: DataCenterComponent,
      meta: {
        titleKey: 'route.datacenter',
        requiresAuth: true,
      },
    },
  ]

  if (enableDebugRoute) {
    routes.push(
      // debug 根路径重定向到默认模块
      {
        path: '/debug',
        redirect: { path: `/debug/${DEFAULT_MODULE}` },
      },
      // v2 debug 模块路由
      {
        path: '/debug/:module(datapoint|access-source|industrial-collector|history-storage|compute|alarm)/:objectId?/:tab?',
        name: 'datacenter-debug',
        component: DataCenterComponent,
        meta: {
          titleKey: 'route.datacenterDebug',
          requiresAuth: false,
        },
      },
    )
  }

  return routes
}
