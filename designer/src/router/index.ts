/**
 * 设计器路由配置
 *
 * 路由说明：
 * - / : 设计器主界面（DesignerView）
 * - /preview : 预览界面（PreviewView）
 */

import type { NavigationGuardNext, RouteLocationNormalized, Router } from "vue-router";
import { createRouter, createWebHistory } from "vue-router";
import type { EditorUiStore } from "@/stores/editor-ui-store";
import { getEditorUiStore } from "@/stores/editor-ui-store";
import { Storage } from "@/utils/storage";
import { resolveRuntimeRouteSyncPlan } from "./runtime-settings";

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
  {
    path: "/preview",
    name: "Preview",
    component: () => import("@/ui/editors/page/preview/PreviewView.vue"),
    meta: {
      title: "预览",
      requiresAuth: true,
    },
  },
];

const router = createRouter({
  history: createWebHistory("/designer/"),
  routes,
});

function applyTheme(theme: string): void {
  document.documentElement.classList.toggle("dark", theme === "dark");
  document.documentElement.setAttribute("data-theme", theme);
}

function clearEditorUiThemeEffects(): void {
  document.documentElement.classList.remove("dark");
  document.documentElement.removeAttribute("data-theme");
}

type RuntimeRouteEffectsDependencies = {
  editorUi: EditorUiStore;
  replaceState: (data: unknown, unused: string, url?: string | URL | null) => void;
};

export function applyRuntimeRouteEffects(
  routePath: string,
  currentUrl: string,
  { editorUi, replaceState }: RuntimeRouteEffectsDependencies,
): void {
  const plan = resolveRuntimeRouteSyncPlan(new URL(currentUrl), routePath, {
    theme: Storage.getDesignerTheme(),
    locale: Storage.getDesignerLanguage(),
  });

  if (plan.tokenFromUrl) {
    Storage.setToken(plan.tokenFromUrl);
  }

  if (plan.refreshTokenFromUrl) {
    Storage.setRefreshToken(plan.refreshTokenFromUrl);
  }

  if (plan.projectIdFromUrl) {
    Storage.setProjectId(plan.projectIdFromUrl);
  }

  if (plan.tenantIdFromUrl) {
    Storage.setTenantId(plan.tenantIdFromUrl);
  }

  if (!plan.shouldSyncEditorUi) {
    clearEditorUiThemeEffects();
  } else {
    editorUi.initFromRuntime(plan.runtimeSettings);
    applyTheme(plan.runtimeSettings.theme);
  }

  if (plan.shouldReplaceUrl) {
    replaceState({}, "", plan.cleanedUrl);
  }
}

function syncRuntimeSettings(routePath = window.location.pathname): void {
  applyRuntimeRouteEffects(routePath, window.location.href, {
    editorUi: getEditorUiStore(),
    replaceState: window.history.replaceState.bind(window.history),
  });
}

export function registerDesignerBeforeEachGuard(targetRouter: Router): () => void {
  return targetRouter.beforeEach(async (to: RouteLocationNormalized, _from, next: NavigationGuardNext) => {
    document.title = `${to.meta.title || "设计器"} - InduForge`;

    syncRuntimeSettings(to.path);

    const isDev = import.meta.env.DEV;
    const devHost = import.meta.env.VITE_DEV_HOST || "localhost";
    const idePort = import.meta.env.VITE_IDE_PORT || 18601;
    const ideOrigin = isDev ? `http://${devHost}:${idePort}` : "";

    const token = Storage.getToken();
    if (!token && to.meta.requiresAuth) {
      const redirectUrl = encodeURIComponent(window.location.href);
      window.location.href = `${ideOrigin}/login?redirect=${redirectUrl}`;
      return;
    }

    const urlParams = new URLSearchParams(window.location.search);
    const projectId = urlParams.get("pid") || urlParams.get("id") || Storage.getProjectId();
    const tenantId = urlParams.get("tenant") || Storage.getTenantId();

    if (!projectId && to.name === "Designer") {
      window.location.href = `${ideOrigin}/`;
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
