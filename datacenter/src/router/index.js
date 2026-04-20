import { createRouter, createWebHistory } from "vue-router";
import { Storage } from "@/utils/storage";
// import DataCenter from "../views/DataCenter.vue"; // 原版本
import DataCenter from "../views/DataCenterNew.vue"; // 重构版本
import {
  applyThemeToDocument,
  buildIdeLoginUrl,
  buildIdeRestoreUrl,
  resolveIdeOriginFromRuntime,
  shouldRedirectTopLevelToIde,
  shouldUseDebugMode,
  waitForHostBootstrap,
} from "../runtime/host-bootstrap.js";

const routes = [
  {
    path: "/",
    name: "datacenter",
    component: DataCenter,
    meta: {
      title: "数据中心",
      requiresAuth: true,
    },
  },
  {
    path: "/debug",
    name: "datacenter-debug",
    component: DataCenter,
    meta: {
      title: "数据中心调试",
      requiresAuth: false,
    },
  },
];

const router = createRouter({
  history: createWebHistory("/datacenter/"),
  routes,
});

/**
 * 根据目标路由拼出当前浏览器里实际会出现的 pathname。
 * 这样即便当前 href 仍停留在正式入口，程序化跳到 `/debug` 时也不会被旧 pathname 误判。
 * @param {string} routePath - Vue Router 路由 path
 * @returns {string} 浏览器 pathname
 */
const resolveRoutePathname = (routePath) => {
  if (!routePath || routePath === "/") {
    return "/datacenter/";
  }

  return `/datacenter${routePath}`.replace(/\/{2,}/g, "/");
};

/**
 * 将当前 URL 与目标路由合并，得到本次守卫应当依据的入口地址。
 * @param {string} currentUrl - 当前浏览器地址
 * @param {string} routePath - 目标路由 path
 * @returns {URL} 目标入口 URL
 */
const buildRouteRuntimeUrl = (currentUrl, routePath) => {
  const nextUrl = new URL(currentUrl);
  nextUrl.pathname = resolveRoutePathname(routePath);
  return nextUrl;
};

export function registerDatacenterBeforeEachGuard(
  targetRouter,
  {
    getCurrentUrl = () => window.location.href,
    getIdeOrigin = (currentUrl) =>
      resolveIdeOriginFromRuntime({
        currentUrl,
        referrer: document.referrer,
      }),
    isTopLevelWindow = () => window.parent === window,
    navigateToUrl = (url) => {
      window.location.href = url;
    },
    waitForBootstrap = waitForHostBootstrap,
  } = {},
) {
  return targetRouter.beforeEach(async (to, from, next) => {
    document.title = `${to.meta.title || "数据中心"} - ProjectIDE`;
    applyThemeToDocument(Storage.getTheme());

    const runtimeUrl = buildRouteRuntimeUrl(getCurrentUrl(), to.path);
    const ideOrigin = getIdeOrigin(runtimeUrl.toString());
    const handoff = runtimeUrl.searchParams.get("handoff");
    const isDebugRoute =
      shouldUseDebugMode(runtimeUrl.pathname) || to.meta.requiresAuth === false;

    if (shouldRedirectTopLevelToIde(runtimeUrl.pathname, isTopLevelWindow())) {
      navigateToUrl(buildIdeRestoreUrl(handoff, ideOrigin));
      return;
    }

    if (isDebugRoute) {
      next();
      return;
    }

    let token = Storage.getToken();
    if (!token) {
      const bootstrapReady = await waitForBootstrap();
      token = Storage.getToken();

      if (!bootstrapReady || !token) {
        navigateToUrl(buildIdeLoginUrl(runtimeUrl.toString(), ideOrigin));
        return;
      }
    }

    const projectId = Storage.getProjectId();
    const tenantId = Storage.getTenantId();

    if (!projectId) {
      navigateToUrl(buildIdeRestoreUrl(handoff, ideOrigin));
      return;
    }

    to.meta.project = {
      id: projectId,
      tenantId,
    };

    next();
  });
}

registerDatacenterBeforeEachGuard(router);

export default router;
