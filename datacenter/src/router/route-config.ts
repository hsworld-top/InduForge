// @ts-nocheck
const DEFAULT_DEBUG_ROUTE_ENABLED =
  typeof __DATACENTER_DEBUG_ROUTE_ENABLED__ !== "undefined"
    ? __DATACENTER_DEBUG_ROUTE_ENABLED__
    : true;

/**
 * 数据中心路由配置工厂。
 *
 * 生产构建会显式裁掉 `/debug`，避免发布包继续暴露独立调试入口。
 */
export function createDatacenterRoutes({
  DataCenterComponent,
  enableDebugRoute = DEFAULT_DEBUG_ROUTE_ENABLED,
}) {
  const routes = [
    {
      path: "/",
      name: "datacenter",
      component: DataCenterComponent,
      meta: {
        titleKey: "route.datacenter",
        requiresAuth: true,
      },
    },
  ];

  if (enableDebugRoute) {
    routes.push({
      path: "/debug",
      name: "datacenter-debug",
      component: DataCenterComponent,
      meta: {
        titleKey: "route.datacenterDebug",
        requiresAuth: false,
      },
    });
  }

  return routes;
}
