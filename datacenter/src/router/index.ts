// @ts-nocheck
import { createRouter, createWebHistory } from "vue-router";
import { watch } from "vue";
import { debugProjectAPI } from "@/api/debug-project.api";
import { STORAGE_KEYS } from "@/constants/index";
import { Storage } from "@/utils/storage";
import { datacenterLocale, getDatacenterRouteTitle } from "@/i18n/runtime";
// import DataCenter from "../views/DataCenter.vue"; // 原版本
import DataCenter from "../views/DataCenterNew.vue"; // 重构版本
import { resolveDatacenterDebugProjectMeta } from "./debug-project";
import { createDatacenterRoutes } from "./route-config";
import {
  applyThemeToDocument,
  buildIdeLoginUrl,
  buildIdeRestoreUrl,
  hasReusableTopLevelSession,
  resolveIdeOriginFromRuntime,
  shouldRedirectTopLevelToIde,
  shouldUseDebugMode,
  waitForHostBootstrap,
} from "../runtime/host-bootstrap";

const routes = createDatacenterRoutes({
  DataCenterComponent: DataCenter,
});

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
    resolveDefaultDebugProject = () =>
      debugProjectAPI.resolveDefaultProjectByName(),
    waitForBootstrap = waitForHostBootstrap,
  } = {},
) {
  return targetRouter.beforeEach(async (to, from, next) => {
    document.title = `${getDatacenterRouteTitle(to.name)} - ProjectIDE`;
    applyThemeToDocument(Storage.getTheme());

    const runtimeUrl = buildRouteRuntimeUrl(getCurrentUrl(), to.path);
    const ideOrigin = getIdeOrigin(runtimeUrl.toString());
    const handoff = runtimeUrl.searchParams.get("handoff");
    const isDebugRoute =
      shouldUseDebugMode(runtimeUrl.pathname) || to.meta.requiresAuth === false;

    if (
      shouldRedirectTopLevelToIde(
        runtimeUrl.pathname,
        isTopLevelWindow(),
        hasReusableTopLevelSession(),
      )
    ) {
      navigateToUrl(buildIdeRestoreUrl(handoff, ideOrigin));
      return;
    }

    if (isDebugRoute) {
      to.meta.project = await resolveDatacenterDebugProjectMeta({
        targetUrl: runtimeUrl.toString(),
        resolveDefaultDebugProject,
        getStoredProjectId: () => Storage.getProjectId(),
        getStoredTenantId: () => Storage.getTenantId(),
        setProjectId: (value) => Storage.setProjectId(value),
        setTenantId: (value) => Storage.setTenantId(value),
      });
      next();
      return;
    }

    let token = Storage.getToken();
    let bootstrapReady = true;
    if (handoff) {
      /**
       * handoff 表示宿主要求恢复新的工程上下文。
       * 等待 bootstrap 前先清掉旧工程与租户，避免超时时继续读到上一工程残留。
       */
      Storage.removeProjectId();
      Storage.remove(STORAGE_KEYS.TENANT_ID);
    }

    if (!token || handoff) {
      bootstrapReady = await waitForBootstrap();
      token = Storage.getToken();
    }

    if (!token) {
      navigateToUrl(buildIdeLoginUrl(runtimeUrl.toString(), ideOrigin));
      return;
    }

    if (!bootstrapReady) {
      // handoff 超时只负责解除等待；后续由 projectId 缺失分支回到 IDE 恢复，不再误导到登录页。
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

watch(
  datacenterLocale,
  () => {
    const currentRoute = router.currentRoute.value;
    document.title = `${getDatacenterRouteTitle(currentRoute.name)} - ProjectIDE`;
  },
  { immediate: true },
);

export default router;
