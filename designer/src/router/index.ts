/**
 * 设计器路由配置。
 *
 * 正式入口只允许两种启动方式：
 * - 被 IDE 以 iframe 形式嵌入，并通过 bootstrap message 注入上下文；
 * - `/designer/debug` 独立调试路由，使用固定默认 UI（light + zh）。
 */

import type { NavigationGuardNext, RouteLocationNormalized, Router } from "vue-router";
import { createRouter, createWebHistory } from "vue-router";
import { STORAGE_KEYS } from "@/constants";
import type { EditorUiStore } from "@/stores/editor-ui-store";
import { getEditorUiStore } from "@/stores/editor-ui-store";
import { debugProjectApi } from "@/services/debugProjectApi";
import { isDesignerDebugRouteEnabled } from "@/runtime/debug-route";
import {
  buildIdeLoginUrl,
  buildIdeRestoreUrl,
  resolveDesignerIdeOriginFromRuntime,
  resolveDesignerEntrypointPlan,
  waitForHostBootstrap,
} from "@/runtime/host-bootstrap";
import { Storage } from "@/utils/storage";
import { shouldSyncEditorUiForPath } from "./runtime-settings";

declare module "vue-router" {
  interface RouteMeta {
    title?: string;
    requiresAuth?: boolean;
    project?: {
      id: string | null | undefined;
      tenantId: string | null | undefined;
    };
  }
}

type DesignerDebugProjectMeta = {
  id: string | null;
  tenantId: string | null;
};

type ResolveDefaultDebugProject = () => Promise<DesignerDebugProjectMeta | null>;

export function createDesignerRoutes(enableDebugRoute?: boolean) {
  const routes = [
    {
      path: "/",
      name: "Designer",
      component: () => import("@/ui/shell/DesignerView.vue"),
      meta: {
        title: "设计器",
        requiresAuth: true,
      },
    },
  ];

  if (isDesignerDebugRouteEnabled(enableDebugRoute)) {
    routes.push({
      path: "/debug",
      name: "DesignerDebug",
      component: () => import("@/ui/shell/DesignerView.vue"),
      meta: {
        title: "设计器调试",
        requiresAuth: false,
      },
    });
  }

  routes.push({
    path: "/preview",
    name: "Preview",
    component: () => import("@/ui/editors/page/preview/PreviewView.vue"),
    meta: {
      title: "预览",
      requiresAuth: true,
    },
  });

  return routes;
}

const routes = createDesignerRoutes();

const router = createRouter({
  history: createWebHistory("/designer/"),
  routes,
});

function clearEditorUiThemeEffects(): void {
  document.documentElement.classList.remove("dark");
  document.documentElement.removeAttribute("data-theme");
}

type RuntimeRouteEffectsDependencies = {
  editorUi: EditorUiStore;
};

type DesignerBeforeEachGuardDependencies = {
  getCurrentUrl?: () => string;
  getIdeOrigin?: () => string;
  isTopLevelWindow?: () => boolean;
  navigateToUrl?: (url: string) => void;
  resolveDefaultDebugProject?: ResolveDefaultDebugProject;
  waitForBootstrap?: () => Promise<boolean>;
};

function asNonEmptyString(value: unknown): string | null {
  return typeof value === "string" && value.trim().length > 0 ? value.trim() : null;
}

async function resolveDesignerDebugProjectMeta(
  targetUrl: string,
  resolveDefaultDebugProject: ResolveDefaultDebugProject,
): Promise<DesignerDebugProjectMeta> {
  const url = new URL(targetUrl);
  const projectIdFromUrl = asNonEmptyString(url.searchParams.get("pid")) ?? asNonEmptyString(url.searchParams.get("id"));
  const tenantIdFromUrl = asNonEmptyString(url.searchParams.get("tenant"));

  if (projectIdFromUrl) {
    Storage.setProjectId(projectIdFromUrl);
    if (tenantIdFromUrl) {
      Storage.setTenantId(tenantIdFromUrl);
    }

    return {
      id: projectIdFromUrl,
      tenantId: tenantIdFromUrl ?? Storage.getTenantId(),
    };
  }

  const defaultDebugProject = await resolveDefaultDebugProject();
  if (defaultDebugProject?.id) {
    Storage.setProjectId(defaultDebugProject.id);
    if (defaultDebugProject.tenantId) {
      Storage.setTenantId(defaultDebugProject.tenantId);
    }

    return {
      id: defaultDebugProject.id,
      tenantId: defaultDebugProject.tenantId ?? Storage.getTenantId(),
    };
  }

  return {
    id: Storage.getProjectId(),
    tenantId: Storage.getTenantId(),
  };
}

export function applyRuntimeRouteEffects(
  routePath: string,
  _currentUrl: string,
  { editorUi }: RuntimeRouteEffectsDependencies,
): void {
  if (!shouldSyncEditorUiForPath(routePath)) {
    clearEditorUiThemeEffects();
    return;
  }

  // 设计态路由只把当前 UI 状态重新同步到 DOM，避免路由切换时覆盖宿主已经下发的主题/语言。
  editorUi.applyThemeToDom();
}

