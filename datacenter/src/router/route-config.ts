/* global __DATACENTER_DEBUG_ROUTE_ENABLED__ */

// v2 模块 ID 常量
export const MODULE_DATAPOINT = "datapoint" as const;
export const MODULE_ACCESS_SOURCE = "access-source" as const;
export const MODULE_COMPUTE = "compute" as const;
export const MODULE_ALARM = "alarm" as const;

export type V2ModuleId =
  | typeof MODULE_DATAPOINT
  | typeof MODULE_ACCESS_SOURCE
  | typeof MODULE_COMPUTE
  | typeof MODULE_ALARM;

/** v2 模块 ID 列表，用于路由正则 */
export const V2_MODULE_IDS: readonly V2ModuleId[] = [
  MODULE_DATAPOINT,
  MODULE_ACCESS_SOURCE,
  MODULE_COMPUTE,
  MODULE_ALARM,
] as const;

/**
 * 旧模块 ID → v2 模块 ID 映射表（用于向后兼容 redirect）。
 * 旧路由访问时自动 redirect 到新 ID，避免旧链接白屏。
 */
export const LEGACY_MODULE_ID_MAP: Record<string, V2ModuleId> = {
  datapoints: MODULE_DATAPOINT,
  "access-sources": MODULE_ACCESS_SOURCE,
  "compute-units": MODULE_COMPUTE,
  "alarm-units": MODULE_ALARM,
};

/** 默认模块 */
export const DEFAULT_MODULE: V2ModuleId = MODULE_DATAPOINT;

const DEFAULT_DEBUG_ROUTE_ENABLED =
  typeof __DATACENTER_DEBUG_ROUTE_ENABLED__ !== "undefined"
    ? __DATACENTER_DEBUG_ROUTE_ENABLED__
    : true;

/**
 * 数据中心路由配置工厂。
 *
 * v2 路由结构：
 *   /datacenter/:module(datapoint|access-source|compute|alarm)/:objectId?/:tab?
 *   /datacenter/debug/:module(...)/:objectId?/:tab?
 *
 * 旧模块 ID 通过 redirect 兼容：
 *   /datacenter/:legacyModule(datapoints|access-sources|compute-units|alarm-units)
 *   → /datacenter/:newModule
 */
export function createDatacenterRoutes({
  DataCenterComponent,
  enableDebugRoute = DEFAULT_DEBUG_ROUTE_ENABLED,
}: {
  DataCenterComponent: unknown;
  enableDebugRoute?: boolean;
}) {
  // 正式入口路由（需要认证）
  const routes: unknown[] = [
    // 根路径重定向到默认模块
    {
      path: "/",
      redirect: { path: `/${DEFAULT_MODULE}` },
    },
    // v2 正式模块路由
    {
      path: "/:module(datapoint|access-source|compute|alarm)/:objectId?/:tab?",
      name: "datacenter",
      component: DataCenterComponent,
      meta: {
        titleKey: "route.datacenter",
        requiresAuth: true,
      },
    },
    // 旧模块 ID 兼容 redirect（旧链接不白屏）
    {
      path: "/:legacyModule(datapoints|access-sources|compute-units|alarm-units)/:rest(.*)?",
      redirect: (to: { params: Record<string, string> }) => {
        const legacyId = to.params.legacyModule as string;
        const newId = LEGACY_MODULE_ID_MAP[legacyId] ?? DEFAULT_MODULE;
        const rest = to.params.rest ? `/${to.params.rest}` : "";
        return { path: `/${newId}${rest}` };
      },
    },
  ];

  if (enableDebugRoute) {
    routes.push(
      // debug 根路径重定向到默认模块
      {
        path: "/debug",
        redirect: { path: `/debug/${DEFAULT_MODULE}` },
      },
      // v2 debug 模块路由
      {
        path: "/debug/:module(datapoint|access-source|compute|alarm)/:objectId?/:tab?",
        name: "datacenter-debug",
        component: DataCenterComponent,
        meta: {
          titleKey: "route.datacenterDebug",
          requiresAuth: false,
        },
      },
      // debug 旧模块 ID 兼容 redirect
      {
        path: "/debug/:legacyModule(datapoints|access-sources|compute-units|alarm-units)/:rest(.*)?",
        redirect: (to: { params: Record<string, string> }) => {
          const legacyId = to.params.legacyModule as string;
          const newId = LEGACY_MODULE_ID_MAP[legacyId] ?? DEFAULT_MODULE;
          const rest = to.params.rest ? `/${to.params.rest}` : "";
          return { path: `/debug/${newId}${rest}` };
        },
      },
    );
  }

  return routes;
}