function syncRuntimeSettings(routePath = window.location.pathname): void {
  applyRuntimeRouteEffects(routePath, window.location.href, {
    editorUi: getEditorUiStore(),
  });
}

/**
 * beforeEach 执行时浏览器地址仍可能停留在上一路由，这里显式用目标路由重建入口 URL。
 *
 * 只保留当前 URL 中仍需跨路由延续的 handoffId，避免把旧路由 pathname 误当成新入口模式。
 */
function resolveEntrypointUrlForRoute(
  currentUrl: string,
  targetRoute: RouteLocationNormalized,
  targetRouter: Router,
): string {
  const current = new URL(currentUrl);
  const resolvedTargetUrl = new URL(targetRouter.resolve(targetRoute).href, current);
  const currentHandoffId = current.searchParams.get("handoffId");

  if (currentHandoffId && !resolvedTargetUrl.searchParams.has("handoffId")) {
    resolvedTargetUrl.searchParams.set("handoffId", currentHandoffId);
  }

  return resolvedTargetUrl.toString();
}

export function registerDesignerBeforeEachGuard(
  targetRouter: Router,
  dependencies: DesignerBeforeEachGuardDependencies = {},
): () => void {
  const getCurrentUrl = dependencies.getCurrentUrl ?? (() => window.location.href);
  const getIdeOrigin = dependencies.getIdeOrigin ?? (() =>
    resolveDesignerIdeOriginFromRuntime({
      currentUrl: getCurrentUrl(),
      referrer: document.referrer,
    }));
  const isTopLevelWindow = dependencies.isTopLevelWindow ?? (() => window.parent === window);
  const navigateToUrl = dependencies.navigateToUrl ?? ((url: string) => window.location.replace(url));
  const resolveDefaultDebugProject = dependencies.resolveDefaultDebugProject ?? (() =>
    debugProjectApi.resolveDefaultProjectByName());
  const waitForBootstrap = dependencies.waitForBootstrap ?? waitForHostBootstrap;

  return targetRouter.beforeEach(async (to: RouteLocationNormalized, _from, next: NavigationGuardNext) => {
    document.title = `${to.meta.title || "设计器"} - InduForge`;

    const currentUrl = getCurrentUrl();
    const targetUrl = resolveEntrypointUrlForRoute(currentUrl, to, targetRouter);
    const entrypointPlan = resolveDesignerEntrypointPlan(targetUrl, {
      hasProjectId: Boolean(Storage.getProjectId()),
      hasToken: Boolean(Storage.getToken()),
      ideOrigin: getIdeOrigin(),
      isTopLevelWindow: isTopLevelWindow(),
      referrer: document.referrer,
    });

    if (entrypointPlan.shouldRedirectToIde && entrypointPlan.ideRedirectUrl) {
      next(false);
      navigateToUrl(entrypointPlan.ideRedirectUrl);
      return;
    }

    let token = Storage.getToken();
    let bootstrapSucceeded = true;

    if (entrypointPlan.shouldWaitForBootstrap) {
      /**
       * handoff 代表宿主要求切换到新的工程上下文。
       * 等待前先清掉旧工程标识，避免 bootstrap 超时后仍把上一工程误当成当前入口恢复。
       */
      if (entrypointPlan.handoffId) {
        Storage.remove(STORAGE_KEYS.PROJECT_ID);
        Storage.remove(STORAGE_KEYS.TENANT_ID);
      }

      // bootstrap 失败时不再挂起，后续继续落到现有登录或 IDE 回跳兜底。
      bootstrapSucceeded = await waitForBootstrap();
      token = Storage.getToken();
    }

    syncRuntimeSettings(to.path);

    if (entrypointPlan.isDebugRoute) {
      to.meta.project = await resolveDesignerDebugProjectMeta(targetUrl, resolveDefaultDebugProject);
    }

    if (!token && to.meta.requiresAuth) {
      next(false);
      navigateToUrl(
        buildIdeLoginUrl({
          currentUrl: targetUrl,
          ideOrigin: getIdeOrigin(),
        }),
      );
      return;
    }

    if (!bootstrapSucceeded) {
      // 无 token 时已在上面的登录分支收敛；保留这里是为了显式表达：
      // 失败只负责解除等待，不改变既有 token/projectId 分支语义。
    }

    const projectId = Storage.getProjectId();
    const tenantId = Storage.getTenantId();

    if (!projectId && to.name === "Designer") {
      next(false);
      navigateToUrl(
        buildIdeRestoreUrl(entrypointPlan.handoffId, getIdeOrigin()),
      );
      return;
    }

    to.meta.project = {
      id: projectId,
      tenantId,
    };

    next();
  });
}

registerDesignerBeforeEachGuard(router);

export default router;
